package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	depth := flag.Int("depth", 2, "max directory depth to scan")
	workers := flag.Int("workers", 4, "parallel workers")
	flag.Parse()

	if *depth < 1 {
		fmt.Fprintf(os.Stderr, "Error: --depth must be at least 1\n")
		os.Exit(1)
	}
	if *workers < 1 {
		fmt.Fprintf(os.Stderr, "Error: --workers must be at least 1\n")
		os.Exit(1)
	}

	baseDir := "."
	if flag.NArg() > 0 {
		baseDir = flag.Arg(0)
	}
	var absErr error
	baseDir, absErr = filepath.Abs(baseDir)
	if absErr != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot resolve directory: %v\n", absErr)
		os.Exit(1)
	}

	if _, err := os.Stat(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access directory %s\n", baseDir)
		os.Exit(1)
	}

	cfg, err := resolveConfig(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config: name=%s, email=%s\n", cfg.Name, cfg.Email)
	fmt.Printf("Scanning: %s (depth %d)\n", baseDir, *depth)
	fmt.Println("------------------------")

	repos, skipped, err := scanDirs(baseDir, *depth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning: %v\n", err)
		os.Exit(1)
	}

	results := runAll(repos, cfg, *workers)

	// Sort results by path for deterministic output
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })

	updated, failed := 0, 0
	for _, r := range results {
		rel, relErr := filepath.Rel(baseDir, r.Path)
		if relErr != nil {
			rel = r.Path
		}
		if r.Err != nil {
			fmt.Printf("Failed: %s (%v)\n", rel, r.Err)
			failed++
		} else {
			fmt.Printf("Updated: %s\n", rel)
			updated++
		}
	}

	sort.Strings(skipped)
	for _, s := range skipped {
		rel, relErr := filepath.Rel(baseDir, s)
		if relErr != nil {
			rel = s
		}
		fmt.Printf("Skipped: %s\n", rel)
	}

	fmt.Println("------------------------")
	msg := fmt.Sprintf("Done. %d updated", updated)
	if failed > 0 {
		msg += fmt.Sprintf(", %d failed", failed)
	}
	if len(skipped) > 0 {
		msg += fmt.Sprintf(", %d skipped", len(skipped))
	}
	fmt.Println(msg)
}
