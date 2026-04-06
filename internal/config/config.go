package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// HideChecked controls visibility of checked checklist items.
type HideChecked string

const (
	HideCheckedOff      HideChecked = "off"       // show all items (default)
	HideCheckedViewOnly HideChecked = "view-only"  // hide in view mode only
	HideCheckedOn       HideChecked = "on"         // hide in both modes
)

// CycleHideChecked returns the next value in the off -> view-only -> on cycle.
func CycleHideChecked(current HideChecked) HideChecked {
	switch current {
	case HideCheckedOff:
		return HideCheckedViewOnly
	case HideCheckedViewOnly:
		return HideCheckedOn
	default:
		return HideCheckedOff
	}
}

// Config holds all user-configurable settings.
type Config struct {
	StorageDir   string      `toml:"storage_dir"`
	Editor       string      `toml:"editor"`
	Theme        string      `toml:"theme"`         // any preset name
	DateFormat   string      `toml:"date_format"`   // "relative" or Go time format
	HideChecked  HideChecked `toml:"hide_checked"`  // "off", "view-only", or "on"
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		StorageDir:  "~/.notebook",
		Editor:      "",
		Theme:       "dark",
		DateFormat:  "relative",
		HideChecked: HideCheckedOff,
	}
}

// ValidKeys returns the set of keys that can be set via "config set".
var ValidKeys = map[string]bool{
	"storage_dir":   true,
	"editor":        true,
	"theme":         true,
	"date_format":   true,
	"show_hints":    true,
	"hide_checked":  true,
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
	case "hide_checked":
		switch HideChecked(value) {
		case HideCheckedOff, HideCheckedViewOnly, HideCheckedOn:
			cfg.HideChecked = HideChecked(value)
		default:
			return fmt.Errorf("hide_checked must be \"off\", \"view-only\", or \"on\"")
		}
	case "show_hints":
		if value != "true" {
			return fmt.Errorf("show_hints only supports \"true\" to re-enable hints")
		}
		ResetHints()
		return nil
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
	case "hide_checked":
		return string(cfg.HideChecked), nil
	case "show_hints":
		dismissed := LoadDismissedHints()
		if len(dismissed) == 0 {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("unknown config key: %q", key)
	}
}
