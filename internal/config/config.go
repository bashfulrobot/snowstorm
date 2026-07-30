// Package config loads snowstorm's optional tool-wide defaults file,
// ~/.snowstorm/config.toml. It is a sibling of ~/.snowstorm/queries/, not a
// per-saved-query file -- those are <name>.toml files with unrelated
// name/description/sql fields (see internal/cli's SavedQuery).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds default flag values a user can set once instead of passing
// them on every invocation. Every field is optional; its zero value means
// "not set", and callers fall back to their own existing defaults.
type Config struct {
	Connection string `toml:"connection"`
	Format     string `toml:"format"`
	Human      bool   `toml:"human"`
	QueryDir   string `toml:"query_dir"`
}

// configRelPath is where the config file lives under the user's home
// directory, mirroring the ~/.snowstorm/queries layout used elsewhere.
const configRelPath = ".snowstorm/config.toml"

// Load reads ~/.snowstorm/config.toml. A missing file is not an error -- it
// returns a zero-value Config so a user with no config file sees no change
// in behavior. If the home directory itself can't be resolved, that's
// treated the same way (no config to load), matching the leniency of
// queryDir()'s own os.UserHomeDir() fallback elsewhere in this codebase.
func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, nil
	}
	return LoadFrom(filepath.Join(home, filepath.FromSlash(configRelPath)))
}

// LoadFrom reads and parses the config file at path. It's split out from
// Load so tests can point at a temp file instead of the real home
// directory. A malformed file is a real error -- unlike a missing file, a
// broken config should be surfaced to the user, not silently ignored.
func LoadFrom(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if _, err := toml.Decode(string(b), &cfg); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return cfg, nil
}
