package testastic

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DiffType represents the type of difference found.
type DiffType int

const (
	// DiffChanged indicates a value was modified.
	DiffChanged DiffType = iota
	// DiffAdded indicates a value was added (exists in actual but not expected).
	DiffAdded
	// DiffRemoved indicates a value was removed (exists in expected but not actual).
	DiffRemoved
	// DiffTypeMismatch indicates the types don't match.
	DiffTypeMismatch
	// DiffMatcherFailed indicates a matcher didn't match the actual value.
	DiffMatcherFailed
)

// String returns a human-readable description of the diff type.
func (d DiffType) String() string {
	switch d {
	case DiffChanged:
		return "changed"
	case DiffAdded:
		return "added"
	case DiffRemoved:
		return "removed"
	case DiffTypeMismatch:
		return "type mismatch"
	case DiffMatcherFailed:
		return "matcher failed"
	default:
		return "unknown"
	}
}

// Difference represents a single difference between expected and actual JSON.
type Difference struct {
	Path     string   // JSON path, e.g., "$.users[0].name"
	Expected any      // Expected value (or matcher description)
	Actual   any      // Actual value
	Type     DiffType // Type of difference
}

// FormatDiff formats a slice of differences into a human-readable string.
// This is the simple format showing paths and values.
func FormatDiff(diffs []Difference) string {
	if len(diffs) == 0 {
		return ""
	}

	var sb strings.Builder

	// Header
	if len(diffs) == 1 {
		sb.WriteString("JSON mismatch at 1 path:\n")
	} else {
		sb.WriteString(fmt.Sprintf("JSON mismatch at %d paths:\n", len(diffs)))
	}

	// Each difference
	for _, d := range diffs {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", d.Path))

		switch d.Type {
		case DiffAdded:
			sb.WriteString("    expected: (missing)\n")
			sb.WriteString(fmt.Sprintf("    actual:   %s\n", formatValue(d.Actual)))

		case DiffRemoved:
			sb.WriteString(fmt.Sprintf("    expected: %s\n", formatValue(d.Expected)))
			sb.WriteString("    actual:   (missing)\n")

		case DiffTypeMismatch:
			sb.WriteString(fmt.Sprintf("    expected: %s (%s)\n", formatValue(d.Expected), typeOf(d.Expected)))
			sb.WriteString(fmt.Sprintf("    actual:   %s (%s)\n", formatValue(d.Actual), typeOf(d.Actual)))

		case DiffChanged, DiffMatcherFailed:
			sb.WriteString(fmt.Sprintf("    expected: %s\n", formatValue(d.Expected)))
			sb.WriteString(fmt.Sprintf("    actual:   %s\n", formatValue(d.Actual)))
		}
	}

	return sb.String()
}

// FormatDiffInline generates a git-style inline diff between expected and actual JSON.
// Shows the full JSON with - prefix for removed lines and + prefix for added lines.
func FormatDiffInline(expected, actual any) string {
	// Convert matchers to their string representation for display
	expClean := cleanMatchersForDisplay(expected)
	actClean := cleanMatchersForDisplay(actual)

	// Marshal both to pretty JSON
	expJSON, err := json.MarshalIndent(expClean, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting expected: %v", err)
	}

	actJSON, err := json.MarshalIndent(actClean, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting actual: %v", err)
	}

	// Split into lines
	expLines := strings.Split(string(expJSON), "\n")
	actLines := strings.Split(string(actJSON), "\n")

	// Generate unified diff
	diff := computeDiff(expLines, actLines)

	// Format output
	var sb strings.Builder

	for _, line := range diff {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// diffOp represents a diff operation type.
type diffOp int

const (
	diffEqual diffOp = iota
	diffDelete
	diffInsert
)

// computeDiff generates a unified diff between two sets of lines.
// Uses a simple LCS-based algorithm for readability.
//
//nolint:funlen // LCS algorithm requires sequential steps.
func computeDiff(expected, actual []string) []string {
	// Compute the longest common subsequence matrix
	m, n := len(expected), len(actual)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if expected[i-1] == actual[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to build the diff
	var result []string

	i, j := m, n

	// Collect operations in reverse order
	var ops []struct {
		op   diffOp
		line string
	}

	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && expected[i-1] == actual[j-1]:
			ops = append(ops, struct {
				op   diffOp
				line string
			}{diffEqual, expected[i-1]})
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			ops = append(ops, struct {
				op   diffOp
				line string
			}{diffInsert, actual[j-1]})
			j--
		case i > 0:
			ops = append(ops, struct {
				op   diffOp
				line string
			}{diffDelete, expected[i-1]})
			i--
		}
	}

	// Reverse the operations
	for k := len(ops) - 1; k >= 0; k-- {
		op := ops[k]
		switch op.op {
		case diffEqual:
			result = append(result, "  "+op.line)
		case diffDelete:
			result = append(result, red("- "+op.line))
		case diffInsert:
			result = append(result, green("+ "+op.line))
		}
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

// formatValue formats a value for display in diff output.
func formatValue(v any) string {
	if v == nil {
		return "null"
	}

	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)

	case float64:
		// Check if it's an integer
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}

		return fmt.Sprintf("%g", val)

	case bool:
		return strconv.FormatBool(val)

	case map[string]any, []any:
		// Compact JSON for objects and arrays
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}

		s := string(data)
		// Truncate if too long
		if len(s) > 80 {
			return s[:77] + "..."
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
