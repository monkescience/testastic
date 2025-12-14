package testastic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// File permission constants for test data files.
const (
	dirPerm  = 0o755
	filePerm = 0o644
)

// updateExpectedFile updates the expected file with the actual value.
// It preserves template matchers from the original file.
func updateExpectedFile(path string, actual []byte, expected *ExpectedJSON) error {
	// Parse actual JSON
	var actualData any

	unmarshalErr := json.Unmarshal(actual, &actualData)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to parse actual JSON for update: %w", unmarshalErr)
	}

	// Get matcher positions from original expected file
	matcherPositions := expected.ExtractMatcherPositions()

	// Generate updated JSON with matchers preserved
	updatedJSON, err := generateUpdatedJSON(actualData, matcherPositions)
	if err != nil {
		return fmt.Errorf("failed to generate updated JSON: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)

	mkdirErr := os.MkdirAll(dir, dirPerm)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}

	// Write to file
	writeErr := os.WriteFile(path, []byte(updatedJSON), filePerm)
	if writeErr != nil {
		return fmt.Errorf("failed to write expected file: %w", writeErr)
	}

	return nil
}

// createExpectedFile creates a new expected file from actual data.
func createExpectedFile(path string, actual []byte) error {
	// Pretty-print the JSON
	var data any

	unmarshalErr := json.Unmarshal(actual, &data)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to parse actual JSON: %w", unmarshalErr)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)

	mkdirErr := os.MkdirAll(dir, dirPerm)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}

	// Write to file
	writeErr := os.WriteFile(path, append(prettyJSON, '\n'), filePerm)
	if writeErr != nil {
		return fmt.Errorf("failed to write expected file: %w", writeErr)
	}

	return nil
}

// generateUpdatedJSON creates JSON output with matchers preserved at their original positions.
func generateUpdatedJSON(data any, matcherPositions map[string]string) (string, error) {
	// First, generate the pretty JSON
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if len(matcherPositions) == 0 {
		return string(prettyJSON) + "\n", nil
	}

	// Replace values at matcher positions with the original matcher expressions
	result := string(prettyJSON)
	for path, matcherExpr := range matcherPositions {
		result = replaceValueAtPath(result, path, matcherExpr)
	}

	return result + "\n", nil
}

// replaceValueAtPath replaces the value at a JSON path with a matcher expression.
// This is a simplified implementation that works for common cases.
func replaceValueAtPath(jsonStr, path, matcherExpr string) string {
	// Convert path to key name
	// e.g., "$.user.id" -> "id"
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return jsonStr
	}

	key := parts[len(parts)-1]

	// Handle array index in key
	if idx := strings.Index(key, "["); idx > 0 {
		key = key[:idx]
	}

	// Create regex to match "key": <value>
	// This is a simplified approach that may not work for all cases
	pattern := fmt.Sprintf(`("%s"\s*:\s*)((?:"[^"]*")|(?:\d+(?:\.\d+)?)|(?:true|false|null))`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(pattern)

	// Replace with matcher expression
	result := re.ReplaceAllStringFunc(jsonStr, func(match string) string {
		// Find the colon position
		colonIdx := strings.Index(match, ":")
		if colonIdx < 0 {
			return match
		}

		prefix := match[:colonIdx+1]
		// Preserve whitespace after colon
		rest := match[colonIdx+1:]

		var whitespace strings.Builder

		for _, c := range rest {
			if c != ' ' && c != '\t' {
				break
			}

			whitespace.WriteRune(c)
		}

		return prefix + whitespace.String() + `"` + matcherExpr + `"`
	})

	return result
}
