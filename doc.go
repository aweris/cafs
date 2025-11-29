// Package cafs provides OCI-aligned content-addressable storage with merkle tree semantics.
//
// CAFS stores blobs by content digest and indexes them by arbitrary keys.
// Directory hashes are computed on-demand from the flat index, enabling
// instant comparison of subtrees without storing tree objects.
//
// Basic usage:
//
//	fs, _ := cafs.Open("ttl.sh/myorg/cache:main")
//
//	// Store content
//	digest, _ := fs.Blobs().Put(data)
//
//	// Index by key
//	fs.Index().Set("src/main.go", digest)
//
//	// Lookup and load
//	digest, _ := fs.Index().Get("src/main.go")
//	data, _ := fs.Blobs().Get(digest)
//
//	// Compare directories
//	if fs.Index().Hash("src/") != previousHash {
//	    // src/ changed
//	}
//
//	// Sync to remote
//	fs.Push(ctx)
//	fs.Pull(ctx)
package cafs
