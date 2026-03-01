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
