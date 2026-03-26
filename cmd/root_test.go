package cmd

import (
	"testing"
)

func TestRootCommandUseField(t *testing.T) {
	if rootCmd.Use != "notebook" {
		t.Errorf("expected Use to be %q, got %q", "notebook", rootCmd.Use)
	}
}

func TestRootCommandSilenceErrors(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Error("expected SilenceErrors to be true")
	}
}

func TestExecuteReturnsNoError(t *testing.T) {
	if err := Execute(); err != nil {
		t.Errorf("Execute() returned unexpected error: %v", err)
	}
}
