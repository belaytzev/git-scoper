# git-scoper Go CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a single Go binary that scans a base directory for git repos and applies `user.name`/`user.email` to each one, reading config from `<base-dir>/gitconfig` or `~/.gitconfig`.

**Architecture:** Flat `package main` across multiple `.go` files — no external dependencies, no subcommands. Parallel worker pool via goroutines + channels. Config falls back from local file to `~/.gitconfig`.

**Tech Stack:** Go stdlib only — `os/exec` for git calls, `filepath.WalkDir` for discovery, `flag` for CLI args, `sync` for parallelism.

---

## Design Reference (from brainstorming)

**CLI usage:**
```
git-scoper                    # scan current directory
git-scoper /Work/Acme         # scan given directory
git-scoper --depth 3 /Work/Acme
```

**Config resolution order:**
1. `<base-dir>/gitconfig` — key=value format: `name=Jane Doe` / `email=jane@co.com`
2. `~/.gitconfig` — standard INI format, reads `[user]` section
3. Error if neither provides both `name` and `email`

**Output format:**
```
Config: name=Jane Doe, email=jane@co.com
Scanning: /Work/Acme (depth 2)
------------------------
Updated: ProjectA
Updated: tools/ProjectB
Skipped: not-a-repo
------------------------
Done. 2 updated, 1 skipped.
```

- `Updated` — git repo where config was applied successfully
- `Skipped` — direct child of base dir that has no `.git` (depth-2+ non-repos are silent)
- `Failed` — git repo where `git config` command failed (shown with error message)

**File layout:**
```
git-scoper/
├── go.mod
├── main.go       # entry point, flag parsing, output
├── config.go     # config reading (key=value + INI formats)
├── discover.go   # repo/dir scanning
├── apply.go      # git config applicator
└── run.go        # parallel worker pool
```

---

## Task 1: Project scaffolding

- [x] Initialize Go module (go.mod)
- [x] Create main.go skeleton with flag parsing and directory validation
- [x] Verify compilation passes

**Files:**
- Create: `go.mod`
- Create: `main.go`

**Step 1: Initialize the Go module**

```bash
go mod init github.com/belaytzev/git-scoper
```

Expected: `go.mod` created with `module github.com/belaytzev/git-scoper` and current Go version.

**Step 2: Create `main.go` skeleton**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	depth := flag.Int("depth", 2, "max directory depth to scan")
	workers := flag.Int("workers", 4, "parallel workers")
	flag.Parse()

	baseDir := "."
	if flag.NArg() > 0 {
		baseDir = flag.Arg(0)
	}
	baseDir, _ = filepath.Abs(baseDir)

	if _, err := os.Stat(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access directory %s\n", baseDir)
		os.Exit(1)
	}

	_ = depth
	_ = workers
	fmt.Println("git-scoper: not yet implemented")
}
```

**Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: no errors, binary produced.

**Step 4: Commit**

```bash
git add go.mod main.go
git commit -m "feat: scaffold go module and main entry point"
```

---

## Task 2: Key=value config reader

- [x] Create config_test.go with tests for parseKeyValue
- [x] Create config.go with Config struct and parseKeyValue function
- [x] Run tests and verify they pass

**Files:**
- Create: `config.go`
- Create: `config_test.go`

**Step 1: Write the failing tests**

Create `config_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "gitconfig")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestParseKeyValue_valid(t *testing.T) {
	path := writeTempFile(t, "name=Jane Doe\nemail=jane@co.com\n")
	cfg, err := parseKeyValue(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Jane Doe" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "Jane Doe")
	}
	if cfg.Email != "jane@co.com" {
		t.Errorf("Email: got %q, want %q", cfg.Email, "jane@co.com")
	}
}

