// Package cafs provides a content-addressable store with OCI registry sync and merkle tree semantics.
//
// CAFS stores blobs by content digest and indexes them by arbitrary keys.
// Directory hashes are computed on-demand from the flat index, enabling
// instant comparison of subtrees without storing tree objects.
//
// Basic usage (local only):
//
//	fs, _ := cafs.Open("myproject:main")
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
//	// Check existence and count
//	if fs.Exists("src/main.go") { ... }
//	fmt.Println(fs.Len(), "entries")
//
//	// Compare directories
//	if fs.Hash("src/") != previousHash {
//	    // src/ changed
//	}
//
//	// Maintenance
//	stats := fs.Stats()           // entry count, blob count, total size
//	removed, _ := fs.GC()         // remove unreferenced blobs
//	fs.Clear()                    // remove all entries
//
// With remote sync:
//
//	fs, _ := cafs.Open("myproject:main", cafs.WithRemote("ttl.sh/myorg/cache"))
//	fs.Push(ctx)
//	fs.Pull(ctx)
//	fmt.Println("remote:", fs.Ref())
package cafs
