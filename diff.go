package testastic

import (
	"fmt"
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

// maxDisplayLineLen is the maximum length for displaying values before truncation.
const maxDisplayLineLen = 80

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

type (
	valueFormatter func(any) string
	typeFormatter  func(any) string
)

func formatDiffList(diffs []Difference, formatName string, fmtValue valueFormatter, fmtType typeFormatter) string {
	if len(diffs) == 0 {
		return ""
	}

	var sb strings.Builder

	if len(diffs) == 1 {
		sb.WriteString(formatName + " mismatch at 1 path:\n")
	} else {
		sb.WriteString(fmt.Sprintf("%s mismatch at %d paths:\n", formatName, len(diffs)))
	}

	for _, d := range diffs {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", d.Path))

		switch d.Type {
		case DiffAdded:
			sb.WriteString("    expected: (missing)\n")
			sb.WriteString(fmt.Sprintf("    actual:   %s\n", fmtValue(d.Actual)))

		case DiffRemoved:
			sb.WriteString(fmt.Sprintf("    expected: %s\n", fmtValue(d.Expected)))
			sb.WriteString("    actual:   (missing)\n")

		case DiffTypeMismatch:
			sb.WriteString(fmt.Sprintf("    expected: %s (%s)\n", fmtValue(d.Expected), fmtType(d.Expected)))
			sb.WriteString(fmt.Sprintf("    actual:   %s (%s)\n", fmtValue(d.Actual), fmtType(d.Actual)))

		case DiffChanged, DiffMatcherFailed:
			sb.WriteString(fmt.Sprintf("    expected: %s\n", fmtValue(d.Expected)))
			sb.WriteString(fmt.Sprintf("    actual:   %s\n", fmtValue(d.Actual)))
		}
	}

	return sb.String()
}

// FormatFileDiffInline generates a git-style inline diff for file comparison.
func FormatFileDiffInline(expected, actual []string) string {
	diff := computeDiff(expected, actual)

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
	// Build the longest common subsequence (LCS) matrix.
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

	// Backtrack through LCS matrix to build diff operations.
	var result []string

	i, j := m, n

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
