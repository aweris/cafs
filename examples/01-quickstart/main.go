package main

import (
	"fmt"
	"log"

	"github.com/aweris/cafs"
)

func main() {
	fs, err := cafs.Open("demo/quickstart:main", cafs.WithCacheDir(".cafs"))
	if err != nil {
		log.Fatal(err)
	}
	defer fs.Close()

	// Store content → get digest
	digest, err := fs.Blobs().Put([]byte("Hello, CAFS!"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Stored: %s\n", digest[:20])

	// Index by key
	fs.Index().Set("greeting", digest)

	// Lookup by key
	found, _ := fs.Index().Get("greeting")
	fmt.Printf("Lookup: %s\n", found[:20])

	// Load content
	data, _ := fs.Blobs().Get(found)
	fmt.Printf("Content: %s\n", data)

	// Root hash changes when index changes
	fmt.Printf("Root: %s\n", fs.Root()[:20])
}
