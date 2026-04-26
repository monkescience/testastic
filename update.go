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
func updateExpectedFile(path string, actual []byte, expected *expectedJSON) error {
	var actualData any

	unmarshalErr := decodeJSON(actual, &actualData)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to parse actual JSON for update: %w", unmarshalErr)
	}

	matcherPositions := expected.extractMatcherPositions()

	updatedJSON, err := generateUpdatedJSON(actualData, matcherPositions)
	if err != nil {
		return fmt.Errorf("failed to generate updated JSON: %w", err)
	}

	dir := filepath.Dir(path)

	mkdirErr := os.MkdirAll(dir, dirPerm)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}

	writeErr := os.WriteFile(path, []byte(updatedJSON), filePerm)
	if writeErr != nil {
		return fmt.Errorf("failed to write expected file: %w", writeErr)
	}

	return nil
}

func createExpectedFile(path string, actual []byte) error {
	var data any

	unmarshalErr := decodeJSON(actual, &data)
	if unmarshalErr != nil {
		return fmt.Errorf("failed to parse actual JSON: %w", unmarshalErr)
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	dir := filepath.Dir(path)

	mkdirErr := os.MkdirAll(dir, dirPerm)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}

	writeErr := os.WriteFile(path, append(prettyJSON, '\n'), filePerm)
	if writeErr != nil {
		return fmt.Errorf("failed to write expected file: %w", writeErr)
	}

	return nil
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

	// Restore matcher expressions at their original positions in the data tree,
	// then marshal and replace the quoted placeholders with the raw expressions.
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

// replaceValueAtPath replaces the value at a JSON path with a matcher expression.
// It walks the path segments to find the correct nesting level and only replaces
// the first occurrence of the key at that level, avoiding false matches on
// duplicate key names at different paths.
func replaceValueAtPath(jsonStr, path, matcherExpr string) string {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return jsonStr
	}

	// Build a regex-based approach that accounts for the path hierarchy.
	// We search for the context of parent keys to narrow down the replacement.
	key := parts[len(parts)-1]

	// Handle array index in key
	if idx := strings.Index(key, "["); idx > 0 {
		key = key[:idx]
	}

	// For simple single-segment paths (e.g., "$.id"), use the old approach
	// since there is no ambiguity at the top level.
	pattern := fmt.Sprintf(`("%s"\s*:\s*)((?:"[^"]*")|(?:\d+(?:\.\d+)?)|(?:true|false|null))`, regexp.QuoteMeta(key))
	re := regexp.MustCompile(pattern)

	// A path like "$.id" has 2 parts: ["$", "id"]. Anything with at most 2 parts
	// is a top-level key with no parent scope needed to disambiguate.
	topLevelMaxParts := 2

	if len(parts) <= topLevelMaxParts {
		return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
			return replaceMatchValue(match, matcherExpr)
		})
	}

	// For nested paths, find the correct occurrence by locating the parent key first,
	// then replacing only the target key within that parent's scope.
	parentKey := parts[len(parts)-2]

	if idx := strings.Index(parentKey, "["); idx > 0 {
		parentKey = parentKey[:idx]
	}

	parentPattern := fmt.Sprintf(`"%s"\s*:`, regexp.QuoteMeta(parentKey))
	parentRe := regexp.MustCompile(parentPattern)

	parentLoc := parentRe.FindStringIndex(jsonStr)
	if parentLoc == nil {
		// Parent not found, fall back to first-match replacement
		replaced := false

		return re.ReplaceAllStringFunc(jsonStr, func(match string) string {
			if replaced {
				return match
			}

			replaced = true

			return replaceMatchValue(match, matcherExpr)
		})
	}

	// Replace only the first occurrence of the key after the parent
	afterParent := jsonStr[parentLoc[0]:]
	loc := re.FindStringIndex(afterParent)

	if loc == nil {
		return jsonStr
	}

	startInOriginal := parentLoc[0] + loc[0]
	endInOriginal := parentLoc[0] + loc[1]
	match := jsonStr[startInOriginal:endInOriginal]

	return jsonStr[:startInOriginal] + replaceMatchValue(match, matcherExpr) + jsonStr[endInOriginal:]
}

// replaceMatchValue replaces the value portion of a "key": value match with a matcher expression.
func replaceMatchValue(match, matcherExpr string) string {
	colonIdx := strings.Index(match, ":")
	if colonIdx < 0 {
		return match
	}

	prefix := match[:colonIdx+1]
	rest := match[colonIdx+1:]

	var whitespace strings.Builder

	for _, c := range rest {
		if c != ' ' && c != '\t' {
			break
		}

		whitespace.WriteRune(c)
	}

	return prefix + whitespace.String() + `"` + matcherExpr + `"`
}
