package testastic

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
)

// AssertYAML compares actual YAML against an expected YAML file.
// T can be: []byte, string, io.Reader, or any struct (auto-marshaled).
//
// Example:
//
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", configBytes)
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", myConfig)
//	testastic.AssertYAML(t, "testdata/config.expected.yaml", resp.Body)
func AssertYAML[T any](tb testing.TB, expectedFile string, actual T, opts ...Option) {
	tb.Helper()

	actualBytes, err := toYAMLBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	cfg := newConfig(opts)

	if unsupported := cfg.validateOptions(assertYAML); len(unsupported) > 0 {
		tb.Fatalf("testastic: unsupported options for AssertYAML: %s", strings.Join(unsupported, ", "))

		return
	}

	if handleMissingExpectedFile(tb, expectedFile, actualBytes, cfg.Update, createExpectedYAMLFile) {
		return
	}

	expected, err := parseExpectedYAMLFile(expectedFile)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	actualDocuments, err := parseActualYAML(actualBytes)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	diffs := compareYAMLDocuments(expected.Documents, actualDocuments, cfg)

	if handleYAMLDiffs(tb, expectedFile, actualBytes, expected, actualDocuments, diffs, cfg) {
		return
	}
}

// handleYAMLDiffs handles update mode and error reporting for YAML.
// Returns true if the assertion should stop.
func handleYAMLDiffs(
	tb testing.TB, path string, actualBytes []byte, expected *expectedYAML,
	actualDocuments yamlDocuments, diffs []difference, cfg *config,
) bool {
	tb.Helper()

	if len(diffs) == 0 {
		return false
	}

	if cfg.Update {
		updateErr := updateExpectedYAMLFile(path, actualBytes, expected)
		if updateErr != nil {
			tb.Fatalf("testastic: failed to update expected YAML file: %v", updateErr)
		}

		tb.Logf("testastic: updated expected YAML file %s", path)

		return true
	}

	msg := formatAssertionMessage("AssertYAML", path, cfg.Message)
	tb.Errorf("testastic: assertion failed\n\n  %s\n%s",
		msg, formatYAMLDiffInline(expected.Documents, actualDocuments, cfg))

	return false
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

func parseActualYAML(data []byte) (yamlDocuments, error) {
	documents, err := parseYAMLDocuments(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actual YAML: %w", err)
	}

	return documents, nil
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

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = normalizeYAMLData(val)
		}

		return result

	case []byte:
		// A !!binary scalar decodes to []byte, but a Go []byte field marshals
		// to a YAML sequence of integers. Normalize both to the same []any so
		// equivalent bytes compare equal regardless of source.
		result := make([]any, len(v))
		for i, b := range v {
			result[i] = uint64(b)
		}

		return result

	case float32:
		return float64(v)

	default:
		return v
	}
}

func createExpectedYAMLFile(path string, actual []byte) error {
	documents, err := parseYAMLDocuments(actual)
	if err != nil {
		return writeYAMLFile(path, actual)
	}

	formatted, err := renderYAMLDocuments(documents)
	if err != nil {
		return writeYAMLFile(path, actual)
	}

	return writeYAMLFile(path, []byte(formatted))
}

func updateExpectedYAMLFile(path string, actual []byte, expected *expectedYAML) error {
	actualDocuments, err := parseYAMLDocuments(actual)
	if err != nil {
		return writeYAMLFile(path, actual)
	}

	matcherPositions := expected.extractMatcherPositions()
	mergedDocuments := make(yamlDocuments, len(actualDocuments))

	for index, document := range actualDocuments {
		path := yamlMatcherDocumentPath(index)
		mergedDocuments[index] = restoreYAMLMatchers(document, matcherPositions, path)
	}

	formatted, err := renderYAMLDocuments(mergedDocuments)
	if err != nil {
		return writeYAMLFile(path, actual)
	}

	content := restoreYAMLTemplateExpressions(formatted, matcherPositions)

	return writeYAMLFile(path, []byte(content))
}

// restoreYAMLMatchers walks the actual data and restores matcher expressions
// at positions where the expected file had matchers.
func restoreYAMLMatchers(data any, matchers map[string]string, path string) any {
	if expr, ok := matchers[path]; ok {
		return expr
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

// restoreYAMLTemplateExpressions strips YAML-added quotes from template expressions.
// The YAML marshaler quotes strings containing {{ }} (since {{ starts a YAML flow mapping),
// producing e.g. "{{anyString}}" or '{{anyString}}'. This function restores the unquoted
// form so the file reads naturally: name: {{anyString}}.
func restoreYAMLTemplateExpressions(content string, _ map[string]string) string {
	re := regexp.MustCompile(`["'](\{\{(?:[^}` + "`" + `]+|` + "`" + `[^` + "`" + `]*` + "`" + `)+\}\})["']`)

	return re.ReplaceAllString(content, "$1")
}

func writeYAMLFile(path string, data []byte) error {
	return writeFileAtomic(path, data)
}