func TestParseKeyValue_missingFile(t *testing.T) {
	_, err := parseKeyValue(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseKeyValue_missingEmail(t *testing.T) {
	path := writeTempFile(t, "name=Jane Doe\n")
	_, err := parseKeyValue(path)
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

func TestParseKeyValue_missingName(t *testing.T) {
	path := writeTempFile(t, "email=jane@co.com\n")
	_, err := parseKeyValue(path)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseKeyValue_extraKeys(t *testing.T) {
	path := writeTempFile(t, "name=Jane Doe\nemail=jane@co.com\nfoo=bar\n")
	cfg, err := parseKeyValue(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Jane Doe" || cfg.Email != "jane@co.com" {
		t.Errorf("unexpected cfg: %+v", cfg)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test -run TestParseKeyValue -v .
```

Expected: compile error — `parseKeyValue` and `Config` not defined.

**Step 3: Implement in `config.go`**

```go
package main

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Name  string
	Email string
}

// parseKeyValue reads a simple key=value config file (name= and email= lines).
func parseKeyValue(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file not found at %s: %w", path, err)
	}
	cfg := &Config{}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "name":
			cfg.Name = strings.TrimSpace(v)
		case "email":
			cfg.Email = strings.TrimSpace(v)
		}
	}
	if cfg.Name == "" || cfg.Email == "" {
		return nil, fmt.Errorf("%s must contain both name= and email=", path)
	}
	return cfg, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test -run TestParseKeyValue -v .
```

Expected: all 5 tests PASS.

**Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add key=value config file reader with tests"
```

---

## Task 3: ~/.gitconfig INI reader

- [x] Add TestParseGitconfig tests to config_test.go
- [x] Implement parseGitconfig in config.go
- [x] Run tests and verify they pass

**Files:**
- Modify: `config.go`
- Modify: `config_test.go`

**Step 1: Write the failing tests**

Add to `config_test.go`:

```go
func TestParseGitconfig_valid(t *testing.T) {
	content := "[core]\n\tautocrlf = false\n[user]\n\tname = Jane Doe\n\temail = jane@co.com\n"
	path := writeTempFile(t, content)
	cfg, err := parseGitconfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Jane Doe" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "Jane Doe")
	}
	if cfg.Email != "jane@co.com" {
		t.Errorf("Email: got %q, want %q", cfg.Email, "jane@co.com")
	}
}

func TestParseGitconfig_missingUserSection(t *testing.T) {
	path := writeTempFile(t, "[core]\n\tautocrlf = false\n")
	_, err := parseGitconfig(path)
	if err == nil {
		t.Fatal("expected error when [user] section is missing")
	}
}

func TestParseGitconfig_missingFile(t *testing.T) {
	_, err := parseGitconfig(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test -run TestParseGitconfig -v .
```

Expected: compile error — `parseGitconfig` not defined.

**Step 3: Implement `parseGitconfig` in `config.go`**

```go
// parseGitconfig reads the [user] section from a standard ~/.gitconfig INI file.
func parseGitconfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	cfg := &Config{}
	inUser := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[user]" {
			inUser = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inUser = false
			continue
		}
		if !inUser {
			continue
		}
		k, v, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "name":
			cfg.Name = strings.TrimSpace(v)
		case "email":
			cfg.Email = strings.TrimSpace(v)
		}
	}
	if cfg.Name == "" || cfg.Email == "" {
		return nil, fmt.Errorf("%s: [user] section missing name or email", path)
	}
	return cfg, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test -run TestParseGitconfig -v .
```

Expected: all 3 tests PASS. Also run full suite: `go test -v .` — all tests PASS.

**Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add ~/.gitconfig INI reader with tests"
```

---

## Task 4: Config resolution (base-dir first, then ~/.gitconfig)

- [x] Add TestResolveConfig tests to config_test.go
- [x] Implement resolveConfig in config.go
- [x] Run tests and verify they pass

**Files:**
- Modify: `config.go`
- Modify: `config_test.go`

**Step 1: Write the failing tests**

Add to `config_test.go`:

```go
func TestResolveConfig_localFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gitconfig")
	os.WriteFile(path, []byte("name=Local User\nemail=local@co.com\n"), 0644)
	cfg, err := resolveConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Local User" {
		t.Errorf("got name %q, want %q", cfg.Name, "Local User")
	}
}

func TestResolveConfig_noConfigAnywhere(t *testing.T) {
	dir := t.TempDir() // no gitconfig, home will have no .gitconfig either
	// We can't mock home dir easily, so just test the local-file branch failing
	// by verifying resolveConfig returns an error when base has no gitconfig
	// and we use a known-bad home. We test this indirectly: if local file
	// doesn't exist and ~/.gitconfig is missing, we get an error.
	// Use an env-var trick to redirect home:
	t.Setenv("HOME", t.TempDir()) // temp home with no .gitconfig
	_, err := resolveConfig(dir)
	if err == nil {
		t.Fatal("expected error when no config found anywhere")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test -run TestResolveConfig -v .
```

Expected: compile error — `resolveConfig` not defined.

**Step 3: Implement `resolveConfig` in `config.go`**

```go
// resolveConfig tries <baseDir>/gitconfig first, then ~/.gitconfig.
func resolveConfig(baseDir string) (*Config, error) {
	local := filepath.Join(baseDir, "gitconfig")
	if cfg, err := parseKeyValue(local); err == nil {
		return cfg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	cfg, err := parseGitconfig(filepath.Join(home, ".gitconfig"))
	if err != nil {
		return nil, fmt.Errorf("no usable config found in %s or ~/.gitconfig", baseDir)
	}
	return cfg, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test -run TestResolveConfig -v .
```

Expected: both tests PASS. Also run `go test -v .` — all tests PASS.

**Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add config resolution with local-file and ~/.gitconfig fallback"
```

---

## Task 5: Directory scanner (find repos and skipped dirs)

- [x] Create discover_test.go with tests for scanDirs
- [x] Create discover.go with scanDirs function
- [x] Run tests and verify they pass

**Files:**
- Create: `discover.go`
- Create: `discover_test.go`

**Step 1: Write the failing tests**

Create `discover_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func makeRepo(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	os.MkdirAll(filepath.Join(dir, ".git"), 0755)
	return dir
}

func makeDir(t *testing.T, parent, name string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	os.MkdirAll(dir, 0755)
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
```

**Step 2: Run tests to verify they fail**

```bash
go test -run TestScanDirs -v .
```

Expected: compile error — `scanDirs` not defined.

**Step 3: Implement `scanDirs` in `discover.go`**

```go
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
```

**Step 4: Run tests to verify they pass**

```bash
go test -run TestScanDirs -v .
```

Expected: all 6 tests PASS. Also run `go test -v .` — all tests PASS.

**Step 5: Commit**

```bash
git add discover.go discover_test.go
git commit -m "feat: add directory scanner with repo detection and depth limiting"
```

---

## Task 6: Git config applicator

**Files:**
- Create: `apply.go`
- Create: `apply_test.go`

**Step 1: Write the failing tests**

Create `apply_test.go`:

```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
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
	got := string(out[:len(out)-1]) // trim newline
	if got != "Jane Doe" {
		t.Errorf("user.name: got %q, want %q", got, "Jane Doe")
	}
	// Verify email was set
	out, err = exec.Command("git", "-C", repo, "config", "--local", "user.email").Output()
	if err != nil {
		t.Fatalf("git config read failed: %v", err)
	}
	got = string(out[:len(out)-1])
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
```

**Step 2: Run tests to verify they fail**

```bash
go test -run TestApplyConfig -v .
```

Expected: compile error — `applyConfig` not defined.

**Step 3: Implement `applyConfig` in `apply.go`**

```go
package main

import (
	"fmt"
	"os/exec"
)

// applyConfig sets user.name and user.email in the given git repo's local config.
func applyConfig(repoPath, name, email string) error {
	if err := exec.Command("git", "-C", repoPath, "config", "--local", "user.name", name).Run(); err != nil {
		return fmt.Errorf("git config user.name failed: %w", err)
	}
	if err := exec.Command("git", "-C", repoPath, "config", "--local", "user.email", email).Run(); err != nil {
		return fmt.Errorf("git config user.email failed: %w", err)
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test -run TestApplyConfig -v .
```

Expected: all 3 tests PASS. Also run `go test -v .` — all tests PASS.

**Step 5: Commit**

```bash
git add apply.go apply_test.go
git commit -m "feat: add git config applicator with tests"
```

---

## Task 7: Parallel worker pool

**Files:**
- Create: `run.go`
- Create: `run_test.go`

**Step 1: Write the failing tests**

Create `run_test.go`:

```go
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
```

**Step 2: Run tests to verify they fail**

```bash
go test -run TestRunAll -v .
```

Expected: compile error — `runAll` and `Result` not defined.

**Step 3: Implement `runAll` in `run.go`**

```go
package main

import (
	"sync"
)

// Result holds the outcome of applying config to a single repo.
type Result struct {
	Path string
	Err  error
}

// runAll applies cfg to each repo path using a pool of workers goroutines.
func runAll(repos []string, cfg *Config, workers int) []Result {
	if len(repos) == 0 {
		return nil
	}

	jobs := make(chan string, len(repos))
	for _, r := range repos {
		jobs <- r
	}
	close(jobs)

	out := make(chan Result, len(repos))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range jobs {
				out <- Result{Path: repo, Err: applyConfig(repo, cfg.Name, cfg.Email)}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	var results []Result
	for r := range out {
		results = append(results, r)
	}
	return results
}
```

**Step 4: Run tests to verify they pass**

```bash
go test -run TestRunAll -v .
```

Expected: all 4 tests PASS. Also run `go test -v .` — all tests PASS.

**Step 5: Commit**

```bash
git add run.go run_test.go
git commit -m "feat: add parallel worker pool for applying git config"
```

---

## Task 8: Wire everything in `main.go` + integration test

**Files:**
- Modify: `main.go`
- Create: `main_test.go`

**Step 1: Write the failing integration test**

Create `main_test.go`:

```go
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
```

**Step 2: Run test to verify it fails**

```bash
go test -run TestIntegration -v .
```

Expected: PASS already (the functions exist) — or fail due to `main.go` not yet wiring things. If it passes, proceed. If not, investigate the error.

**Step 3: Replace `main.go` with the full implementation**

```go
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

	baseDir := "."
	if flag.NArg() > 0 {
		baseDir = flag.Arg(0)
	}
	baseDir, _ = filepath.Abs(baseDir)

	if _, err := os.Stat(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot access directory %s\n", baseDir)
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

	// Sort results by path for deterministic output
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })

	updated, failed := 0, 0
	for _, r := range results {
		rel, _ := filepath.Rel(baseDir, r.Path)
		if r.Err != nil {
			fmt.Printf("Failed: %s (%v)\n", rel, r.Err)
			failed++
		} else {
			fmt.Printf("Updated: %s\n", rel)
			updated++
		}
	}

	sort.Strings(skipped)
	for _, s := range skipped {
		rel, _ := filepath.Rel(baseDir, s)
		fmt.Printf("Skipped: %s\n", rel)
	}

	fmt.Println("------------------------")
	msg := fmt.Sprintf("Done. %d updated", updated)
	if failed > 0 {
		msg += fmt.Sprintf(", %d failed", failed)
	}
	if len(skipped) > 0 {
		msg += fmt.Sprintf(", %d skipped", len(skipped))
	}
	fmt.Println(msg)
}
```

**Step 4: Run full test suite**

```bash
go test -v ./...
```

Expected: all tests PASS.

**Step 5: Build and smoke-test the binary**

```bash
go build -o git-scoper .
./git-scoper --help
```

Expected: usage text with `--depth` and `--workers` flags listed.

**Step 6: Commit**

```bash
git add main.go main_test.go
git commit -m "feat: wire full pipeline in main — config, scan, apply, output"
```

---

## Task 9: Verify acceptance criteria

**Step 1: Build check**

```bash
go build ./...
```

Expected: no errors.

**Step 2: Vet check**

```bash
go vet ./...
```

Expected: no issues reported.

**Step 3: Full test suite**

```bash
go test -v ./...
```

Expected: all tests PASS, no skipped.

**Step 4: Smoke test against a real directory**

```bash
./git-scoper ~/some/dir/with/repos
```

Verify output shows `Updated:` lines for each repo.

**Step 5: Dry-run / missing config error**

```bash
./git-scoper /tmp/empty-dir-with-no-gitconfig
```

Expected: `Error: no usable config found in ...` message, exit code 1.

**Step 6: Commit**

```bash
git commit -m "chore: verify acceptance criteria (all green)"
```

---

## Task 10: [Final] Documentation & cleanup

**Files:**
- Modify: `README.md`
- Move: `docs/plans/20260301-git-scoper-go-cli.md` → `docs/plans/completed/`

**Step 1: Update README.md**

```markdown
# git-scoper

Applies git `user.name` and `user.email` to every git repository found in a base directory.

## Install

```bash
go install github.com/belaytzev/git-scoper@latest
```

## Usage

```
git-scoper [flags] [base-dir]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--depth` | `2` | Max directory depth to scan |
| `--workers` | `4` | Parallel workers |

If `base-dir` is omitted, the current directory is used.

## Config

Reads from (in order):
1. `<base-dir>/gitconfig` — key=value format:
   ```
   name=Jane Doe
   email=jane@example.com
   ```
2. `~/.gitconfig` — standard git INI format, reads `[user]` section

## Example

```
$ git-scoper /Work/Acme
Config: name=Jane Doe, email=jane@acme.com
Scanning: /Work/Acme (depth 2)
------------------------
Updated: ProjectAlpha
Updated: tools/ProjectBeta
Skipped: docs
------------------------
Done. 2 updated, 1 skipped.
```
```

**Step 2: Move completed plan**

```bash
mkdir -p docs/plans/completed
mv docs/plans/20260301-git-scoper-go-cli.md docs/plans/completed/
```

**Step 3: Commit everything**

```bash
git add README.md docs/
git commit -m "docs: update README with usage and move plan to completed"
```
