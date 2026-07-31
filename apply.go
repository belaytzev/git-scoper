package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	locksMu sync.Mutex
	locks   = map[string]*sync.Mutex{}
)

// configOwner returns a key identifying the config file a repo writes to.
// A worktree's .git is a file pointing into <main>/.git/worktrees/<name>, and
// `git config --local` there writes the *main* repo's config — so a worktree
// and its main repo must never be written concurrently, or git's lock collides.
func configOwner(repoPath string) string {
	gitPath := filepath.Join(repoPath, ".git")
	st, err := os.Stat(gitPath)
	if err != nil || st.IsDir() {
		return resolve(gitPath)
	}
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return resolve(gitPath)
	}
	dir := filepath.ToSlash(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(data)), "gitdir:")))
	if i := strings.Index(dir, "/worktrees/"); i >= 0 {
		return resolve(dir[:i])
	}
	return resolve(dir) // submodule: has its own config
}

// resolve canonicalises a path so that two spellings of the same directory
// (e.g. /var vs /private/var) map to one lock.
func resolve(path string) string {
	if p, err := filepath.EvalSymlinks(path); err == nil {
		return p
	}
	return path
}

// lockFor returns the mutex guarding a given config file.
func lockFor(key string) *sync.Mutex {
	locksMu.Lock()
	defer locksMu.Unlock()
	mu, ok := locks[key]
	if !ok {
		mu = &sync.Mutex{}
		locks[key] = mu
	}
	return mu
}

// applyConfig sets user.name and user.email in the given git repo's local config.
func applyConfig(repoPath, name, email string) error {
	mu := lockFor(configOwner(repoPath))
	mu.Lock()
	defer mu.Unlock()

	cmd := exec.Command("git", "-C", repoPath, "config", "--local", "--", "user.name", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("git config user.name failed: %s", msg)
		}
		return fmt.Errorf("git config user.name failed: %w", err)
	}
	cmd = exec.Command("git", "-C", repoPath, "config", "--local", "--", "user.email", email)
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("git config user.email failed: %s", msg)
		}
		return fmt.Errorf("git config user.email failed: %w", err)
	}
	return nil
}
