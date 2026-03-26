package cmd

import (
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	out, err := executeCapture([]string{"version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "notebook dev") {
		t.Errorf("expected %q in output, got %q", "notebook dev", out)
	}
}

func TestVersionVariable(t *testing.T) {
	if Version != "dev" {
		t.Errorf("default Version should be %q, got %q", "dev", Version)
	}
}
