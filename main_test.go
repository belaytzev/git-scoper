package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegration_fullPipeline(t *testing.T) {
	base := t.TempDir()

	// Create config file
	os.WriteFile(filepath.Join(base, "gitconfig"), []byte("name=Test User\nemail=test@example.com\n"), 0644)

	// Create two repos and one plain dir
	repoA := filepath.Join(base, "RepoA")
	repoB := filepath.Join(base, "sub", "RepoB")
	exec.Command("git", "init", repoA).Run()
	os.MkdirAll(filepath.Join(base, "sub"), 0755)
	exec.Command("git", "init", repoB).Run()
	os.MkdirAll(filepath.Join(base, "not-a-repo"), 0755)

	// Resolve config
	cfg, err := resolveConfig(base)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.Name != "Test User" {
		t.Errorf("Name: got %q", cfg.Name)
	}

	// Scan
	repos, skipped, err := scanDirs(base, 2)
	if err != nil {
		t.Fatalf("scanDirs: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("repos: got %d, want 2: %v", len(repos), repos)
	}
	if len(skipped) != 1 {
		t.Errorf("skipped: got %d, want 1: %v", len(skipped), skipped)
	}
	if !strings.HasSuffix(skipped[0], "not-a-repo") {
		t.Errorf("unexpected skipped dir: %s", skipped[0])
	}

	// Apply
	results := runAll(repos, cfg, 2)
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("runAll: %s failed: %v", r.Path, r.Err)
		}
	}

	// Verify git config was actually set
	for _, repo := range repos {
		out, err := exec.Command("git", "-C", repo, "config", "--local", "user.name").Output()
		if err != nil {
			t.Errorf("verify %s: %v", repo, err)
			continue
		}
		if strings.TrimSpace(string(out)) != "Test User" {
			t.Errorf("repo %s: user.name = %q", repo, strings.TrimSpace(string(out)))
		}
	}
}
