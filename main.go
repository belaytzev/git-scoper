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

	if info, err := os.Stat(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access directory %s: %v\n", baseDir, err)
		os.Exit(1)
	} else if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", baseDir)
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

	type entry struct {
		label string
		path  string
		err   error
	}
	var entries []entry
	for _, r := range results {
		rel, relErr := filepath.Rel(baseDir, r.Path)
		if relErr != nil {
			rel = r.Path
		}
		if r.Err != nil {
			entries = append(entries, entry{"Failed", rel, r.Err})
		} else {
			entries = append(entries, entry{"Updated", rel, nil})
		}
	}
	for _, s := range skipped {
		rel, relErr := filepath.Rel(baseDir, s)
		if relErr != nil {
			rel = s
		}
		entries = append(entries, entry{"Skipped", rel, nil})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })

	updated, failed, skippedCount := 0, 0, 0
	for _, e := range entries {
		switch e.label {
		case "Updated":
			fmt.Printf("Updated: %s\n", e.path)
			updated++
		case "Failed":
			fmt.Printf("Failed: %s (%v)\n", e.path, e.err)
			failed++
		case "Skipped":
			fmt.Printf("Skipped: %s\n", e.path)
			skippedCount++
		}
	}

	fmt.Println("------------------------")
	fmt.Printf("Done. %d updated, %d failed, %d skipped.\n", updated, failed, skippedCount)
	if failed > 0 {
		os.Exit(1)
	}
}
