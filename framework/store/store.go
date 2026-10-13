// Package store keeps small JSON files in ~/.config/<app>.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Dir is ~/.config/<app>. $HOME falls back to /home/root, the home on reMarkable tablets.
func Dir(app string) string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/root"
	}
	return filepath.Join(home, ".config", app)
}

// Load decodes the JSON file at path into v. A missing file returns an error
// matching os.ErrNotExist and leaves v alone.
func Load(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Save writes v as JSON to path. It writes a temporary file and renames it, so
// a crash or power loss never leaves a half-written file behind.
func Save(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
