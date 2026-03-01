package cmd

import (
	"bytes"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected help output, got empty string")
	}
}

func TestRootCommandUsage(t *testing.T) {
	if rootCmd.Use != "git-scoper" {
		t.Errorf("expected Use to be 'git-scoper', got %q", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if rootCmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}
