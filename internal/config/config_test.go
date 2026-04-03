package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.StorageDir != "~/.notebook" {
		t.Errorf("StorageDir = %q, want %q", cfg.StorageDir, "~/.notebook")
	}
	if cfg.Editor != "" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "")
	}
	if cfg.Theme != "auto" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "auto")
	}
	if cfg.DateFormat != "relative" {
		t.Errorf("DateFormat = %q, want %q", cfg.DateFormat, "relative")
	}
}

func TestLoadNoFile(t *testing.T) {
	// Point at a path that does not exist.
	cfg, err := LoadFrom("/tmp/notebook-test-nonexistent/config.toml")
	if err != nil {
		t.Fatalf("LoadFrom non-existent path should not error, got: %v", err)
	}

	// Should return defaults.
	want := DefaultConfig()
	if cfg != want {
		t.Errorf("LoadFrom non-existent = %+v, want defaults %+v", cfg, want)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `storage_dir = "/custom/notes"
editor = "nano"
theme = "dark"
date_format = "2006-01-02"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.StorageDir != "/custom/notes" {
		t.Errorf("StorageDir = %q, want %q", cfg.StorageDir, "/custom/notes")
	}
	if cfg.Editor != "nano" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "nano")
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme = %q, want %q", cfg.Theme, "dark")
	}
	if cfg.DateFormat != "2006-01-02" {
		t.Errorf("DateFormat = %q, want %q", cfg.DateFormat, "2006-01-02")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.toml")

	cfg := Config{
		StorageDir: "/my/notes",
		Editor:     "vim",
		Theme:      "light",
		DateFormat: "relative",
	}

	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom after save: %v", err)
	}

	if loaded != cfg {
		t.Errorf("round-trip failed: got %+v, want %+v", loaded, cfg)
	}
}

func TestLoadFromEmptyPath(t *testing.T) {
	cfg, err := LoadFrom("")
	if err != nil {
		t.Fatalf("LoadFrom empty path should not error, got: %v", err)
	}
	want := DefaultConfig()
	if cfg != want {
		t.Errorf("LoadFrom empty = %+v, want defaults %+v", cfg, want)
	}
}

func TestSetValidKeys(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		key   string
		value string
		check func() string
	}{
		{"storage_dir", "/new/path", func() string { return cfg.StorageDir }},
		{"editor", "nano", func() string { return cfg.Editor }},
		{"theme", "dark", func() string { return cfg.Theme }},
		{"date_format", "2006-01-02", func() string { return cfg.DateFormat }},
	}

	for _, tt := range tests {
		if err := Set(&cfg, tt.key, tt.value); err != nil {
			t.Errorf("Set(%q, %q) error: %v", tt.key, tt.value, err)
		}
		if got := tt.check(); got != tt.value {
			t.Errorf("after Set(%q, %q), got %q", tt.key, tt.value, got)
		}
	}
}

func TestSetUnknownKey(t *testing.T) {
	cfg := DefaultConfig()
	err := Set(&cfg, "nonexistent", "value")
	if err == nil {
		t.Fatal("Set with unknown key should return error")
	}
}

func TestGetValidKeys(t *testing.T) {
	cfg := Config{
		StorageDir: "/notes",
		Editor:     "nano",
		Theme:      "dark",
		DateFormat: "relative",
	}

	tests := []struct {
		key  string
		want string
	}{
		{"storage_dir", "/notes"},
		{"editor", "nano"},
		{"theme", "dark"},
		{"date_format", "relative"},
	}

	for _, tt := range tests {
		got, err := Get(cfg, tt.key)
		if err != nil {
			t.Errorf("Get(%q) error: %v", tt.key, err)
		}
		if got != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestGetUnknownKey(t *testing.T) {
	cfg := DefaultConfig()
	_, err := Get(cfg, "nonexistent")
	if err == nil {
		t.Fatal("Get with unknown key should return error")
	}
}

func TestPath(t *testing.T) {
	p := Path()
	if p == "" {
		t.Skip("could not determine home directory")
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "notebook", "config.toml")
	if p != want {
		t.Errorf("Path() = %q, want %q", p, want)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "config.toml")

	cfg := DefaultConfig()
	if err := SaveTo(cfg, path); err != nil {
		t.Fatalf("SaveTo should create nested dirs: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file should exist after SaveTo")
	}
}

func TestLoadPartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Only set one field -- others should keep defaults.
	content := `editor = "code"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	if cfg.Editor != "code" {
		t.Errorf("Editor = %q, want %q", cfg.Editor, "code")
	}
	// Defaults should still apply for fields not in the file.
	if cfg.StorageDir != "~/.notebook" {
		t.Errorf("StorageDir = %q, want default %q", cfg.StorageDir, "~/.notebook")
	}
	if cfg.Theme != "auto" {
		t.Errorf("Theme = %q, want default %q", cfg.Theme, "auto")
	}
	if cfg.DateFormat != "relative" {
		t.Errorf("DateFormat = %q, want default %q", cfg.DateFormat, "relative")
	}
}

func TestLegacyUIThemeMigration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Simulate a legacy config file with ui_theme set and theme at default.
	content := `theme = "auto"
ui_theme = "ocean"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if loaded.Theme != "ocean" {
		t.Errorf("after migration Theme = %q, want %q", loaded.Theme, "ocean")
	}
}

func TestLegacyUIThemeNoOverrideExplicitTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// If theme is already set to a non-auto value, ui_theme should not override it.
	content := `theme = "forest"
ui_theme = "ocean"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if loaded.Theme != "forest" {
		t.Errorf("Theme = %q, want %q (should not be overridden by legacy ui_theme)", loaded.Theme, "forest")
	}
}
