package cmd

import (
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

func TestExecuteReturnsNoError(t *testing.T) {
	// Execute with no args should print help and return nil
	rootCmd.SetArgs([]string{})
	if err := Execute(); err != nil {
		t.Errorf("Execute() returned unexpected error: %v", err)
	}
}
