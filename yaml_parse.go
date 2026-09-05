package testastic

import (
	"errors"
	"fmt"
	"os"
)

type expectedYAML struct {
	Documents yamlDocuments
	Raw       string
}

const yamlMatcherPlaceholderPrefix = "__TESTASTIC_YAML_MATCHER_"

type yamlMatcherPreparer struct{}

func (yamlMatcherPreparer) prepare(source string) (preparedMatcherSource, error) {
	sites := matcherSourceSites(
		source,
		matcherSourceRules{unclosedBacktickConsumes: true},
		rawMatcherPlaceholder,
	)
	if len(sites) == 0 {
		return preparedMatcherSource{source: source}, nil
	}

	literalSource := substituteMatcherSourceSites(source, sites, "null")

	documents, err := parseYAMLDocuments([]byte(literalSource))
	if err != nil {
		return preparedMatcherSource{}, err
	}

	literalValues := make(map[string]struct{})
	for _, document := range documents {
		collectStringValues(document, literalValues)
	}

	return preparedMatcherSource{
		source:      source,
		sites:       sites,
		placeholder: yamlMatcherPlaceholderPrefix,
		collides: func(candidate string) bool {
			_, exists := literalValues[candidate]

			return exists
		},
	}, nil
}

// parseExpectedYAMLFile reads and parses an expected YAML file, replacing template expressions with matchers.
func parseExpectedYAMLFile(path string) (*expectedYAML, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected YAML file: %w", err)
	}

	return parseExpectedYAMLString(string(content))
}

func parseExpectedYAMLString(content string) (*expectedYAML, error) {
	program, err := compileMatcherProgram(content, yamlMatcherPreparer{})
	if err != nil {
		compileErr, ok := errors.AsType[*matcherCompileError](err)
		if ok {
			return nil, fmt.Errorf("failed to parse matcher %q: %w", compileErr.expression, compileErr.cause)
		}

		return nil, fmt.Errorf("failed to parse expected YAML: %w", err)
	}

	expected := &expectedYAML{Raw: program.original}

	documents, err := parseYAMLDocuments([]byte(program.sourceForParser()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected YAML: %w", err)
	}

	for index, document := range documents {
		replaced, resolveErr := program.resolve(document)
		if resolveErr != nil {
			return nil, resolveErr
		}

		documents[index] = replaced
	}

	expected.Documents = documents

	return expected, nil
}
