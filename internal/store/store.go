// Package store implements the local content storage layer.
//
// The Store interface provides content-addressed object storage:
// - PutBlob/PutTree for storing files and directories
// - Get/Has for retrieval and existence checks
// - Filesystem-based with LRU cache
package store

import "context"

// Store handles local content storage.
type Store interface {
	// Get retrieves an object by hash.
	Get(ctx context.Context, hash string) ([]byte, error)

	// PutBlob stores file content with git-style blob hashing.
	// Hash = SHA256("blob {size}\0" + content), stores raw content on disk.
	PutBlob(ctx context.Context, content []byte) (hash string, err error)

	// PutTree stores tree structure as-is.
	// Hash = SHA256(encoded), stores encoded data on disk.
	PutTree(ctx context.Context, encoded []byte) (hash string, err error)

	// Has checks if an object exists.
	Has(ctx context.Context, hash string) (bool, error)

	// GetMulti retrieves multiple objects (batch operation).
	GetMulti(ctx context.Context, hashes []string) (map[string][]byte, error)

	// PutWithHash stores data at a given hash (for objects from remote).
	PutWithHash(ctx context.Context, hash string, data []byte) error

	// PutMulti stores multiple objects by hash (batch operation, for remote).
	PutMulti(ctx context.Context, objects map[string][]byte) error

	// GetRef retrieves a reference (namespace:ref → hash).
	GetRef(namespace, ref string) (string, error)

	// PutRef stores a reference.
	PutRef(namespace, ref, hash string) error

	// Evict removes an object from cache (not from disk).
	Evict(hash string)

	// Clear clears the in-memory cache.
	Clear()

	// Path returns the filesystem path for a given hash.
	Path(hash string) string
}
