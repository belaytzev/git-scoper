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
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
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

func TestParseGitconfig_inlineComments(t *testing.T) {
	content := "[user]\n\tname = Jane Doe  # work account\n\temail = jane@co.com ; note\n"
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
	if err := os.WriteFile(path, []byte("name=Local User\nemail=local@co.com\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := resolveConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Local User" {
		t.Errorf("got name %q, want %q", cfg.Name, "Local User")
	}
	if cfg.Email != "local@co.com" {
		t.Errorf("got email %q, want %q", cfg.Email, "local@co.com")
	}
}

func TestResolveConfig_fallbackToGitconfig(t *testing.T) {
	dir := t.TempDir() // no local gitconfig file
	home := t.TempDir()
	t.Setenv("HOME", home)
	content := "[core]\n\tautocrlf = false\n[user]\n\tname = Fallback User\n\temail = fallback@co.com\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := resolveConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Fallback User" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "Fallback User")
	}
	if cfg.Email != "fallback@co.com" {
		t.Errorf("Email: got %q, want %q", cfg.Email, "fallback@co.com")
	}
}

func TestResolveConfig_localMalformedErrors(t *testing.T) {
	dir := t.TempDir()
	// Local file exists but is missing email — should error, not fall through
	if err := os.WriteFile(filepath.Join(dir, "gitconfig"), []byte("name=Only Name\n"), 0644); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Even with a valid ~/.gitconfig, the malformed local file should cause an error
	content := "[user]\n\tname = Should Not Use\n\temail = should@not.use\n"
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := resolveConfig(dir)
	if err == nil {
		t.Fatal("expected error for malformed local gitconfig, not silent fallthrough")
	}
}

func TestResolveConfig_noConfigAnywhere(t *testing.T) {
	dir := t.TempDir() // no gitconfig, home will have no .gitconfig either
	t.Setenv("HOME", t.TempDir()) // temp home with no .gitconfig
	_, err := resolveConfig(dir)
	if err == nil {
		t.Fatal("expected error when no config found anywhere")
	}
}

func TestParseKeyValue_inlineComments(t *testing.T) {
	path := writeTempFile(t, "name=Jane Doe  # work account\nemail=jane@co.com ; note\n")
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

func TestParseGitconfig_quotedValues(t *testing.T) {
	// git writes quoted values when the name/email contains special characters
	content := "[user]\n\tname = \"Jane O'Brien\"\n\temail = \"jane@co.com\"\n"
	path := writeTempFile(t, content)
	cfg, err := parseGitconfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Jane O'Brien" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "Jane O'Brien")
	}
	if cfg.Email != "jane@co.com" {
		t.Errorf("Email: got %q, want %q", cfg.Email, "jane@co.com")
	}
}

func TestParseGitconfig_quotedValueWithHash(t *testing.T) {
	// A # inside a quoted value must not be treated as an inline comment
	content := "[user]\n\tname = \"Jane # Doe\"\n\temail = \"jane@co.com\"\n"
	path := writeTempFile(t, content)
	cfg, err := parseGitconfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Jane # Doe" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "Jane # Doe")
	}
}

func TestParseGitconfig_quotedValueEscapedQuote(t *testing.T) {
	content := "[user]\n\tname = \"Jane \\\"J\\\" Doe\"\n\temail = \"jane@co.com\"\n"
	path := writeTempFile(t, content)
	cfg, err := parseGitconfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != `Jane "J" Doe` {
		t.Errorf("Name: got %q, want %q", cfg.Name, `Jane "J" Doe`)
	}
}

func TestParseGitconfig_quotedValueWithTrailingComment(t *testing.T) {
	// Quoted value followed by an inline comment — quotes must be stripped, comment ignored.
	content := "[user]\n\tname = \"Jane Doe\" # work account\n\temail = \"jane@co.com\" ; note\n"
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

func TestParseGitconfig_quotedValueEscapeSequences(t *testing.T) {
	// Standard git-config escape sequences inside quoted values.
	content := "[user]\n\tname = \"Jane\\\\Doe\"\n\temail = \"jane@co.com\"\n"
	path := writeTempFile(t, content)
	cfg, err := parseGitconfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != `Jane\Doe` {
		t.Errorf("Name: got %q, want %q", cfg.Name, `Jane\Doe`)
	}
}

func TestParseGitconfig_subsectionIgnored(t *testing.T) {
	// [user "work"] subsection must not be read; only plain [user] applies
	content := "[user \"work\"]\n\tname = Work User\n\temail = work@co.com\n[user]\n\tname = Main User\n\temail = main@co.com\n"
	path := writeTempFile(t, content)
	cfg, err := parseGitconfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != "Main User" {
		t.Errorf("Name: got %q, want %q", cfg.Name, "Main User")
	}
	if cfg.Email != "main@co.com" {
		t.Errorf("Email: got %q, want %q", cfg.Email, "main@co.com")
	}
}
