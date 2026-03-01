package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// applyConfig sets user.name and user.email in the given git repo's local config.
func applyConfig(repoPath, name, email string) error {
	cmd := exec.Command("git", "-C", repoPath, "config", "--local", "--", "user.name", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("git config user.name failed: %s", msg)
		}
		return fmt.Errorf("git config user.name failed: %w", err)
	}
	cmd = exec.Command("git", "-C", repoPath, "config", "--local", "--", "user.email", email)
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("git config user.email failed: %s", msg)
		}
		return fmt.Errorf("git config user.email failed: %w", err)
	}
	return nil
}
