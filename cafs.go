// Package cafs provides a content-addressed virtual filesystem with OCI backend.
//
// CAFS implements a Git-like content-addressed storage system that works with
// OCI registries for distributed storage. Think "Git's object model as a
// filesystem library with Docker-like namespace:ref addressing."
//
// Basic usage:
//
//	fs, err := cafs.Open("myorg/project:main")
//	if err != nil {
//		return err
//	}
//
//	// Standard filesystem operations
//	data, err := fs.ReadFile("/config.json")
//	err = fs.WriteFile("/output.txt", []byte("data"), 0644)
//
//	// Sync to remote
//	hash, err := fs.Push(context.Background())
package cafs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aweris/cafs/internal/remote"
	"github.com/aweris/cafs/internal/store"
)

// Open creates or opens a workspace at the given namespace:ref.
// Format: "organization/project:reference"
// Examples: "foo/bar:main", "team/cache:latest", "myapp/data:v1.0.0"
func Open(namespaceRef string, opts ...Option) (FS, error) {
	namespace, ref, err := parseNamespaceRef(namespaceRef)
	if err != nil {
		return nil, err
	}

	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	cacheDir := expandPath(options.CacheDir)
	localStore, err := store.NewLocalStore(cacheDir, namespace, options.CacheSize, options.CompressionLevel, options.CompressionEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	auth := options.Auth
	if auth == nil {
		auth = remote.NewDefaultAuthenticator()
	}
	ociRemote := remote.NewOCIRemote(options.Registry, namespace, ref, auth)

	ws := newWorkspace(namespace, ref, localStore, ociRemote)

	hash, err := localStore.GetRef(namespace, ref)
	localRefExists := err == nil && hash != ""

	if localRefExists {
		ws.baseHash = hash
		snapshot := newSnapshot(hash, localStore)
		rootNode, err := snapshot.loadNode(hash)
		if err == nil {
			ws.root = rootNode
			ws.dirty = make(map[string]struct{})
		}
	}

	shouldPull := (options.AutoPull == "always") || (options.AutoPull == "missing" && !localRefExists)

	if shouldPull && ociRemote != nil {
		if err := ws.Pull(context.Background()); err != nil {
			return nil, fmt.Errorf("auto-pull failed: %w", err)
		}
	}

	if len(options.Prefetch) > 0 {
		for _, path := range options.Prefetch {
			ws.prefetchPath(path)
		}
	}

	return ws, nil
}

// parseNamespaceRef parses "org/project:ref" format.
func parseNamespaceRef(s string) (namespace, ref string, err error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", "", ErrInvalidRef
	}
	namespace = parts[0]
	ref = parts[1]

	if namespace == "" || ref == "" {
		return "", "", ErrInvalidRef
	}

	return namespace, ref, nil
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// OpenWithOptions is an alias for Open with options.
func OpenWithOptions(namespaceRef string, opts *Options) (FS, error) {
	return Open(namespaceRef, WithOptions(opts))
}

// LoadSnapshot opens an immutable snapshot by hash from a specific namespace.
// Format: "organization/project:hash" or just "hash" (uses current workspace namespace)
func LoadSnapshot(namespaceOrHash string, opts ...Option) (*Snapshot, error) {
	var namespace, hash string

	// Parse format: "org/proj:hash" or just "hash"
	if strings.Contains(namespaceOrHash, ":") {
		parts := strings.Split(namespaceOrHash, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid snapshot format: %s", namespaceOrHash)
		}
		namespace = parts[0]
		hash = parts[1]
	} else {
		hash = namespaceOrHash
		return nil, fmt.Errorf("LoadSnapshot requires namespace:hash format (e.g., 'org/proj:%s')", hash)
	}

	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	cacheDir := expandPath(options.CacheDir)
	localStore, err := store.NewLocalStore(cacheDir, namespace, options.CacheSize, options.CompressionLevel, options.CompressionEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	return newSnapshot(hash, localStore), nil
}

// Version returns the CAFS library version.
func Version() string {
	return "0.1.0-dev"
}
