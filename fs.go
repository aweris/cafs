package cafs

import (
	"context"
	"iter"
	"os"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

// Digest is an OCI content identifier (e.g., "sha256:abc123...").
type Digest string

// Info represents metadata about a stored entry.
type Info struct {
	Digest Digest // content hash
	Size   int64  // content size
	Meta   any    // optional user-defined metadata
}

// DecodeMeta decodes the metadata into a typed struct using mapstructure.
func (i Info) DecodeMeta(out any) error {
	if i.Meta == nil {
		return nil
	}
	return mapstructure.Decode(i.Meta, out)
}

// FileMeta provides common file system metadata.
type FileMeta struct {
	Mode    os.FileMode `json:"mode,omitempty" mapstructure:"mode"`
	ModTime time.Time   `json:"mtime,omitempty" mapstructure:"mtime"`
}

// FileMetaFrom creates FileMeta from os.FileInfo.
func FileMetaFrom(info os.FileInfo) FileMeta {
	return FileMeta{
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
	}
}

// Stats contains storage statistics.
type Stats struct {
	Entries   int   // number of user entries
	Blobs     int   // number of unique blobs on disk
	TotalSize int64 // total size of all blobs
}

// Store provides content-addressed storage with OCI sync.
type Store interface {
	// Core operations
	Put(key string, data []byte, opts ...Option) error
	Get(key string) ([]byte, error)
	Stat(key string) (Info, bool)
	Delete(key string)
	Clear()

	// Iteration
	List(prefix string) iter.Seq2[string, Info]

	// Tree hash
	Hash(prefix string) Digest

	// Sync
	Sync() error
	Push(ctx context.Context, tags ...string) error
	Pull(ctx context.Context) error
	Close() error

	// Status
	Root() Digest
	Dirty() bool
	Len() int
	Ref() string
	Exists(key string) bool
	Stats() Stats

	// Maintenance
	GC() (removed int, err error)

	// Advanced
	Path(digest Digest) string
}

// Option configures a Put operation.
type Option func(*Info)

// WithMeta sets custom metadata on the entry.
func WithMeta(v any) Option {
	return func(i *Info) {
		i.Meta = v
	}
}
