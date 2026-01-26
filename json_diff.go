package testastic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FormatDiff formats a slice of differences into a human-readable string.
// This is the simple format showing paths and values.
func FormatDiff(diffs []Difference) string {
	return formatDiffList(diffs, "JSON", formatValue, typeOf)
}

// FormatDiffInline generates a git-style inline diff between expected and actual JSON.
// Shows the full JSON with - prefix for removed lines and + prefix for added lines.
func FormatDiffInline(expected, actual any) string {
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

// formatValue formats a value for display in diff output.
func formatValue(v any) string {
	if v == nil {
		return "null"
	}

	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)

	case float64:
		// Display integers without decimal point.
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}

		return fmt.Sprintf("%g", val)

	case bool:
		return strconv.FormatBool(val)

	case map[string]any, []any:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}

		s := string(data)
		if len(s) > maxDisplayLineLen {
			return s[:maxDisplayLineLen-3] + "..."
		}

		return s

	case Matcher:
		return val.String()

	default:
		return fmt.Sprintf("%v", val)
	}
}

// typeOf returns a human-readable type name for a value.
func typeOf(v any) string {
	if v == nil {
		return "null"
	}

	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case Matcher:
		return "matcher"
	default:
		return fmt.Sprintf("%T", v)
	}
}
