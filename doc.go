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
//	// Store content by key
//	fs.Put("src/main.go", data)
//
//	// Store with metadata
//	fs.Put("src/util.go", data, cafs.WithMeta(cafs.FileMeta{Mode: 0644}))
//
//	// Retrieve content
//	data, _ := fs.Get("src/main.go")
//
//	// Get entry info
//	info, ok := fs.Stat("src/main.go")
//	fmt.Println(info.Digest, info.Size)
//
//	// Decode typed metadata
//	var meta cafs.FileMeta
//	info.DecodeMeta(&meta)
//
//	// Compare directories
//	if fs.Hash("src/") != previousHash {
//	    // src/ changed
//	}
//
//	// Sync to remote
//	fs.Push(ctx)
//	fs.Pull(ctx)
package cafs
