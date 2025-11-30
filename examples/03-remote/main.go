package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aweris/cafs"
)

func main() {
	// Use ttl.sh (free, anonymous, temporary registry)
	remoteRef := fmt.Sprintf("ttl.sh/cafs-demo-%d:main", time.Now().Unix())
	ctx := context.Background()

	// Create store with remote sync
	fs, err := cafs.Open("demo/remote:main",
		cafs.WithCacheDir("/tmp/cafs-remote-demo"),
		cafs.WithRemote(remoteRef),
	)
	if err != nil {
		log.Fatal(err)
	}

	fs.Put("message", []byte("Hello from CAFS!"))
	fmt.Printf("Created: root=%s\n", fs.Root()[:20])

	// Push to remote
	fmt.Println("Pushing to remote...")
	if err := fs.Push(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Pushed!")

	rootAfterPush := fs.Root()
	fs.Close()

	// Simulate fresh start - open new store, pull from remote
	fmt.Println("\nOpening fresh store...")
	fs2, err := cafs.Open("demo/remote:main",
		cafs.WithCacheDir("/tmp/cafs-remote-demo-2"),
		cafs.WithRemote(remoteRef),
		cafs.WithAutoPull("always"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer fs2.Close()

	fmt.Printf("Pulled: root=%s\n", fs2.Root()[:20])
	fmt.Printf("Roots match: %v\n", fs2.Root() == rootAfterPush)

	// Verify data
	data, _ := fs2.Get("message")
	fmt.Printf("Content: %s\n", data)
}
