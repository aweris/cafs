package cafs

import (
	"context"
	"io/fs"
)

// FS is a content-addressed filesystem that composes stdlib interfaces
// with write and sync operations.
type FS interface {
	Reader
	Writer
	Syncer
	Info
}

// Reader provides read-only filesystem access (stdlib compatible).
type Reader interface {
	fs.FS          // Open(name string) (fs.File, error)
	fs.ReadFileFS  // ReadFile(name string) ([]byte, error)
	fs.StatFS      // Stat(name string) (fs.FileInfo, error)
	fs.ReadDirFS   // ReadDir(name string) ([]fs.DirEntry, error)
}

// Writer provides write operations for content.
// Directories are implicit (derived from file paths in the merkle tree).
type Writer interface {
	WriteFile(name string, data []byte, perm fs.FileMode) error
}

// Syncer handles remote synchronization of content-addressed snapshots.
type Syncer interface {
	Push(ctx context.Context) (string, error)
	Pull(ctx context.Context) error
	IsDirty() bool
}

// Info provides metadata about the filesystem state.
type Info interface {
	CurrentDigest() string
	Namespace() string
	Ref() string
}
