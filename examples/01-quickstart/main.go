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

	// Store content at key
	_ = fs.Put("greeting", []byte("Hello, CAFS!"))

	// Get info about the entry
	info, ok := fs.Stat("greeting")
	if ok {
		fmt.Printf("Stored: %s (size: %d)\n", info.Digest[:20], info.Size)
	}

	// Load content by key
	data, _ := fs.Get("greeting")
	fmt.Printf("Content: %s\n", data)

	// Root hash changes when entries change
	fmt.Printf("Root: %s\n", fs.Root()[:20])
}
