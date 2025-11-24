package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aweris/cafs"
)

func main() {
	ctx := context.Background()
	namespace := fmt.Sprintf("test-%d/distributed:main", time.Now().Unix())

	fmt.Println("=== Machine A: Creating initial files ===")
	machineA, err := cafs.Open(namespace,
		cafs.WithRegistry("ttl.sh"),
		cafs.WithCacheDir(".cafs/machineA"))
	if err != nil {
		log.Fatal(err)
	}

	if err := machineA.WriteFile("/README.md", []byte("# Project Docs"), 0644); err != nil {
		log.Fatal(err)
	}
	if err := machineA.WriteFile("/config.json", []byte(`{"version": "1.0"}`), 0644); err != nil {
		log.Fatal(err)
	}

	hashA, err := machineA.Push(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Pushed snapshot: %s\n", hashA[:16])

	fmt.Println("\n=== Machine B: Pull, modify, and push ===")
	machineB, err := cafs.Open(namespace,
		cafs.WithRegistry("ttl.sh"),
		cafs.WithCacheDir(".cafs/machineB"),
		cafs.WithAutoPullIfMissing(),
	)
	if err != nil {
		log.Fatal(err)
	}

	readme, _ := machineB.ReadFile("/README.md")
	fmt.Printf("Pulled README: %s\n", readme)

	if err := machineB.WriteFile("/README.md", []byte("# Project Docs - Updated by B"), 0644); err != nil {
		log.Fatal(err)
	}
	if err := machineB.WriteFile("/guide.md", []byte("# Quick Start Guide"), 0644); err != nil {
		log.Fatal(err)
	}

	hashB, err := machineB.Push(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Modified and pushed: %s\n", hashB[:16])

	fmt.Println("\n=== Machine C: Pull with prefetch (read-only) ===")
	machineC, err := cafs.Open(namespace,
		cafs.WithRegistry("ttl.sh"),
		cafs.WithCacheDir(".cafs/machineC"),
		cafs.WithAutoPullIfMissing(),
		cafs.WithReadOnly(),
		cafs.WithPrefetch([]string{"/"}),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("All content eagerly loaded via prefetch")

	readmeC, _ := machineC.ReadFile("/README.md")
	fmt.Printf("README: %s\n", readmeC)

	guideC, _ := machineC.ReadFile("/guide.md")
	fmt.Printf("Guide: %s\n", guideC)

	configC, _ := machineC.ReadFile("/config.json")
	fmt.Printf("Config: %s\n", configC)

	hashC := machineC.CurrentDigest()
	fmt.Printf("\nFinal hash matches B: %v\n", hashB == hashC)
	fmt.Printf("All files accessible (old + new): %v\n", len(readmeC) > 0 && len(guideC) > 0 && len(configC) > 0)
}
