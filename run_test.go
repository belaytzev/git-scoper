package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAll_updatesAllRepos(t *testing.T) {
	base := t.TempDir()
	repoA := filepath.Join(base, "A")
	repoB := filepath.Join(base, "B")
	for _, r := range []string{repoA, repoB} {
		if out, err := exec.Command("git", "init", r).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
	}

	cfg := &Config{Name: "Jane Doe", Email: "jane@co.com"}
	results := runAll([]string{repoA, repoB}, cfg, 2)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("repo %s failed: %v", r.Path, r.Err)
		}
	}
	// Verify config was actually written, not just that no error was returned
	for _, repo := range []string{repoA, repoB} {
		out, err := exec.Command("git", "-C", repo, "config", "--local", "user.name").Output()
		if err != nil {
			t.Errorf("read user.name from %s: %v", repo, err)
			continue
		}
		if strings.TrimSpace(string(out)) != "Jane Doe" {
			t.Errorf("repo %s: user.name = %q, want %q", repo, strings.TrimSpace(string(out)), "Jane Doe")
		}
		out, err = exec.Command("git", "-C", repo, "config", "--local", "user.email").Output()
		if err != nil {
			t.Errorf("read user.email from %s: %v", repo, err)
			continue
		}
		if strings.TrimSpace(string(out)) != "jane@co.com" {
			t.Errorf("repo %s: user.email = %q, want %q", repo, strings.TrimSpace(string(out)), "jane@co.com")
		}
	}
}

func TestRunAll_emptyList(t *testing.T) {
	cfg := &Config{Name: "Jane Doe", Email: "jane@co.com"}
	results := runAll([]string{}, cfg, 4)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRunAll_failingRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo — applyConfig will fail
	cfg := &Config{Name: "Jane Doe", Email: "jane@co.com"}
	results := runAll([]string{dir}, cfg, 1)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err == nil {
		t.Error("expected error for non-repo directory")
	}
}

func TestRunAll_singleWorker(t *testing.T) {
	base := t.TempDir()
	var repos []string
	for _, name := range []string{"R1", "R2", "R3"} {
		r := filepath.Join(base, name)
		if out, err := exec.Command("git", "init", r).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
		repos = append(repos, r)
	}
	cfg := &Config{Name: "Jane Doe", Email: "jane@co.com"}
	results := runAll(repos, cfg, 1) // single worker
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("repo %s failed: %v", r.Path, r.Err)
		}
	}
}

func TestRunAll_zeroWorkers(t *testing.T) {
	base := t.TempDir()
	repo := filepath.Join(base, "R")
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	cfg := &Config{Name: "Jane Doe", Email: "jane@co.com"}
	// workers=0 should be clamped to 1 by runAll's guard
	results := runAll([]string{repo}, cfg, 0)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("unexpected error: %v", results[0].Err)
	}
}
