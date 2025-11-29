# cafs

Content-Addressable File System with OCI registry sync.

Store files by content hash. Index them by any key. Compare directories instantly. Sync anywhere.

> **Note:** This project is in early MVP stage. APIs may change. Use at your own risk.

## Install

**Library:**
```bash
go get github.com/aweris/cafs
```

**CLI:**
```bash
go install github.com/aweris/cafs/cmd/cafs@latest
```

**Docker:**
```bash
docker pull ghcr.io/aweris/cafs:latest
```

## Quick Start

```go
fs, _ := cafs.Open("ttl.sh/myorg/cache:main")

// Store content
digest, _ := fs.Blobs().Put([]byte("hello world"))

// Index by key
fs.Index().Set("greeting.txt", digest)

// Retrieve
d, _ := fs.Index().Get("greeting.txt")
data, _ := fs.Blobs().Get(d)

// Compare directories instantly
if fs.Index().Hash("src/") != lastKnownHash {
    // something changed
}

// Sync to registry
fs.Push(ctx)
fs.Close()
```

## CLI Usage

```bash
# Pull from registry
cafs pull ttl.sh/myorg/cache:main

# Push to registry
cafs push ttl.sh/myorg/cache:main

# Push to multiple tags
cafs push ttl.sh/myorg/cache:main latest v1.0

# List entries
cafs list ttl.sh/myorg/cache:main
cafs list ttl.sh/myorg/cache:main src/  # with prefix filter
```

## Core Concepts

**Blobs** — Content-addressed storage. Same content = same digest = stored once.

**Index** — Maps keys to digests. Like git's index, but flat.

**Merkle** — Directory hashes computed on-demand from flat index:

```
Index (flat):                    Computed:
┌─────────────────────────┐     ┌─────────────────────────┐
│ foo/bar/a.go → abc123   │     │ foo/bar/   → hash(...)  │
│ foo/bar/b.go → def456   │ ──▶ │ foo/       → hash(...)  │
│ foo/x.go     → ghi789   │     │ (root)     → hash(...)  │
└─────────────────────────┘     └─────────────────────────┘
```

**Sync** — Push/pull to OCI registries with zstd compression and incremental updates.

## Configuration

### Options

```go
fs, _ := cafs.Open("ttl.sh/myorg/cache:main",
    cafs.WithCacheDir("/tmp/my-cache"),     // custom cache location
    cafs.WithAutoPull(cafs.AutoPullAlways), // auto-pull on open
)
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `XDG_DATA_HOME` | Base for cache dir (default: `~/.local/share`) |
| `XDG_CONFIG_HOME` | Base for config dir (default: `~/.config`) |
| `CAFS_CACHE_DIR` | Override cache directory |

### CLI Config

Config file: `~/.config/cafs/config.yaml`

```yaml
cache_dir: /custom/cache/path
```

## API Reference

### FS Interface

```go
type FS interface {
    Blobs() BlobStore
    Index() Index

    Sync() error                                    // persist locally
    Push(ctx context.Context, tags ...string) error // push to registry
    Pull(ctx context.Context) error                 // pull from registry
    Close() error                                   // calls Sync()

    Root() Digest  // root hash of entire index
    Dirty() bool   // true if unsaved changes
}
```

### BlobStore Interface

```go
type BlobStore interface {
    Put(data []byte) (Digest, error)
    Get(digest Digest) ([]byte, error)
    Stat(digest Digest) (size int64, exists bool)
    Path(digest Digest) string
}
```

### Index Interface

```go
type Index interface {
    Set(key string, digest Digest)
    Get(key string) (Digest, bool)
    Delete(key string)
    Entries() iter.Seq2[string, Digest]

    Hash(prefix string) Digest              // merkle hash of subtree
    List(prefix string) iter.Seq2[string, Digest]
}
```

## Examples

See [examples/](examples/) for complete working examples:

- **01-quickstart** — Basic blob storage and indexing
- **02-directories** — Merkle tree directory hashing
- **03-remote** — Push/pull to OCI registries
- **04-gocacheprog** — GOCACHEPROG implementation for Go build cache

## Why CAFS?

- **Deduplication** — Same content stored once, referenced many times
- **Instant diff** — Compare directories by hash, not file-by-file
- **Portable** — Push/pull snapshots to any OCI registry
- **Concurrent** — All operations are lockless and thread-safe
- **Incremental** — Only changed prefixes are uploaded

## License

Apache 2.0
