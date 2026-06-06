package testastic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// File permission constants for test data files.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// updateExpectedFile updates the expected file with the actual value.
// It preserves template matchers from the original file.
func updateExpectedFile(path string, actual []byte, expected *expectedJSON) error {
	var actualData any
	if err := decodeJSON(actual, &actualData); err != nil {
		return fmt.Errorf("failed to parse actual JSON for update: %w", err)
	}

	matcherPositions := expected.extractMatcherPositions()

	updatedJSON, err := generateUpdatedJSON(actualData, matcherPositions)
	if err != nil {
		return fmt.Errorf("failed to generate updated JSON: %w", err)
	}

	return writeFileAtomic(path, []byte(updatedJSON))
}

func createExpectedFile(path string, actual []byte) error {
	var data any
	if err := decodeJSON(actual, &data); err != nil {
		return fmt.Errorf("failed to parse actual JSON: %w", err)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	return writeFileAtomic(path, append(prettyJSON, '\n'))
}

// generateUpdatedJSON creates JSON output with matchers preserved at their original positions.
func generateUpdatedJSON(data any, matcherPositions map[string]string) (string, error) {
	if len(matcherPositions) == 0 {
		prettyJSON, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}

		return string(prettyJSON) + "\n", nil
	}

	// Restore matcher expressions as string values at their original positions,
	// then marshal. JSON matchers are quoted strings, so no post-marshal fixup
	// is needed (unlike the YAML path).
	merged := restoreJSONMatchers(data, matcherPositions, "$")

	prettyJSON, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(prettyJSON) + "\n", nil
}

// restoreJSONMatchers walks the data tree and inserts matcher expression strings
// at positions where the expected file had matchers.
func restoreJSONMatchers(data any, matchers map[string]string, path string) any {
	if expr, ok := matchers[path]; ok {
		return expr
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			childPath := path + "." + key
			result[key] = restoreJSONMatchers(val, matchers, childPath)
		}

		return result

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			result[i] = restoreJSONMatchers(val, matchers, childPath)
		}

		return result

	default:
		return v
	}
}

// writeFileAtomic writes data to path by writing a temp file in the same
// directory and renaming it into place, so a reader never observes a partial
// write and a failed write cannot truncate an existing file.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".testastic-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)

		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)

		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpName, filePerm); err != nil {
		_ = os.Remove(tmpName)

		return fmt.Errorf("failed to set file mode: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)

		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
