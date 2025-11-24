// Package store implements the local content storage layer.
//
// The Store interface provides a simple key-value abstraction for
// content-addressed objects. Based on Perkeep's simplicity:
// - Get/Put/Has for basic operations
// - Minimal metadata (just size)
// - Filesystem-based with LRU cache (no index initially)
package store

import "context"

// Store handles local content storage.
type Store interface {
	// Get retrieves an object by hash.
	Get(ctx context.Context, hash string) ([]byte, error)

	// Put stores an object and returns its hash.
	Put(ctx context.Context, data []byte) (hash string, err error)

	// Has checks if an object exists.
	Has(ctx context.Context, hash string) (bool, error)

	// GetMulti retrieves multiple objects (batch operation).
	GetMulti(ctx context.Context, hashes []string) (map[string][]byte, error)

	// PutMulti stores multiple objects (batch operation).
	PutMulti(ctx context.Context, objects map[string][]byte) error

	// GetRef retrieves a reference (namespace:ref → hash).
	GetRef(namespace, ref string) (string, error)

	// PutRef stores a reference.
	PutRef(namespace, ref, hash string) error

	// Evict removes an object from cache (not from disk).
	Evict(hash string)

	// Clear clears the in-memory cache.
	Clear()
}
