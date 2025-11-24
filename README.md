# CAFS

> **⚠️ Early Development - Expect Breaking Changes**
>
> Rough MVP implementation. APIs unstable, architecture shifting, not production-ready.

Content-Addressed Filesystem with OCI Backend - a Go library that makes content-addressed storage feel like a regular filesystem.

## Core Idea

```go
fs, _ := cafs.Open("myorg/project:main", cafs.WithRegistry("ttl.sh"))
fs.WriteFile("/config.json", data, 0644)
hash, _ := fs.Push(ctx)
```

Files stored content-addressed, synced to OCI registries, accessed via familiar filesystem operations.

## Status

**Working:** File ops (read/write), streaming, OCI push/pull, local storage, ref persistence

**Not Working:** Remove/rename ops, file writes, many filesystem methods

Inspired by: Afero, Git, OCI, IPFS
