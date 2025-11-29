package cafs

import (
	"context"
	"iter"
)

// Digest is an OCI content identifier (e.g., "sha256:abc123...").
type Digest string

// FS provides content-addressed storage with OCI sync.
type FS interface {
	Blobs() BlobStore
	Index() Index

	Sync() error                                  // persist index locally
	Push(ctx context.Context, tags ...string) error // push to remote tags (default: current tag)
	Pull(ctx context.Context) error               // pull from remote
	Close() error                                 // calls Sync()

	Root() Digest
	Dirty() bool
}

// BlobStore handles content-addressed blob operations.
type BlobStore interface {
	Put(data []byte) (Digest, error)
	Get(digest Digest) ([]byte, error)
	Stat(digest Digest) (size int64, exists bool)
	Path(digest Digest) string
}

// Index maps keys to digests with merkle tree computation.
type Index interface {
	Set(key string, digest Digest)
	Get(key string) (Digest, bool)
	Delete(key string)
	Entries() iter.Seq2[string, Digest]

	Hash(prefix string) Digest
	List(prefix string) iter.Seq2[string, Digest]
}
