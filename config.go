package main

import (
	"fmt"
	"os"
	"path/filepath"
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

// resolveConfig tries <baseDir>/gitconfig first, then ~/.gitconfig.
// If the local gitconfig file exists but is malformed or inaccessible, it returns an error
// immediately rather than silently falling back to ~/.gitconfig.
func resolveConfig(baseDir string) (*Config, error) {
	local := filepath.Join(baseDir, "gitconfig")
	info, statErr := os.Stat(local)
	if statErr == nil {
		if info.IsDir() {
			return nil, fmt.Errorf("config path %s is a directory, not a file", local)
		}
		// File exists — parse it; any error is fatal (don't silently fall through)
		return parseKeyValue(local)
	}
	if !os.IsNotExist(statErr) {
		// Permission denied or other non-NotExist error — fail explicitly rather than
		// silently applying the wrong identity from ~/.gitconfig.
		return nil, fmt.Errorf("cannot access config file %s: %w", local, statErr)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	cfg, err := parseGitconfig(filepath.Join(home, ".gitconfig"))
	if err != nil {
		return nil, fmt.Errorf("no usable config found in %s or ~/.gitconfig: %w", baseDir, err)
	}
	return cfg, nil
}

// stripInlineComment removes a trailing # or ; comment from a git INI value.
// Git allows inline comments after the value: name = Jane  # comment
// This handles only unquoted values; quoted values are not used by user.name/email in practice.
func stripInlineComment(v string) string {
	for i, ch := range v {
		if (ch == '#' || ch == ';') && i > 0 && (v[i-1] == ' ' || v[i-1] == '\t') {
			return v[:i]
		}
	}
	return v
}

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
		if strings.HasPrefix(trimmed, "[") {
			// Extract name between '[' and ']', ignoring inline comments after ']'
			// Handles "[ user ]", "[user] # comment", but not subsections like [user "work"]
			closing := strings.Index(trimmed, "]")
			if closing == -1 {
				inUser = false
				continue
			}
			inner := strings.TrimSpace(trimmed[1:closing])
			inUser = strings.EqualFold(inner, "user") && !strings.Contains(inner, "\"")
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
			cfg.Name = strings.TrimSpace(stripInlineComment(v))
		case "email":
			cfg.Email = strings.TrimSpace(stripInlineComment(v))
		}
	}
	if cfg.Name == "" || cfg.Email == "" {
		return nil, fmt.Errorf("%s: [user] section missing name or email", path)
	}
	return cfg, nil
}
