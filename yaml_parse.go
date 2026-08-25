package testastic

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type expectedYAML struct {
	Documents yamlDocuments
	Matchers  map[string]string
	Raw       string
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

	literalDocuments, err := parseYAMLDocuments([]byte(literalContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected YAML: %w", err)
	}

	literalValues := make(map[string]struct{})
	for _, document := range literalDocuments {
		collectStringValues(document, literalValues)
	}

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

	documents, err := parseYAMLDocuments([]byte(processedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected YAML: %w", err)
	}

	for index, document := range documents {
		replaced, replaceErr := replaceYAMLPlaceholders(document, expected.Matchers)
		if replaceErr != nil {
			return nil, replaceErr
		}

		documents[index] = replaced
	}

	expected.Documents = documents

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
