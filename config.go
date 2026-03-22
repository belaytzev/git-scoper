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
		return nil, fmt.Errorf("cannot read config file %s: %w", path, err)
	}
	cfg := &Config{}
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "name":
			cfg.Name = strings.TrimSpace(stripInlineComment(v))
		case "email":
			cfg.Email = strings.TrimSpace(stripInlineComment(v))
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

// stripInlineComment removes a trailing # or ; comment from an unquoted git INI value.
// Git allows inline comments after the value: name = Jane  # comment
func stripInlineComment(v string) string {
	runes := []rune(v)
	for i, ch := range runes {
		if (ch == '#' || ch == ';') && i > 0 && (runes[i-1] == ' ' || runes[i-1] == '\t') {
			return string(runes[:i])
		}
	}
	return v
}

// unquoteGitValue normalises a git INI value from the right-hand side of a key=value
// line.  If the trimmed value starts with a double-quote, the quoted string is parsed
// up to the first unescaped closing quote (per git-config spec), ignoring any trailing
// inline comment after the closing quote.  Standard git-config escape sequences
// (\", \\, \n, \t, \b) are expanded.  Otherwise inline comments are stripped.
func unquoteGitValue(v string) string {
	trimmed := strings.TrimSpace(v)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return strings.TrimSpace(stripInlineComment(v))
	}
	// Scan for the closing quote, expanding escape sequences as we go.
	var buf strings.Builder
	i := 1 // skip opening quote
	for i < len(trimmed) {
		ch := trimmed[i]
		if ch == '"' {
			// Closing quote found; any trailing content (e.g. inline comment) is ignored.
			return buf.String()
		}
		if ch == '\\' && i+1 < len(trimmed) {
			i++
			switch trimmed[i] {
			case '"':
				buf.WriteByte('"')
			case '\\':
				buf.WriteByte('\\')
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case 'b':
				buf.WriteByte('\b')
			default:
				// Unknown escape: preserve the backslash and the following character.
				buf.WriteByte('\\')
				buf.WriteByte(trimmed[i])
			}
		} else {
			buf.WriteByte(ch)
		}
		i++
	}
	// No closing quote found (malformed); return whatever was accumulated.
	return buf.String()
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
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "name":
			cfg.Name = unquoteGitValue(v)
		case "email":
			cfg.Email = unquoteGitValue(v)
		}
	}
	if cfg.Name == "" || cfg.Email == "" {
		return nil, fmt.Errorf("%s: [user] section missing name or email", path)
	}
	return cfg, nil
}
