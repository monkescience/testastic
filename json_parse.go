package testastic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// errJSONTrailingContent is returned when JSON contains more than one top-level value.
var errJSONTrailingContent = errors.New("trailing content after top-level JSON value")

type expectedJSON struct {
	Data     any               // Parsed JSON with Matcher objects in place of template expressions
	Matchers map[string]string // Map of placeholder to original template expression
	Raw      string            // Original file content for update operations
}

const jsonMatcherPlaceholderPrefix = "__TESTASTIC_MATCHER_"

// jsonTemplateExprRegex matches {{...}} expressions.
var jsonTemplateExprRegex = regexp.MustCompile(
	`"?\{\{((?:[^}` + "`" + `]+|` + "`" + `[^` + "`" + `]*` + "`" + `)+)\}\}"?`,
)

// parseExpectedJSONFile reads and parses an expected file, replacing template expressions with matchers.
func parseExpectedJSONFile(path string) (*expectedJSON, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected file: %w", err)
	}

	return parseExpectedJSONString(string(content))
}

func parseExpectedJSONString(content string) (*expectedJSON, error) {
	literalContent := jsonTemplateExprRegex.ReplaceAllString(content, "null")

	var literalData any

	err := decodeJSON([]byte(literalContent), &literalData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected file as JSON: %w", err)
	}

	literalValues := stringValues(literalData)
	expected := &expectedJSON{
		Matchers: make(map[string]string),
		Raw:      content,
	}

	matcherIndex := 0
	processedContent := jsonTemplateExprRegex.ReplaceAllStringFunc(content, func(match string) string {
		expr := match

		// Strip surrounding quotes if the expression was quoted in JSON.
		if strings.HasPrefix(expr, `"{{`) {
			expr = strings.TrimPrefix(expr, `"`)
		}

		if strings.HasSuffix(expr, `}}"`) {
			expr = strings.TrimSuffix(expr, `"`)
		}

		expr = strings.TrimPrefix(expr, "{{")
		expr = strings.TrimSuffix(expr, "}}")
		expr = trimSpace(expr)

		placeholder, nextIndex := matcherPlaceholder(jsonMatcherPlaceholderPrefix, matcherIndex, literalValues)
		matcherIndex = nextIndex
		expected.Matchers[placeholder] = expr

		return fmt.Sprintf(`"%s"`, placeholder)
	})

	var data any

	err = decodeJSON([]byte(processedContent), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected file as JSON: %w", err)
	}

	replaced, err := replaceJSONPlaceholders(data, expected.Matchers)
	if err != nil {
		return nil, err
	}

	expected.Data = replaced

	return expected, nil
}

func decodeJSON(data []byte, target *any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	err := decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	var extra any

	extraErr := decoder.Decode(&extra)
	if errors.Is(extraErr, io.EOF) {
		return nil
	}

	if extraErr != nil {
		return fmt.Errorf("decode JSON: %w", extraErr)
	}

	return errJSONTrailingContent
}

// replaceJSONPlaceholders walks the parsed JSON and replaces placeholder strings with Matcher objects.
func replaceJSONPlaceholders(data any, matchers map[string]string) (any, error) {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			replaced, err := replaceJSONPlaceholders(val, matchers)
			if err != nil {
				return nil, err
			}

			result[key] = replaced
		}

		return result, nil

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			replaced, err := replaceJSONPlaceholders(val, matchers)
			if err != nil {
				return nil, err
			}

			result[i] = replaced
		}

		return result, nil

	case string:
		expr, ok := matchers[v]
		if !ok {
			return v, nil
		}

		matcher, err := parseMatcher(expr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse matcher %q: %w", expr, err)
		}

		return matcher, nil

	default:
		return v, nil
	}
}

// extractMatcherPositions returns a map of JSON paths to their original template expressions.
// This is used when updating expected files to preserve matchers.
func (e *expectedJSON) extractMatcherPositions() map[string]string {
	positions := make(map[string]string)
	extractJSONMatcherPaths(e.Data, "$", positions)

	return positions
}

func extractJSONMatcherPaths(data any, path string, positions map[string]string) {
	if m, ok := data.(Matcher); ok {
		positions[path] = m.String()

		return
	}

	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			childPath := path + "." + key
			if m, ok := val.(Matcher); ok {
				positions[childPath] = m.String()
			} else {
				extractJSONMatcherPaths(val, childPath, positions)
			}
		}

	case []any:
		for i, val := range v {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if m, ok := val.(Matcher); ok {
				positions[childPath] = m.String()
			} else {
				extractJSONMatcherPaths(val, childPath, positions)
			}
		}
	}
}
