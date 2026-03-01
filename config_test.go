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
