package wordlecount

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	weeklyFile   = "weekly_scores.json"
	lifetimeFile = "lifetime_tally.json"
)

// loadJSON reads path into v, leaving v untouched if the file doesn't exist.
func loadJSON(path string, v any) error {
	data, err := os.ReadFile(path)

	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	return nil
}

// saveJSON writes v to path via a temporary file, so a crash mid-write can't
// leave a truncated file behind.
func saveJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ") // todo: validate

	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
