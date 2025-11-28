package cafs

import "context"

// FS is a content-addressable storage with key-based indexing.
type FS interface {
	// Store saves data and returns its content hash and disk path.
	// Lockless, atomic, idempotent - same content always yields same hash.
	Store(data []byte) (hash, path string, err error)

	// Load retrieves data by its content hash.
	Load(hash string) ([]byte, error)

	// Exists checks if content with given hash exists.
	Exists(hash string) bool

	// Path returns the disk path for a given hash.
	Path(hash string) string

	// Index associates a key with a content hash.
	// Lockless - uses sync.Map internally.
	Index(key, hash string)

	// Lookup returns the hash associated with a key.
	Lookup(key string) (hash string, ok bool)

	// Indexed checks if a key exists in the index.
	Indexed(key string) bool

	// Push syncs local state to remote storage.
	// Returns the index hash.
	Push(ctx context.Context) (string, error)

	// Pull fetches state from remote storage.
	Pull(ctx context.Context) error
}
