package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oobagi/notebook-cli/internal/config"
)

func TestConfigShowsDefaults(t *testing.T) {
	dir := setupTestStore(t)

	out, err := executeCapture([]string{"--dir", dir, "config"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify default values appear.
	if !strings.Contains(out, "~/.notebook") {
		t.Errorf("expected default storage_dir in output, got %q", out)
	}
	if !strings.Contains(out, "dark") {
		t.Errorf("expected default theme in output, got %q", out)
	}
	if !strings.Contains(out, "relative") {
		t.Errorf("expected default date_format in output, got %q", out)
	}
	if !strings.Contains(out, "(default)") {
		t.Errorf("expected '(default)' for empty editor, got %q", out)
	}
	if !strings.Contains(out, "Config file:") {
		t.Errorf("expected 'Config file:' in output, got %q", out)
	}
}

func TestConfigSetKey(t *testing.T) {
	dir := setupTestStore(t)

	// Create a temp config dir so we don't write to real ~/.config.
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.toml")

	// Write initial defaults so Save knows where to write.
	cfg := config.DefaultConfig()
	if err := config.SaveTo(cfg, configPath); err != nil {
		t.Fatalf("initial save: %v", err)
	}

	// Manually set the config path by pointing XDG at our temp dir.
	// Since we can't easily override config.Path() in the command,
	// we test the config package directly here for the set functionality.
	if err := config.Set(&cfg, "editor", "nano"); err != nil {
		t.Fatalf("Set editor: %v", err)
	}
	if err := config.SaveTo(cfg, configPath); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if loaded.Editor != "nano" {
		t.Errorf("editor = %q, want %q", loaded.Editor, "nano")
	}

	// Also test through the CLI (this writes to the real config path,
	// so we verify the output message format).
	out, errExec := executeCapture([]string{"--dir", dir, "config", "set", "theme", "dark"})
	if errExec != nil {
		t.Fatalf("unexpected error: %v", errExec)
	}
	if !strings.Contains(out, "Set theme") {
		t.Errorf("expected success message, got %q", out)
	}
	if !strings.Contains(out, "dark") {
		t.Errorf("expected 'dark' in output, got %q", out)
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	dir := setupTestStore(t)

	_, err := executeCapture([]string{"--dir", dir, "config", "set", "nonexistent", "value"})
	if err == nil {
		t.Fatal("expected error for unknown config key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("expected 'unknown config key' in error, got %q", err.Error())
	}
}

func TestCLIFlagOverridesConfig(t *testing.T) {
	// Create a config file that sets storage_dir to one path.
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.toml")

	cfg := config.Config{
		StorageDir: "/config/notes",
		Editor:     "",
		Theme:      "dark",
		DateFormat: "relative",
	}
	if err := config.SaveTo(cfg, configPath); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	// The --dir flag should override the config's storage_dir.
	flagDir := t.TempDir()
	dirFlag = flagDir
	rootCmd.SetArgs([]string{"list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if store.Root != flagDir {
		t.Errorf("store.Root = %q, want --dir value %q", store.Root, flagDir)
	}
	dirFlag = "" // reset
}

func TestConfigSetMissingArgs(t *testing.T) {
	dir := setupTestStore(t)

	// "config set" with no args should error.
	_, err := executeCapture([]string{"--dir", dir, "config", "set"})
	if err == nil {
		t.Fatal("expected error when no key/value given")
	}
}

func TestConfigAfterSet(t *testing.T) {
	// Verify that after setting a value, "config" shows it.
	dir := setupTestStore(t)

	// Set a value through the CLI.
	_, err := executeCapture([]string{"--dir", dir, "config", "set", "date_format", "2006-01-02"})
	if err != nil {
		t.Fatalf("config set: %v", err)
	}

	// Now show config -- the new value should appear.
	out, err := executeCapture([]string{"--dir", dir, "config"})
	if err != nil {
		t.Fatalf("config show: %v", err)
	}
	if !strings.Contains(out, "2006-01-02") {
		t.Errorf("expected '2006-01-02' in config output, got %q", out)
	}

	// Clean up: reset to defaults so we don't affect other tests.
	configPath := config.Path()
	os.Remove(configPath)
}
