package testastic

import (
	"fmt"
)

type comparisonTrace struct {
	matcherChecked bool
	matched        bool
	children       map[string]comparisonTrace
	alignment      *unorderedMatchResult
}

type treeComparison struct {
	differences []difference
	expected    any
	actual      any
	path        string
	config      *config
	trace       comparisonTrace
}

type treeDiagnostic struct {
	expected any
	actual   any
}

func compare(expected, actual any, path string, cfg *config) []difference {
	return compareWithTrace(expected, actual, path, cfg, nil)
}

func compareTree(expected, actual any, path string, cfg *config) treeComparison {
	c := treeComparison{expected: expected, actual: actual, path: path, config: cfg}
	c.differences = compareWithTrace(expected, actual, path, cfg, &c.trace)

	return c
}

func compareChild(expected, actual any, path string, cfg *config, trace *comparisonTrace, key string) []difference {
	if trace == nil {
		return compare(expected, actual, path, cfg)
	}

	var child comparisonTrace

	diffs := compareWithTrace(expected, actual, path, cfg, &child)
	if child.matcherChecked || child.alignment != nil || len(child.children) > 0 {
		if trace.children == nil {
			trace.children = make(map[string]comparisonTrace)
		}

		trace.children[key] = child
	}

	return diffs
}

func (c treeComparison) diagnostic() treeDiagnostic {
	return treeDiagnostic{
		expected: diagnosticExpected(c.expected, c.actual, c.path, c.config, c.trace),
		actual:   cleanMatchersForDisplay(c.actual),
	}
}

func diagnosticExpected(expected, actual any, path string, cfg *config, trace comparisonTrace) any {
	if m, ok := expected.(Matcher); ok {
		matched := trace.matched
		if !trace.matcherChecked {
			matched = m.Match(actual)
		}

		if matched {
			return actual
		}

		return m.String()
	}

	switch exp := expected.(type) {
	case map[string]any:
		actMap, _ := actual.(map[string]any)
		result := make(map[string]any, len(exp))

		for key, val := range exp {
			childPath := path + "." + key
			result[key] = diagnosticExpected(val, actMap[key], childPath, cfg, trace.children[key])
		}

		return result

	case []any:
		actArr, _ := actual.([]any)
		if cfg.ShouldIgnoreArrayOrder(path) {
			return diagnosticUnordered(exp, actArr, path, cfg, trace.alignment)
		}

		result := make([]any, len(exp))

		for i, val := range exp {
			var actVal any
			if i < len(actArr) {
				actVal = actArr[i]
			}

			childPath := fmt.Sprintf("%s[%d]", path, i)
			result[i] = diagnosticExpected(val, actVal, childPath, cfg, trace.children[childPath])
		}

		return result

	default:
		return expected
	}
}

func diagnosticUnordered(expected, actual []any, path string, cfg *config, alignment *unorderedMatchResult) []any {
	if alignment == nil {
		matches := findUnorderedMatches(expected, actual, func(expectedIndex int, actualValue any) bool {
			childPath := fmt.Sprintf("%s[%d]", path, expectedIndex)

			return len(compare(expected[expectedIndex], actualValue, childPath, cfg)) == 0
		})

		alignment = &matches
	}

	matches := alignment
	result := make([]any, 0, len(expected))
	unmatchedIndex := 0

	for actualIndex, expectedIndex := range matches.expectedByActual {
		if expectedIndex >= 0 {
			result = append(result, cleanMatchersForDisplay(actual[actualIndex]))

			continue
		}

		if unmatchedIndex >= len(matches.unmatchedExpected) {
			continue
		}

		expectedIndex = matches.unmatchedExpected[unmatchedIndex]
		unmatchedIndex++
		childPath := fmt.Sprintf("%s[%d]", path, expectedIndex)
		result = append(result,
			compareTree(expected[expectedIndex], actual[actualIndex], childPath, cfg).diagnostic().expected)
	}

	for ; unmatchedIndex < len(matches.unmatchedExpected); unmatchedIndex++ {
		expectedIndex := matches.unmatchedExpected[unmatchedIndex]
		childPath := fmt.Sprintf("%s[%d]", path, expectedIndex)
		result = append(result, diagnosticExpected(expected[expectedIndex], nil, childPath, cfg, comparisonTrace{}))
	}

	return result
}

func cleanMatchersForDisplay(data any) any {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			result[key] = cleanMatchersForDisplay(val)
		}

		return result

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = cleanMatchersForDisplay(val)
		}

		return result

	case Matcher:
		return v.String()

	default:
		return v
	}
}
