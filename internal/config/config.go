// Package config loads and persists seshy's user settings.
//
// Settings live in a small JSON file at ~/.config/seshy/config.json (or under
// $XDG_CONFIG_HOME when set). A missing file is not an error — callers get the
// zero value, which preserves seshy's default "show everything" behavior.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds seshy's persisted settings. The zero value is the default
// behavior: nothing is hidden.
type Config struct {
	// HideClaudeHeadless drops Claude sessions started headless (claude -p /
	// the Agent SDK, entrypoint "sdk-cli") from all listings.
	HideClaudeHeadless bool `json:"hideClaudeHeadless"`
	// HideCodexExec drops Codex sessions started via `codex exec` (source
	// "exec") from all listings.
	HideCodexExec bool `json:"hideCodexExec"`
}

// Dir returns the directory holding seshy's config file.
func Dir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "seshy")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "seshy")
}

// Path returns the full path to the config file.
func Path() string { return filepath.Join(Dir(), "config.json") }

// Load reads the config file. A missing or unreadable file yields the zero
// value (defaults) and no error; only malformed JSON is reported.
func Load() (Config, error) {
	var c Config
	b, err := os.ReadFile(Path())
	if err != nil {
		return c, nil // absent file → defaults
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Save writes c to the config file, creating the directory as needed.
func Save(c Config) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(Path(), b, 0o644)
}
