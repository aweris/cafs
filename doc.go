// Package cafs provides content-addressable storage with OCI registry backend.
//
// CAFS stores objects by content hash and indexes them by arbitrary keys.
// All operations are lockless and safe for concurrent use.
//
// Basic usage:
//
//	fs, _ := cafs.Open("myorg/cache:main", cafs.WithRegistry("ttl.sh"))
//
//	// Store content
//	hash, path, _ := fs.Store(data)
//
//	// Index by key
//	fs.Index("my-key", hash)
//
//	// Lookup and load
//	hash, _ := fs.Lookup("my-key")
//	data, _ := fs.Load(hash)
//
//	// Sync to remote
//	fs.Push(ctx)
//	fs.Pull(ctx)
package cafs
