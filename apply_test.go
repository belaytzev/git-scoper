package main

import (
	"fmt"
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

// A worktree shares the main repo's .git/config; concurrent writes used to
// collide on git's config lock file.
func TestApplyConfig_concurrentWorktrees(t *testing.T) {
	main := initGitRepo(t)
	for _, args := range [][]string{
		{"-C", main, "commit", "--allow-empty", "-m", "init", "--author", "a <a@b.c>"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Env = append(exec.Command("git").Environ(),
			"GIT_AUTHOR_NAME=a", "GIT_AUTHOR_EMAIL=a@b.c",
			"GIT_COMMITTER_NAME=a", "GIT_COMMITTER_EMAIL=a@b.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, out)
		}
	}

	repos := []string{main}
	for i := 0; i < 4; i++ {
		wt := filepath.Join(t.TempDir(), fmt.Sprintf("wt%d", i))
		out, err := exec.Command("git", "-C", main, "worktree", "add", "-b",
			fmt.Sprintf("b%d", i), wt).CombinedOutput()
		if err != nil {
			t.Fatalf("worktree add failed: %v\n%s", err, out)
		}
		repos = append(repos, wt)
	}

	for _, r := range runAll(repos, &Config{Name: "Jane", Email: "jane@co.com"}, 8) {
		if r.Err != nil {
			t.Errorf("%s: %v", r.Path, r.Err)
		}
	}
}
