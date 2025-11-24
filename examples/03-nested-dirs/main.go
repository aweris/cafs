package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"time"

	"github.com/aweris/cafs"
)

func main() {
	namespace := fmt.Sprintf("test-%d/nested:main", time.Now().Unix())
	workspace, err := cafs.Open(namespace,
		cafs.WithRegistry("ttl.sh"),
		cafs.WithCacheDir(".cafs"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Creating nested directory structure...")

	files := map[string]string{
		"/README.md":              "# My Application",
		"/src/main.go":            "package main\n\nfunc main() {}",
		"/src/handlers/users.go":  "package handlers\n\nfunc GetUsers() {}",
		"/src/handlers/orders.go": "package handlers\n\nfunc GetOrders() {}",
		"/config/app.yaml":        "port: 8080\nenv: production",
		"/docs/api.md":            "# API Documentation",
	}

	for path, content := range files {
		if err := workspace.WriteFile(path, []byte(content), 0644); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\nReading files:")
	readme, _ := workspace.ReadFile("/README.md")
	fmt.Printf("  %s\n", readme)

	fmt.Println("\nListing /src/handlers:")
	entries, _ := workspace.ReadDir("/src/handlers")
	for _, entry := range entries {
		fmt.Printf("  %s\n", entry.Name())
	}

	fmt.Println("\nDirectory tree:")
	fs.WalkDir(workspace, "/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			fmt.Printf("  📁 %s/\n", path)
		} else {
			info, _ := d.Info()
			fmt.Printf("  📄 %s (%d bytes)\n", path, info.Size())
		}
		return nil
	})

	fmt.Println("\nPushing snapshot...")
	hash, err := workspace.Push(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Snapshot: %s\n", hash[:16])
}
