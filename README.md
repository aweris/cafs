# CAFS

Content-Addressable Storage with OCI registry backend.

## Overview

Simple, lockless content-addressed storage with key-based indexing. Objects are stored by content hash, indexed by arbitrary keys, and synced to OCI registries.

```go
fs, _ := cafs.Open("myorg/cache:main", cafs.WithRegistry("ttl.sh"))

// Store content (lockless, content-addressed)
hash, path, _ := fs.Store(data)

// Index by key (lockless)
fs.Index("my-key", hash)

// Lookup and load
hash, _ := fs.Lookup("my-key")
data, _ := fs.Load(hash)

// Sync to remote
fs.Push(ctx)
```

## Features

- **Lockless operations** - Store, Load, Index, Lookup all use atomic operations
- **Content-addressed** - Same content = same hash, automatic deduplication
- **Key-based indexing** - Map arbitrary keys to content hashes
- **OCI registry sync** - Push/Pull to any OCI-compatible registry
- **Direct disk path** - Get filesystem path for zero-copy access

## Installation

```bash
go get github.com/aweris/cafs
```

## API

```go
type FS interface {
    // Object store (lockless)
    Store(data []byte) (hash, path string, err error)
    Load(hash string) ([]byte, error)
    Exists(hash string) bool
    Path(hash string) string

    // Index (lockless)
    Index(key, hash string)
    Lookup(key string) (hash string, ok bool)
    Indexed(key string) bool

    // Sync
    Push(ctx context.Context) (string, error)
    Pull(ctx context.Context) error
}
```

## Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/aweris/cafs"
)

func main() {
    fs, _ := cafs.Open("myorg/cache:main",
        cafs.WithRegistry("ttl.sh"),
        cafs.WithCacheDir(".cafs"))

    // Store content
    hash, path, _ := fs.Store([]byte("Hello, World!"))
    fmt.Printf("hash=%s path=%s\n", hash, path)

    // Index by key
    fs.Index("greeting", hash)

    // Lookup by key
    h, _ := fs.Lookup("greeting")
    data, _ := fs.Load(h)
    fmt.Printf("data=%s\n", data)

    // Push to remote
    fs.Push(context.Background())
}
```

## Options

```go
cafs.WithRegistry("ttl.sh")      // OCI registry URL
cafs.WithCacheDir("~/.cafs")     // Local cache directory
cafs.WithCacheSize(1000)         // In-memory LRU cache size
cafs.WithAutoPull("missing")     // Auto-pull on open: "never", "missing", "always"
cafs.WithAuth(authenticator)     // Custom registry authentication
```

## License

Apache License 2.0
