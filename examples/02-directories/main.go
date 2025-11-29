package main

import (
	"fmt"
	"log"

	"github.com/aweris/cafs"
)

func main() {
	fs, err := cafs.Open("test/merkle:main", cafs.WithCacheDir("/tmp/cafs-merkle"))
	if err != nil {
		log.Fatal(err)
	}

	// Store some files in a directory structure
	files := map[string]string{
		"src/main.go":        "package main\n\nfunc main() {}\n",
		"src/util.go":        "package main\n\nfunc helper() {}\n",
		"src/lib/foo.go":     "package lib\n\nfunc Foo() {}\n",
		"src/lib/bar.go":     "package lib\n\nfunc Bar() {}\n",
		"tests/main_test.go": "package main\n\nfunc TestMain() {}\n",
	}

	fmt.Println("=== Storing files ===")
	for path, content := range files {
		digest, err := fs.Blobs().Put([]byte(content))
		if err != nil {
			log.Fatal(err)
		}
		fs.Index().Set(path, digest)
		fmt.Printf("  %s → %s\n", path, digest[:24])
	}

	// Compute merkle hashes for directories
	fmt.Println("\n=== Directory hashes (computed, not stored) ===")
	fmt.Printf("  src/lib/  → %s\n", fs.Index().Hash("src/lib/")[:24])
	fmt.Printf("  src/      → %s\n", fs.Index().Hash("src/")[:24])
	fmt.Printf("  tests/    → %s\n", fs.Index().Hash("tests/")[:24])
	fmt.Printf("  (root)    → %s\n", fs.Root()[:24])

	// List files in a directory
	fmt.Println("\n=== List src/lib/ ===")
	for name, digest := range fs.Index().List("src/lib/") {
		fmt.Printf("  %s → %s\n", name, digest[:24])
	}

	// Detect changes
	fmt.Println("\n=== Detect changes ===")
	oldLibHash := fs.Index().Hash("src/lib/")

	// Modify a file in src/lib/
	newDigest, _ := fs.Blobs().Put([]byte("package lib\n\nfunc Foo() { /* updated */ }\n"))
	fs.Index().Set("src/lib/foo.go", newDigest)

	newLibHash := fs.Index().Hash("src/lib/")
	if oldLibHash != newLibHash {
		fmt.Printf("  src/lib/ changed!\n")
		fmt.Printf("    old: %s\n", oldLibHash[:24])
		fmt.Printf("    new: %s\n", newLibHash[:24])
	}

	// Deduplication demo
	fmt.Println("\n=== Deduplication ===")
	content := []byte("shared content")
	d1, _ := fs.Blobs().Put(content)
	d2, _ := fs.Blobs().Put(content)
	fmt.Printf("  Same content, same digest: %v\n", d1 == d2)
	fmt.Printf("  Digest: %s\n", d1[:24])

	// Index same content under different keys
	fs.Index().Set("config/dev.json", d1)
	fs.Index().Set("config/prod.json", d1)
	fmt.Println("  Indexed under config/dev.json and config/prod.json")
	fmt.Printf("  config/ hash: %s\n", fs.Index().Hash("config/")[:24])
}
