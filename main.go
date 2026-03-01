package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	depth := flag.Int("depth", 2, "max directory depth to scan")
	workers := flag.Int("workers", 4, "parallel workers")
	flag.Parse()

	baseDir := "."
	if flag.NArg() > 0 {
		baseDir = flag.Arg(0)
	}
	baseDir, _ = filepath.Abs(baseDir)

	if _, err := os.Stat(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access directory %s\n", baseDir)
		os.Exit(1)
	}

	_ = depth
	_ = workers
	fmt.Println("git-scoper: not yet implemented")
}
