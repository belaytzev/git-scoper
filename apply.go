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
