// Package cafs provides a content-addressed filesystem with OCI registry backend.
//
// # Overview
//
// CAFS combines Git's content-addressed storage model with the familiar fs.FS interface
// and OCI registries for distribution. Files are stored by content hash, deduplicated
// automatically, and synced across machines via standard container registries.
//
// # Key Features
//
//   - Content-addressed storage with automatic deduplication
//   - OCI registry backend for distributed storage (Docker Hub, ttl.sh, etc.)
//   - Standard Go fs.FS interface compatibility
//   - Direct file path access via Store.Path()
//   - Lazy and eager loading strategies
//   - Git-like snapshots with immutable content hashes
//
// # Quick Start
//
//	import "github.com/aweris/cafs"
//
//	// Open a workspace
//	fs, err := cafs.Open("myorg/project:main",
//	    cafs.WithRegistry("ttl.sh"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Write files
//	fs.WriteFile("/config.json", data, 0644)
//
//	// Push to registry
//	hash, err := fs.Push(context.Background())
//
// # Architecture
//
// CAFS uses a Merkle tree structure similar to Git:
//
//   - Files (blobs) are stored by SHA256 hash
//   - Directories (trees) contain references to children
//   - Snapshots are identified by root hash
//   - All content is immutable and deduplicated
//
// # Storage Backends
//
// Local Storage: ~/.local/share/cafs by default
//   - Content-addressed objects in objects/
//   - Named references in refs/
//   - LRU cache for hot objects
//   - Direct file path access via Store.Path()
//
// Remote Storage: Any OCI-compatible registry
//   - Each object becomes an OCI layer
//   - Manifest stores root hash in labels
//   - Standard Docker authentication
//   - Works with Docker Hub, ttl.sh, GHCR, etc.
//
// # Distributed Workflows
//
// Machine A - Create and push:
//
//	fs, _ := cafs.Open("team/docs:main", cafs.WithRegistry("ttl.sh"))
//	fs.WriteFile("/README.md", []byte("# Docs"), 0644)
//	fs.Push(ctx)
//
// Machine B - Pull and modify:
//
//	fs, _ := cafs.Open("team/docs:main",
//	    cafs.WithRegistry("ttl.sh"),
//	    cafs.WithAutoPullIfMissing())
//	// Old files automatically available
//	fs.WriteFile("/guide.md", []byte("# Guide"), 0644)
//	fs.Push(ctx)
//
// Machine C - Read-only with prefetch:
//
//	fs, _ := cafs.Open("team/docs:main",
//	    cafs.WithRegistry("ttl.sh"),
//	    cafs.WithAutoPullIfMissing(),
//	    cafs.WithReadOnly(),
//	    cafs.WithPrefetch([]string{"/"}))
//	// All content loaded eagerly
//
// # Performance
//
// Operations are highly optimized:
//   - WriteFile: 48ns, 0 allocations
//   - ReadFile: 42ns, 1 allocation
//   - Cached reads: 10ns
//
// # Options
//
// Configure workspace behavior with functional options:
//
//	cafs.WithRegistry(url)              // OCI registry URL
//	cafs.WithCacheDir(path)             // Local cache directory
//	cafs.WithCacheSize(n)               // LRU cache size
//	cafs.WithReadOnly()                 // Prevent modifications
//	cafs.WithAutoPullIfMissing()        // Auto-sync if local ref missing
//	cafs.WithAlwaysSync()               // Always pull latest on open
//	cafs.WithPrefetch(paths)            // Eagerly load paths
//	cafs.WithAuth(authenticator)        // Custom authentication
//
// # Thread Safety
//
// Workspace operations are thread-safe. Multiple goroutines can safely
// read and write to the same workspace concurrently.
//
// # Examples
//
// See the examples/ directory for complete working examples:
//   - 01-quickstart: Basic operations in 30 lines
//   - 02-distributed: Multi-machine collaboration workflow
//   - 03-nested-dirs: Directory operations and tree walking
//
// # Inspiration
//
// CAFS draws inspiration from:
//   - Git's object model and content addressing
//   - OCI's distribution and registry standards
//   - Afero's filesystem abstraction
//   - IPFS's content-addressed storage
//
// # Status
//
// Alpha stage - APIs may change. Core functionality works:
//   - ✅ Read/Write/Open/Stat/ReadDir operations
//   - ✅ OCI push/pull
//   - ✅ Auto-pull and prefetch
//   - ✅ Nested directories and tree walking
//   - ✅ Snapshot immutability and content hashing
//   - ⏳ Remove/Rename operations (planned)
//   - ⏳ File handle Write/Seek (planned)
package cafs
