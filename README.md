# cafs

Content-Addressable File System with OCI registry sync.

Store files by content hash. Index them by any key. Compare directories instantly. Sync anywhere.

## Install

```bash
go get github.com/aweris/cafs
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

**Sync** — Push/pull to OCI registries. Blobs as layers, index as manifest.

## Example

```go
// Standard Docker image ref format
fs, _ := cafs.Open("ttl.sh/myorg/project:main")

// Store content → get digest
digest, _ := fs.Blobs().Put([]byte("hello world"))

// Index by key
fs.Index().Set("config.json", digest)

// Retrieve
d, _ := fs.Index().Get("config.json")
data, _ := fs.Blobs().Get(d)

// Compare directories
if fs.Index().Hash("src/") != lastKnownHash {
    // something changed
}

// Persistence
fs.Sync()          // persist index locally
fs.Close()         // same as Sync()

// Remote (zstd compressed, incremental sync)
fs.Push(ctx)                  // push to current tag
fs.Push(ctx, "v1", "latest")  // push to multiple tags
fs.Pull(ctx)                  // pull from current ref
```

## Why

- **Deduplication** — Same content stored once, referenced many times
- **Instant diff** — Compare directories by hash, not file-by-file
- **Portable** — Push/pull snapshots to any OCI registry
- **Concurrent** — All operations are lockless and thread-safe

## License

Apache 2.0
