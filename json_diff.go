package testastic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatJSONDiffInline generates a git-style inline diff between expected and actual JSON.
// Shows the full JSON with - prefix for removed lines and + prefix for added lines.
func formatJSONDiffInline(expected, actual any) string {
	expClean := cleanMatchersForDisplay(expected)
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
