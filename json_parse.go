package testastic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

// errJSONTrailingContent is returned when JSON contains more than one top-level value.
var errJSONTrailingContent = errors.New("trailing content after top-level JSON value")

type expectedJSON struct {
	Data any    // Parsed JSON with Matcher objects in place of template expressions
	Raw  string // Original file content for update operations
}

const jsonMatcherPlaceholderPrefix = "__TESTASTIC_MATCHER_"

type jsonMatcherPreparer struct{}

func (jsonMatcherPreparer) prepare(source string) (preparedMatcherSource, error) {
	sites := matcherSourceSites(source, matcherSourceRules{
		includeWholeQuotes:       true,
		unclosedBacktickConsumes: true,
	}, strconv.Quote)
	if len(sites) == 0 {
		return preparedMatcherSource{source: source}, nil
	}

	literalSource := substituteMatcherSourceSites(source, sites, "null")

	var literalData any

	err := decodeJSON([]byte(literalSource), &literalData)
	if err != nil {
		return preparedMatcherSource{}, err
	}

	literalValues := stringValues(literalData)

	return preparedMatcherSource{
		source:      source,
		sites:       sites,
		placeholder: jsonMatcherPlaceholderPrefix,
		collides: func(candidate string) bool {
			_, exists := literalValues[candidate]

			return exists
		},
	}, nil
}

// parseExpectedJSONFile reads and parses an expected file, replacing template expressions with matchers.
func parseExpectedJSONFile(path string) (*expectedJSON, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected file: %w", err)
	}

	return parseExpectedJSONString(string(content))
}

func parseExpectedJSONString(content string) (*expectedJSON, error) {
	program, err := compileMatcherProgram(content, jsonMatcherPreparer{})
	if err != nil {
		compileErr, ok := errors.AsType[*matcherCompileError](err)
		if ok {
			return nil, fmt.Errorf("failed to parse matcher %q: %w", compileErr.expression, compileErr.cause)
		}

		return nil, fmt.Errorf("failed to parse expected file as JSON: %w", err)
	}

	expected := &expectedJSON{Raw: program.original}

	var data any

	err = decodeJSON([]byte(program.sourceForParser()), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected file as JSON: %w", err)
	}

	replaced, err := program.resolve(data)
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
