package testastic

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

type expectedYAML struct {
	Data     any               // Parsed YAML with Matcher objects in place of template expressions
	Matchers map[string]string // Map of placeholder to original template expression
	Raw      string            // Original file content for update operations
}

const yamlMatcherPlaceholderPrefix = "__TESTASTIC_YAML_MATCHER_"

// yamlTemplateExprRegex matches {{...}} expressions in YAML.
var yamlTemplateExprRegex = regexp.MustCompile(`\{\{((?:[^}` + "`" + `]+|` + "`" + `[^` + "`" + `]*` + "`" + `)+)\}\}`)

// parseExpectedYAMLFile reads and parses an expected YAML file, replacing template expressions with matchers.
func parseExpectedYAMLFile(path string) (*expectedYAML, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected YAML file: %w", err)
	}

	return parseExpectedYAMLString(string(content))
}

func parseExpectedYAMLString(content string) (*expectedYAML, error) {
	literalContent := yamlTemplateExprRegex.ReplaceAllString(content, "null")

	var literalData any

	err := yaml.Unmarshal([]byte(literalContent), &literalData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected YAML: %w", err)
	}

	literalValues := stringValues(literalData)
	expected := &expectedYAML{
		Matchers: make(map[string]string),
		Raw:      content,
	}

	matcherIndex := 0
	processedContent := yamlTemplateExprRegex.ReplaceAllStringFunc(content, func(match string) string {
		expr := match
		expr = strings.TrimPrefix(expr, "{{")
		expr = strings.TrimSuffix(expr, "}}")
		expr = trimSpace(expr)

		placeholder, nextIndex := matcherPlaceholder(yamlMatcherPlaceholderPrefix, matcherIndex, literalValues)
		matcherIndex = nextIndex
		expected.Matchers[placeholder] = expr

		return placeholder
	})

	var data any

	err = yaml.Unmarshal([]byte(processedContent), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected YAML: %w", err)
	}

	normalized := normalizeYAMLData(data)

	replaced, err := replaceYAMLPlaceholders(normalized, expected.Matchers)
	if err != nil {
		return nil, err
	}

	expected.Data = replaced

	return expected, nil
}

// replaceYAMLPlaceholders walks the parsed YAML and replaces placeholder strings with Matcher objects.
func replaceYAMLPlaceholders(data any, matchers map[string]string) (any, error) {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			replaced, err := replaceYAMLPlaceholders(val, matchers)
			if err != nil {
				return nil, err
			}

			result[key] = replaced
		}

		return result, nil

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			replaced, err := replaceYAMLPlaceholders(val, matchers)
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

func (e *expectedYAML) extractMatcherPositions() map[string]string {
	positions := make(map[string]string)
	extractYAMLMatcherPaths(e.Data, "$", positions)

	return positions
}

func extractYAMLMatcherPaths(data any, path string, positions map[string]string) {
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
				extractYAMLMatcherPaths(val, childPath, positions)
			}
		}

	case []any:
		for i, val := range v {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if m, ok := val.(Matcher); ok {
				positions[childPath] = m.String()
			} else {
				extractYAMLMatcherPaths(val, childPath, positions)
			}
		}
	}
}
