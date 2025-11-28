package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aweris/cafs"
)

func main() {
	namespace := fmt.Sprintf("test-%d/quickstart:main", time.Now().Unix())

	fs, err := cafs.Open(namespace,
		cafs.WithRegistry("ttl.sh"),
		cafs.WithCacheDir(".cafs"))
	if err != nil {
		log.Fatal(err)
	}

	// Store content (lockless, content-addressed)
	hash, path, err := fs.Store([]byte("Hello, CAFS!"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Stored: hash=%s path=%s\n", hash[:16], path)

	// Index by key (lockless)
	fs.Index("greeting", hash)

	// Lookup by key
	foundHash, ok := fs.Lookup("greeting")
	if !ok {
		log.Fatal("key not found")
	}
	fmt.Printf("Lookup: hash=%s\n", foundHash[:16])

	// Load content by hash
	data, err := fs.Load(foundHash)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Loaded: %s\n", data)

	// Push to remote
	indexHash, err := fs.Push(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Pushed index: %s\n", indexHash[:16])
}
