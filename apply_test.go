package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
	return dir
}

func TestApplyConfig_success(t *testing.T) {
	repo := initGitRepo(t)
	err := applyConfig(repo, "Jane Doe", "jane@co.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify name was set
	out, err := exec.Command("git", "-C", repo, "config", "--local", "user.name").Output()
	if err != nil {
		t.Fatalf("git config read failed: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "Jane Doe" {
		t.Errorf("user.name: got %q, want %q", got, "Jane Doe")
	}
	// Verify email was set
	out, err = exec.Command("git", "-C", repo, "config", "--local", "user.email").Output()
	if err != nil {
		t.Fatalf("git config read failed: %v", err)
	}
	got = strings.TrimSpace(string(out))
	if got != "jane@co.com" {
		t.Errorf("user.email: got %q, want %q", got, "jane@co.com")
	}
}

func TestApplyConfig_invalidRepo(t *testing.T) {
	dir := t.TempDir() // not a git repo
	err := applyConfig(dir, "Jane Doe", "jane@co.com")
	if err == nil {
		t.Fatal("expected error for non-repo directory")
	}
}

func TestApplyConfig_nonexistentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	err := applyConfig(dir, "Jane Doe", "jane@co.com")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}
