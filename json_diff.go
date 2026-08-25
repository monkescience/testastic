package testastic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatJSONDiffInline generates a git-style inline diff between expected and actual JSON.
// Shows the full JSON with - prefix for removed lines and + prefix for added lines.
func formatJSONDiffInline(expected, actual any, cfg *config) string {
	expClean := substituteMatchedMatchers(expected, actual, "$", cfg)
	actClean := cleanMatchersForDisplay(actual)

	expJSON, err := json.MarshalIndent(expClean, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting expected: %v", err)
	}

	actJSON, err := json.MarshalIndent(actClean, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting actual: %v", err)
	}

	expLines := strings.Split(string(expJSON), "\n")
	actLines := strings.Split(string(actJSON), "\n")
	diff := computeDiff(expLines, actLines)

	var sb strings.Builder

	for _, line := range diff {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// substituteMatchedMatchers returns a display copy of expected where every
// matcher the corresponding actual value satisfies is replaced by that actual
// value, so a matched matcher renders identically and is not shown as a
// spurious difference. Unmatched matchers keep their template string.
func substituteMatchedMatchers(expected, actual any, path string, cfg *config) any {
	if m, ok := expected.(Matcher); ok {
		if m.Match(actual) {
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
			result[key] = substituteMatchedMatchers(val, actMap[key], childPath, cfg)
		}

		return result

	case []any:
		actArr, _ := actual.([]any)
		if cfg.ShouldIgnoreArrayOrder(path) {
			return substituteUnorderedMatchers(exp, actArr, path, cfg)
		}

		result := make([]any, len(exp))

		for i, val := range exp {
			var actVal any
			if i < len(actArr) {
				actVal = actArr[i]
			}

			childPath := fmt.Sprintf("%s[%d]", path, i)
			result[i] = substituteMatchedMatchers(val, actVal, childPath, cfg)
		}

		return result

	default:
		return expected
	}
}

func substituteUnorderedMatchers(expected, actual []any, path string, cfg *config) []any {
	matches := findUnorderedMatches(expected, actual, func(expectedIndex int, actualValue any) bool {
		childPath := fmt.Sprintf("%s[%d]", path, expectedIndex)

		return len(compare(expected[expectedIndex], actualValue, childPath, cfg)) == 0
	})

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
			substituteMatchedMatchers(expected[expectedIndex], actual[actualIndex], childPath, cfg))
	}

	for ; unmatchedIndex < len(matches.unmatchedExpected); unmatchedIndex++ {
		expectedIndex := matches.unmatchedExpected[unmatchedIndex]
		childPath := fmt.Sprintf("%s[%d]", path, expectedIndex)
		result = append(result, substituteMatchedMatchers(expected[expectedIndex], nil, childPath, cfg))
	}

	return result
}

// cleanMatchersForDisplay converts Matcher objects to their string representation
// so they can be displayed in the diff output.
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
