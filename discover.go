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

	// Track which direct children contain at least one repo (so we don't skip them)
	directChildHasRepo := map[string]bool{}
	// Track direct children that could not be read — they are silently ignored, not "Skipped"
	unreadableDirs := map[string]bool{}
	// Track all repo paths found in the first pass to avoid redundant stat calls
	repoSet := map[string]bool{}

	// First pass: collect all repos
	err = filepath.WalkDir(baseClean, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			rel, relErr := filepath.Rel(baseClean, path)
			if relErr == nil && rel != "." {
				topSegment := strings.SplitN(rel, string(os.PathSeparator), 2)[0]
				unreadableDirs[filepath.Join(baseClean, topSegment)] = true
			}
			return nil // skip unreadable directories
		}
		if !d.IsDir() {
			return nil
		}
		if path == baseClean {
			return nil
		}

		rel, _ := filepath.Rel(baseClean, path)
		depth := len(strings.Split(rel, string(os.PathSeparator)))

		_, statErr := os.Stat(filepath.Join(path, ".git"))
		isRepo := statErr == nil

		if isRepo {
			repos = append(repos, path)
			repoSet[path] = true
			// Mark the direct child ancestor as having a repo
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
		if directChildHasRepo[childPath] || unreadableDirs[childPath] || repoSet[childPath] {
			continue
		}
		skipped = append(skipped, childPath)
	}

	return
}
