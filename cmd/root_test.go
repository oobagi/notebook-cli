package cmd

import (
	"bytes"
	"testing"
)

func TestRootCommandExists(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}
	if rootCmd.Use != "notebook" {
		t.Errorf("expected Use to be 'notebook', got %q", rootCmd.Use)
	}
}

func TestRootHasSubcommands(t *testing.T) {
	expected := []string{"version", "list", "new", "delete", "search"}
	commands := rootCmd.Commands()

	found := make(map[string]bool)
	for _, c := range commands {
		found[c.Name()] = true
	}

	for _, name := range expected {
		if !found[name] {
			t.Errorf("expected subcommand %q to be registered", name)
		}
	}
}

func TestDirFlagExists(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("dir")
	if flag == nil {
		t.Fatal("expected --dir persistent flag to exist")
	}
}

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("version command error: %v", err)
	}
}

func TestExecuteReturnsNoError(t *testing.T) {
	rootCmd.SetArgs([]string{})
	if err := Execute(); err != nil {
		t.Errorf("Execute() returned unexpected error: %v", err)
	}
}
