# CAFS

[![Go Reference](https://pkg.go.dev/badge/github.com/aweris/cafs.svg)](https://pkg.go.dev/github.com/aweris/cafs)
[![Go Report Card](https://goreportcard.com/badge/github.com/aweris/cafs)](https://goreportcard.com/report/github.com/aweris/cafs)

Content-Addressed Filesystem with OCI registry backend for Go.

## Overview

CAFS combines Git's content-addressed storage with the standard `fs.FS` interface and OCI registries for distribution. Files are stored by content hash, automatically deduplicated, and synced across machines via container registries.

```go
fs, _ := cafs.Open("myorg/project:main", cafs.WithRegistry("ttl.sh"))
fs.WriteFile("/config.json", data, 0644)
hash, _ := fs.Push(ctx)  // Push to OCI registry
```

## Features

- Content-addressed storage with automatic deduplication
- OCI registry backend (Docker Hub, ttl.sh, GHCR, or any OCI-compatible registry)
- Standard Go `fs.FS` interface
- Zstd compression with intelligent skip for incompressible data
- Distributed workflows with auto-pull and prefetch
- Git-like snapshots with immutable content hashing

## Installation

```bash
go get github.com/aweris/cafs
```

## Quick Start

```go
import (
    "context"
    "github.com/aweris/cafs"
)

// Open workspace
fs, err := cafs.Open("myorg/project:main",
    cafs.WithRegistry("ttl.sh"))

// Write files (nested directories created automatically)
fs.WriteFile("/src/main.go", []byte("package main"), 0644)
fs.WriteFile("/config.json", []byte(`{"port": 8080}`), 0644)

// Push snapshot to registry
hash, err := fs.Push(context.Background())

// Another machine: pull and read
fs2, _ := cafs.Open("myorg/project:main",
    cafs.WithRegistry("ttl.sh"),
    cafs.WithAutoPullIfMissing())

data, _ := fs2.ReadFile("/config.json")  // Content automatically available
```

## Documentation

- [GoDoc](https://pkg.go.dev/github.com/aweris/cafs) - Complete API reference
- [Examples](./examples) - Working code examples
  - [01-quickstart](./examples/01-quickstart) - Basic operations
  - [02-distributed](./examples/02-distributed) - Multi-machine workflow
  - [03-nested-dirs](./examples/03-nested-dirs) - Directory operations

## Status

⚠️ **Alpha** - Core functionality works, APIs may change

**Working:**
- ✅ File operations (read/write/open/stat/readdir)
- ✅ OCI push/pull with compression
- ✅ Auto-pull and eager prefetch
- ✅ Nested directories and tree walking
- ✅ Immutable snapshots with content hashing

**Planned:**
- ⏳ Remove/Rename operations
- ⏳ File handle Write/Seek
- ⏳ Diff and merge operations

## Architecture

CAFS uses a Git-like Merkle tree:
- **Blobs** - Files stored by SHA256 hash
- **Trees** - Directories containing references to children
- **Snapshots** - Immutable root hashes
- **Local store** - Compressed objects in `~/.local/share/cafs`
- **Remote store** - OCI layers in container registries

## Inspiration

- **Git** - Content addressing and object model
- **OCI** - Distribution and registry standards
- **Afero** - Filesystem abstraction
- **IPFS** - Content-addressed storage

## License

Apache License 2.0 - see [LICENSE](LICENSE) file

## Contributing

Contributions welcome! Please open an issue first to discuss changes.
