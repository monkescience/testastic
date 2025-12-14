package testastic

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ErrUnknownPlaceholder is returned when a placeholder is not found in the matcher map.
var ErrUnknownPlaceholder = errors.New("unknown placeholder")

// ExpectedJSON represents a parsed expected file with matchers.
type ExpectedJSON struct {
	Data     any               // Parsed JSON with Matcher objects in place of template expressions
	Matchers map[string]string // Map of placeholder to original template expression
	Raw      string            // Original file content for update operations
}

// matcherPlaceholderPrefix is the prefix used for matcher placeholders.
const matcherPlaceholderPrefix = "__TESTASTIC_MATCHER_"

// templateExprRegex matches {{...}} expressions.
var templateExprRegex = regexp.MustCompile(`"?\{\{([^}]+)\}\}"?`)

// ParseExpectedFile reads and parses an expected file, replacing template expressions with matchers.
func ParseExpectedFile(path string) (*ExpectedJSON, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected file: %w", err)
	}

	return ParseExpectedString(string(content))
}

// ParseExpectedString parses an expected JSON string with template expressions.
func ParseExpectedString(content string) (*ExpectedJSON, error) {
	expected := &ExpectedJSON{
		Matchers: make(map[string]string),
		Raw:      content,
	}

	// Find all template expressions and replace with placeholders
	matcherIndex := 0
	processedContent := templateExprRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the expression (remove {{ and }})
		expr := match
		// Remove surrounding quotes if present
		if strings.HasPrefix(expr, `"{{`) {
			expr = strings.TrimPrefix(expr, `"`)
		}

		if strings.HasSuffix(expr, `}}"`) {
			expr = strings.TrimSuffix(expr, `"`)
		}
		// Remove {{ and }}
		expr = strings.TrimPrefix(expr, "{{")
		expr = strings.TrimSuffix(expr, "}}")
		expr = trimSpace(expr)

		placeholder := fmt.Sprintf(`"%s%d__"`, matcherPlaceholderPrefix, matcherIndex)
		expected.Matchers[fmt.Sprintf("%s%d__", matcherPlaceholderPrefix, matcherIndex)] = expr
		matcherIndex++

		return placeholder
	})

	// Parse as standard JSON
	var data any

	err := json.Unmarshal([]byte(processedContent), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected file as JSON: %w", err)
	}

	// Walk the parsed structure and replace placeholders with Matcher objects
	replaced, err := replacePlaceholders(data, expected.Matchers)
	if err != nil {
		return nil, err
	}

	expected.Data = replaced

	return expected, nil
}

// replacePlaceholders walks the parsed JSON and replaces placeholder strings with Matcher objects.
func replacePlaceholders(data any, matchers map[string]string) (any, error) {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			replaced, err := replacePlaceholders(val, matchers)
			if err != nil {
				return nil, err
			}

			result[key] = replaced
		}

		return result, nil

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			replaced, err := replacePlaceholders(val, matchers)
			if err != nil {
				return nil, err
			}

			result[i] = replaced
		}

		return result, nil

	case string:
		if strings.HasPrefix(v, matcherPlaceholderPrefix) {
			expr, ok := matchers[v]
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrUnknownPlaceholder, v)
			}

			matcher, err := ParseMatcher(expr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse matcher %q: %w", expr, err)
			}

			return matcher, nil
		}

		return v, nil

	default:
		return v, nil
	}
}

// ExtractMatcherPositions returns a map of JSON paths to their original template expressions.
// This is used when updating expected files to preserve matchers.
func (e *ExpectedJSON) ExtractMatcherPositions() map[string]string {
	positions := make(map[string]string)
	extractMatcherPaths(e.Data, "$", positions)

	return positions
}

// extractMatcherPaths recursively finds all Matcher positions in the data structure.
func extractMatcherPaths(data any, path string, positions map[string]string) {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			childPath := path + "." + key
			if m, ok := val.(Matcher); ok {
				positions[childPath] = m.String()
			} else {
				extractMatcherPaths(val, childPath, positions)
			}
		}

	case []any:
		for i, val := range v {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if m, ok := val.(Matcher); ok {
				positions[childPath] = m.String()
			} else {
				extractMatcherPaths(val, childPath, positions)
			}
		}
	}
}
