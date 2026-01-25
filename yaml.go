package testastic

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// AssertYAML compares actual YAML against an expected YAML file.
// T can be: []byte, string, io.Reader, or any struct (auto-marshaled).
//
// Example:
//
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", configBytes)
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", myConfig)
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", resp.Body)
//
//nolint:funlen // Main assertion function needs sequential validation steps.
func AssertYAML[T any](tb testing.TB, expectedFile string, actual T, opts ...interface{}) {
	tb.Helper()

	// Convert actual to []byte
	actualBytes, err := toYAMLBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	// Build config
	cfg := &YAMLConfig{
		BaseConfig: BaseConfig{
			Update: shouldUpdate(),
		},
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case AssertionOption:
			o.applyToYAMLConfig(cfg)
		case YAMLOption:
			o(cfg)
		default:
			tb.Fatalf("testastic: invalid option type: %T", opt)
		}
	}

	// Check if expected file exists
	_, statErr := os.Stat(expectedFile)
	if os.IsNotExist(statErr) {
		if cfg.Update {
			createErr := createExpectedYAMLFile(expectedFile, actualBytes)
			if createErr != nil {
				tb.Fatalf("testastic: failed to create expected YAML file: %v", createErr)
			}

			tb.Logf("testastic: created expected YAML file %s", expectedFile)

			return
		}

		tb.Fatalf(
			"testastic: expected YAML file does not exist: %s (run with -update to create)",
			expectedFile,
		)

		return
	}

	// Parse expected file
	expected, err := ParseExpectedYAMLFile(expectedFile)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	// Parse actual YAML
	actualData, err := parseActualYAML(actualBytes)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	// Compare using the same logic as JSON (YAML parses to same Go types)
	diffs := compare(expected.Data, actualData, "$", cfg)

	// If update mode and there are differences, update the file
	if cfg.Update && len(diffs) > 0 {
		updateErr := updateExpectedYAMLFile(expectedFile, actualBytes, expected)
		if updateErr != nil {
			tb.Fatalf("testastic: failed to update expected YAML file: %v", updateErr)
		}

		tb.Logf("testastic: updated expected YAML file %s", expectedFile)

		return
	}

	// Report differences
	if len(diffs) > 0 {
		sortDiffs(diffs)
		msg := formatAssertionMessage("AssertYAML", expectedFile, cfg.Message)
		tb.Errorf(
			"testastic: assertion failed\n\n  %s\n%s",
			msg, FormatYAMLDiffInline(expected.Data, actualData),
		)
	}
}

// toYAMLBytes converts various input types to []byte of YAML.
func toYAMLBytes[T any](v T) ([]byte, error) {
	if data, handled, err := bytesFromCommonInput(v); handled {
		return data, err
	}

	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	return data, nil
}

// parseActualYAML converts the actual YAML bytes to a comparable structure.
func parseActualYAML(data []byte) (any, error) {
	var result any

	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actual YAML: %w", err)
	}

	// Normalize the data structure to match JSON conventions
	return normalizeYAMLData(result), nil
}

// normalizeYAMLData converts YAML-specific types to JSON-compatible types.
// YAML uses map[string]any for objects (same as JSON) but may have different
// numeric types.
func normalizeYAMLData(data any) any {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			result[key] = normalizeYAMLData(val)
		}

		return result

	case map[any]any:
		// YAML can produce map[any]any, convert to map[string]any
		result := make(map[string]any, len(v))
		for key, val := range v {
			keyStr := fmt.Sprintf("%v", key)
			result[keyStr] = normalizeYAMLData(val)
		}

		return result

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = normalizeYAMLData(val)
		}

		return result

	case int:
		return float64(v)

	case int64:
		return float64(v)

	case int32:
		return float64(v)

	case float32:
		return float64(v)

	default:
		return v
	}
}

// createExpectedYAMLFile creates a new expected YAML file.
func createExpectedYAMLFile(path string, actual []byte) error {
	// Parse and re-render for consistent formatting
	var data any

	err := yaml.Unmarshal(actual, &data)
	if err != nil {
		// If parsing fails, just write the raw content
		return writeYAMLFile(path, actual)
	}

	formatted, err := yaml.Marshal(data)
	if err != nil {
		return writeYAMLFile(path, actual)
	}

	return writeYAMLFile(path, formatted)
}

// updateExpectedYAMLFile updates an existing expected YAML file.
func updateExpectedYAMLFile(path string, actual []byte, expected *ExpectedYAML) error {
	// Parse actual to preserve any matchers from expected
	var actualData any

	err := yaml.Unmarshal(actual, &actualData)
	if err != nil {
		return writeYAMLFile(path, actual)
	}

	// Get matcher positions from expected
	matcherPositions := expected.ExtractMatcherPositions()

	// Restore matchers in actual data
	mergedData := restoreYAMLMatchers(actualData, matcherPositions, "$")

	// Marshal back to YAML
	formatted, err := yaml.Marshal(mergedData)
	if err != nil {
		return writeYAMLFile(path, actual)
	}

	// Replace matcher placeholders back to template syntax
	content := restoreYAMLTemplateExpressions(string(formatted), matcherPositions)

	return writeYAMLFile(path, []byte(content))
}

// restoreYAMLMatchers walks the actual data and restores matcher expressions
// at positions where the expected file had matchers.
func restoreYAMLMatchers(data any, matchers map[string]string, path string) any {
	if expr, ok := matchers[path]; ok {
		return "{{" + expr + "}}"
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			childPath := path + "." + key
			result[key] = restoreYAMLMatchers(val, matchers, childPath)
		}

		return result

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			result[i] = restoreYAMLMatchers(val, matchers, childPath)
		}

		return result

	default:
		return v
	}
}

// restoreYAMLTemplateExpressions converts quoted template expressions back to unquoted.
func restoreYAMLTemplateExpressions(content string, _ map[string]string) string {
	// YAML marshaling may quote the {{...}} expressions, we need to handle this
	// The expressions should already be in the right format from restoreYAMLMatchers
	return content
}

// writeYAMLFile writes data to a file with proper error wrapping.
func writeYAMLFile(path string, data []byte) error {
	err := os.WriteFile(path, data, filePerm)
	if err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	return nil
}
