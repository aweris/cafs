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
	fs, err := cafs.Open(namespace, cafs.WithRegistry("ttl.sh"))
	if err != nil {
		log.Fatal(err)
	}

	if err := fs.WriteFile("/hello.txt", []byte("Hello, CAFS!"), 0644); err != nil {
		log.Fatal(err)
	}

	data, err := fs.ReadFile("/hello.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Read: %s\n", data)

	hash, err := fs.Push(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Snapshot: %s\n", hash)
}
