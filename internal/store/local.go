package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// LocalStore implements Store using the local filesystem.
//
// Storage layout (namespace-isolated):
//
//	basePath/namespace/
//	  objects/
//	    ab/cd123...  (content-addressed objects)
//	  refs/
//	    main  (plain text: "sha256:abc123...")
//
// Each namespace has its own isolated storage.
type LocalStore struct {
	basePath  string
	namespace string
	cache     Cache
}

func NewLocalStore(basePath, namespace string, cacheSize int) (*LocalStore, error) {
	nsPath := filepath.Join(basePath, namespace)

	objectsDir := filepath.Join(nsPath, "objects")
	refsDir := filepath.Join(nsPath, "refs")

	for _, dir := range []string{objectsDir, refsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &LocalStore{
		basePath:  nsPath,
		namespace: namespace,
		cache:     NewLRUCache(cacheSize),
	}, nil
}

// Get retrieves an object by hash.
func (s *LocalStore) Get(ctx context.Context, hash string) ([]byte, error) {
	if data, ok := s.cache.Get(hash); ok {
		return data, nil
	}

	path := s.objectPath(hash)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found: %s", hash)
		}
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	s.cache.Add(hash, data)
	return data, nil
}

// Put stores an object and returns its hash.
func (s *LocalStore) Put(ctx context.Context, data []byte) (string, error) {
	h := sha256.Sum256(data)
	hash := hex.EncodeToString(h[:])

	path := s.objectPath(hash)
	if _, err := os.Stat(path); err == nil {
		return hash, nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write object: %w", err)
	}

	s.cache.Add(hash, data)
	return hash, nil
}

// Has checks if an object exists.
func (s *LocalStore) Has(ctx context.Context, hash string) (bool, error) {
	// Check cache first
	if s.cache.Has(hash) {
		return true, nil
	}

	// Check filesystem
	path := s.objectPath(hash)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetMulti retrieves multiple objects.
func (s *LocalStore) GetMulti(ctx context.Context, hashes []string) (map[string][]byte, error) {
	// TODO: Implement parallel fetching
	result := make(map[string][]byte)
	for _, hash := range hashes {
		data, err := s.Get(ctx, hash)
		if err != nil {
			return nil, err
		}
		result[hash] = data
	}
	return result, nil
}

// PutMulti stores multiple objects.
func (s *LocalStore) PutMulti(ctx context.Context, objects map[string][]byte) error {
	// TODO: Implement parallel storage
	for _, data := range objects {
		if _, err := s.Put(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

// GetRef retrieves a reference.
func (s *LocalStore) GetRef(namespace, ref string) (string, error) {
	// Note: namespace parameter is for interface compatibility
	// This store is already namespace-isolated
	path := s.refPath(ref)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("ref not found: %s:%s", s.namespace, ref)
		}
		return "", err
	}
	return string(data), nil
}

// PutRef stores a reference.
func (s *LocalStore) PutRef(namespace, ref, hash string) error {
	// Note: namespace parameter is for interface compatibility
	// This store is already namespace-isolated
	path := s.refPath(ref)
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create ref directory: %w", err)
	}

	return os.WriteFile(path, []byte(hash), 0644)
}

// Evict removes an object from cache.
func (s *LocalStore) Evict(hash string) {
	s.cache.Remove(hash)
}

// Clear clears the cache.
func (s *LocalStore) Clear() {
	s.cache.Clear()
}

// Path returns the filesystem path for a given hash.
func (s *LocalStore) Path(hash string) string {
	return s.objectPath(hash)
}

func (s *LocalStore) objectPath(hash string) string {
	if len(hash) < 2 {
		return filepath.Join(s.basePath, "objects", hash)
	}
	return filepath.Join(s.basePath, "objects", hash[:2], hash[2:])
}

// refPath returns the filesystem path for a reference.
func (s *LocalStore) refPath(ref string) string {
	return filepath.Join(s.basePath, "refs", ref)
}
