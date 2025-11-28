package cafs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aweris/cafs/internal/remote"
	"github.com/aweris/cafs/internal/store"
)

// CAS is the main content-addressable storage implementation.
type CAS struct {
	objects   *store.LocalStore
	remote    *remote.OCIRemote
	index     sync.Map
	dirty     atomic.Bool
	namespace string
	ref       string
}

// Open creates or opens a store at the given namespace:ref.
// Format: "organization/project:reference"
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
	localStore, err := store.NewLocalStore(cacheDir, namespace, options.CacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	var ociRemote *remote.OCIRemote
	if options.Registry != "" {
		auth := options.Auth
		if auth == nil {
			auth = remote.NewDefaultAuthenticator()
		}
		ociRemote = remote.NewOCIRemote(options.Registry, namespace, ref, auth)
	}

	s := &CAS{
		objects:   localStore,
		remote:    ociRemote,
		namespace: namespace,
		ref:       ref,
	}

	// Load existing index if present
	if hash, err := localStore.GetRef(namespace, ref); err == nil && hash != "" {
		if data, err := localStore.Get(context.Background(), hash); err == nil {
			s.loadIndex(data)
		}
	}

	// Auto-pull if configured
	if options.AutoPull == "always" || options.AutoPull == "missing" {
		if ociRemote != nil {
			_ = s.Pull(context.Background())
		}
	}

	return s, nil
}

// Store saves data and returns its content hash and disk path.
func (s *CAS) Store(data []byte) (string, string, error) {
	hash, err := s.objects.PutBlob(context.Background(), data)
	if err != nil {
		return "", "", err
	}
	return hash, s.objects.Path(hash), nil
}

// Load retrieves data by its content hash.
func (s *CAS) Load(hash string) ([]byte, error) {
	return s.objects.Get(context.Background(), hash)
}

// Exists checks if content with given hash exists.
func (s *CAS) Exists(hash string) bool {
	ok, _ := s.objects.Has(context.Background(), hash)
	return ok
}

// Path returns the disk path for a given hash.
func (s *CAS) Path(hash string) string {
	return s.objects.Path(hash)
}

// Index associates a key with a content hash.
func (s *CAS) Index(key, hash string) {
	s.index.Store(key, hash)
	s.dirty.Store(true)
}

// Lookup returns the hash associated with a key.
func (s *CAS) Lookup(key string) (string, bool) {
	v, ok := s.index.Load(key)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// Indexed checks if a key exists in the index.
func (s *CAS) Indexed(key string) bool {
	_, ok := s.index.Load(key)
	return ok
}

// Push syncs local state to remote storage.
func (s *CAS) Push(ctx context.Context) (string, error) {
	if s.remote == nil {
		return "", ErrNoRemote
	}

	indexData, err := s.serializeIndex()
	if err != nil {
		return "", fmt.Errorf("failed to serialize index: %w", err)
	}

	indexHash, err := s.objects.PutBlob(ctx, indexData)
	if err != nil {
		return "", fmt.Errorf("failed to store index: %w", err)
	}

	objects := make(map[string][]byte)
	objects[indexHash] = indexData

	s.index.Range(func(k, v any) bool {
		hash := v.(string)
		if data, err := s.objects.Get(ctx, hash); err == nil {
			objects[hash] = data
		}
		return true
	})

	if err := s.remote.Push(ctx, indexHash, objects); err != nil {
		return "", fmt.Errorf("failed to push: %w", err)
	}

	if err := s.objects.PutRef(s.namespace, s.ref, indexHash); err != nil {
		return "", fmt.Errorf("failed to update ref: %w", err)
	}

	s.dirty.Store(false)
	return indexHash, nil
}

// Pull fetches state from remote storage.
func (s *CAS) Pull(ctx context.Context) error {
	if s.remote == nil {
		return ErrNoRemote
	}

	indexHash, objects, err := s.remote.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}

	for hash, data := range objects {
		if err := s.objects.PutWithHash(ctx, hash, data); err != nil {
			return fmt.Errorf("failed to store object %s: %w", hash, err)
		}
	}

	indexData, ok := objects[indexHash]
	if !ok {
		indexData, err = s.objects.Get(ctx, indexHash)
		if err != nil {
			return fmt.Errorf("failed to load index: %w", err)
		}
	}

	if err := s.loadIndex(indexData); err != nil {
		return fmt.Errorf("failed to parse index: %w", err)
	}

	if err := s.objects.PutRef(s.namespace, s.ref, indexHash); err != nil {
		return fmt.Errorf("failed to update ref: %w", err)
	}

	s.dirty.Store(false)
	return nil
}

func (s *CAS) serializeIndex() ([]byte, error) {
	m := make(map[string]string)
	s.index.Range(func(k, v any) bool {
		m[k.(string)] = v.(string)
		return true
	})
	return json.Marshal(m)
}

func (s *CAS) loadIndex(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		s.index.Store(k, v)
	}
	return nil
}

func parseNamespaceRef(s string) (namespace, ref string, err error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", ErrInvalidRef
	}
	return parts[0], parts[1], nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
