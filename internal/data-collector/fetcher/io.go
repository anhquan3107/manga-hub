package fetcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// writeJSON writes a data structure to a JSON file with indentation.
func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write json file: %w", err)
	}

	return nil
}
