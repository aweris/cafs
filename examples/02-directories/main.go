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
	defer fs.Close()

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
		if err := fs.Put(path, []byte(content)); err != nil {
			log.Fatal(err)
		}
		info, _ := fs.Stat(path)
		fmt.Printf("  %s → %s\n", path, info.Digest[:24])
	}

	// Compute merkle hashes for directories
	fmt.Println("\n=== Directory hashes (computed, not stored) ===")
	fmt.Printf("  src/lib/  → %s\n", fs.Hash("src/lib/")[:24])
	fmt.Printf("  src/      → %s\n", fs.Hash("src/")[:24])
	fmt.Printf("  tests/    → %s\n", fs.Hash("tests/")[:24])
	fmt.Printf("  (root)    → %s\n", fs.Root()[:24])

	// List files in a directory
	fmt.Println("\n=== List src/lib/ ===")
	for name, info := range fs.List("src/lib/") {
		fmt.Printf("  %s → %s\n", name, info.Digest[:24])
	}

	// Detect changes
	fmt.Println("\n=== Detect changes ===")
	oldLibHash := fs.Hash("src/lib/")

	// Modify a file in src/lib/
	fs.Put("src/lib/foo.go", []byte("package lib\n\nfunc Foo() { /* updated */ }\n"))

	newLibHash := fs.Hash("src/lib/")
	if oldLibHash != newLibHash {
		fmt.Printf("  src/lib/ changed!\n")
		fmt.Printf("    old: %s\n", oldLibHash[:24])
		fmt.Printf("    new: %s\n", newLibHash[:24])
	}

	// Deduplication demo
	fmt.Println("\n=== Deduplication ===")
	content := []byte("shared content")
	fs.Put("file1", content)
	fs.Put("file2", content)
	i1, _ := fs.Stat("file1")
	i2, _ := fs.Stat("file2")
	fmt.Printf("  Same content, same digest: %v\n", i1.Digest == i2.Digest)
	fmt.Printf("  Digest: %s\n", i1.Digest[:24])

	// Index same content under different keys
	fs.Put("config/dev.json", content)
	fs.Put("config/prod.json", content)
	fmt.Println("  Indexed under config/dev.json and config/prod.json")
	fmt.Printf("  config/ hash: %s\n", fs.Hash("config/")[:24])
}
