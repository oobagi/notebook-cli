package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all user-configurable settings.
type Config struct {
	StorageDir string `toml:"storage_dir"`
	Editor     string `toml:"editor"`
	Theme      string `toml:"theme"`       // "auto" or any preset name
	DateFormat string `toml:"date_format"` // "relative" or Go time format
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		StorageDir: "~/.notebook",
		Editor:     "",
		Theme:      "auto",
		DateFormat: "relative",
	}
}

// ValidKeys returns the set of keys that can be set via "config set".
var ValidKeys = map[string]bool{
	"storage_dir": true,
	"editor":      true,
	"theme":       true,
	"date_format": true,
}

// Path returns the path to the config file: ~/.config/notebook/config.toml.
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "notebook", "config.toml")
}

// Load reads the config file if it exists, otherwise returns defaults.
func Load() (Config, error) {
	return LoadFrom(Path())
}

// LoadFrom reads the config file at the given path.
// If the file does not exist, defaults are returned without error.
func LoadFrom(path string) (Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	// Migrate deprecated ui_theme → theme.
	var legacy struct {
		UITheme string `toml:"ui_theme"`
	}
	_ = toml.Unmarshal(data, &legacy)
	if legacy.UITheme != "" && legacy.UITheme != "auto" && (cfg.Theme == "" || cfg.Theme == "auto") {
		cfg.Theme = legacy.UITheme
	}

	return cfg, nil
}

// Save writes the config to the default path.
func Save(cfg Config) error {
	return SaveTo(cfg, Path())
}

// SaveTo writes the config to the given path, creating directories as needed.
func SaveTo(cfg Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Set updates a single key in the config struct. Returns an error for unknown keys.
func Set(cfg *Config, key, value string) error {
	switch key {
	case "storage_dir":
		cfg.StorageDir = value
	case "editor":
		cfg.Editor = value
	case "theme":
		cfg.Theme = value
	case "date_format":
		cfg.DateFormat = value
	default:
		return fmt.Errorf("unknown config key: %q", key)
	}
	return nil
}

// Get returns the value of a config key. Returns an error for unknown keys.
func Get(cfg Config, key string) (string, error) {
	switch key {
	case "storage_dir":
		return cfg.StorageDir, nil
	case "editor":
		return cfg.Editor, nil
	case "theme":
		return cfg.Theme, nil
	case "date_format":
		return cfg.DateFormat, nil
	default:
		return "", fmt.Errorf("unknown config key: %q", key)
	}
}
