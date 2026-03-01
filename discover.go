package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// scanDirs walks baseDir up to maxDepth, returning:
//   - repos: paths that contain a .git directory
//   - skipped: direct children of baseDir with no .git and no repo-containing subdirs
func scanDirs(baseDir string, maxDepth int) (repos []string, skipped []string, err error) {
	baseClean := filepath.Clean(baseDir)

	// If baseDir itself is a git repo, return it directly — no child scanning needed.
	if _, statErr := os.Stat(filepath.Join(baseClean, ".git")); statErr == nil {
		repos = append(repos, baseClean)
		return
	}

	baseParts := len(strings.Split(baseClean, string(os.PathSeparator)))

	// Track which direct children contain at least one repo (so we don't skip them)
	directChildHasRepo := map[string]bool{}

	// First pass: collect all repos
	err = filepath.WalkDir(baseClean, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable directories
		}
		if !d.IsDir() {
			return nil
		}
		if path == baseClean {
			return nil
		}

		parts := len(strings.Split(filepath.Clean(path), string(os.PathSeparator)))
		depth := parts - baseParts

		_, statErr := os.Stat(filepath.Join(path, ".git"))
		isRepo := statErr == nil

		if isRepo {
			repos = append(repos, path)
			// Mark the direct child ancestor as having a repo
			rel, _ := filepath.Rel(baseClean, path)
			topSegment := strings.SplitN(rel, string(os.PathSeparator), 2)[0]
			directChildHasRepo[filepath.Join(baseClean, topSegment)] = true
			return filepath.SkipDir
		}

		if depth >= maxDepth {
			return filepath.SkipDir
		}

		return nil
	})
	if err != nil {
		return
	}

	// Second pass: find direct children that have no repo and weren't traversed into productively
	entries, readErr := os.ReadDir(baseClean)
	if readErr != nil {
		err = readErr
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		childPath := filepath.Join(baseClean, e.Name())
		if directChildHasRepo[childPath] {
			continue
		}
		// Check if it's itself a repo (already in repos list)
		_, statErr := os.Stat(filepath.Join(childPath, ".git"))
		if statErr == nil {
			continue // it's a repo, not skipped
		}
		skipped = append(skipped, childPath)
	}

	return
}
