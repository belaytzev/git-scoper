package main

import (
	"os/exec"
	"path/filepath"
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
		exec.Command("git", "init", r).Run()
		repos = append(repos, r)
	}
	cfg := &Config{Name: "Jane Doe", Email: "jane@co.com"}
	results := runAll(repos, cfg, 1) // single worker
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}
