package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func makeRepo(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("makeRepo: %v", err)
	}
	return dir
}

func makeDir(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("makeDir: %v", err)
	}
	return dir
}

func TestScanDirs_singleRepo(t *testing.T) {
	base := t.TempDir()
	makeRepo(t, base, "ProjectA")
	repos, skipped, err := scanDirs(base, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("repos: got %d, want 1: %v", len(repos), repos)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped: got %d, want 0: %v", len(skipped), skipped)
	}
}

func TestScanDirs_mixedChildren(t *testing.T) {
	base := t.TempDir()
	makeRepo(t, base, "ProjectA")
	makeDir(t, base, "not-a-repo")
	repos, skipped, err := scanDirs(base, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("repos: got %d, want 1: %v", len(repos), repos)
	}
	if len(skipped) != 1 {
		t.Errorf("skipped: got %d, want 1: %v", len(skipped), skipped)
	} else if !strings.HasSuffix(skipped[0], "not-a-repo") {
		t.Errorf("unexpected skipped dir: %s", skipped[0])
	}
}

func TestScanDirs_nestedRepo(t *testing.T) {
	// tools/ is a plain dir at depth 1; tools/ProjectB is a repo at depth 2
	base := t.TempDir()
	tools := makeDir(t, base, "tools")
	makeRepo(t, tools, "ProjectB")
	repos, skipped, err := scanDirs(base, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("repos: got %d, want 1: %v", len(repos), repos)
	}
	// tools/ itself is a direct child with no .git but has children with repos,
	// so it should NOT appear as skipped (we continue into it)
	if len(skipped) != 0 {
		t.Errorf("skipped: got %d, want 0: %v", len(skipped), skipped)
	}
}

func TestScanDirs_depthLimit(t *testing.T) {
	// Repo at depth 3 should not be found when maxDepth=2
	base := t.TempDir()
	a := makeDir(t, base, "a")
	b := makeDir(t, a, "b")
	makeRepo(t, b, "TooDeep")
	repos, _, err := scanDirs(base, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Errorf("expected no repos at depth 3, got: %v", repos)
	}
}

func TestScanDirs_repoNotDescendedInto(t *testing.T) {
	// A repo containing a nested .git should only appear once
	base := t.TempDir()
	outer := makeRepo(t, base, "outer")
	makeRepo(t, outer, "inner") // nested — should be ignored
	repos, _, err := scanDirs(base, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("expected 1 repo (outer), got %d: %v", len(repos), repos)
	}
}

func TestScanDirs_emptyBase(t *testing.T) {
	base := t.TempDir()
	repos, skipped, err := scanDirs(base, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 || len(skipped) != 0 {
		t.Errorf("expected empty results, got repos=%v skipped=%v", repos, skipped)
	}
}

func TestScanDirs_unreadableDirSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Chmod(0000) does not reliably prevent directory reads on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("running as root; permission restrictions do not apply")
	}
	base := t.TempDir()
	unreadable := filepath.Join(base, "locked")
	if err := os.MkdirAll(unreadable, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unreadable, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0755) })

	repos, skipped, err := scanDirs(base, 2)
	if err != nil {
		t.Errorf("expected no error for unreadable directory, got: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected no repos, got: %v", repos)
	}
	if len(skipped) != 0 {
		t.Errorf("unreadable dir must be silently ignored, not appear in skipped: %v", skipped)
	}
}

func TestScanDirs_baseDirIsRepo(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}
	repos, skipped, err := scanDirs(base, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0] != base {
		t.Errorf("repos: got %v, want [%s]", repos, base)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped: got %v, want []", skipped)
	}
}
