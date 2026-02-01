package testastic

import (
	"strings"
)

// diffType represents the type of difference found.
type diffType int

const (
	// diffChanged indicates a value was modified.
	diffChanged diffType = iota
	// diffAdded indicates a value was added (exists in actual but not expected).
	diffAdded
	// diffRemoved indicates a value was removed (exists in expected but not actual).
	diffRemoved
	// diffTypeMismatch indicates the types don't match.
	diffTypeMismatch
	// diffMatcherFailed indicates a matcher didn't match the actual value.
	diffMatcherFailed
)

// String returns a human-readable description of the diff type.
func (d diffType) String() string {
	switch d {
	case diffChanged:
		return "changed"
	case diffAdded:
		return "added"
	case diffRemoved:
		return "removed"
	case diffTypeMismatch:
		return "type mismatch"
	case diffMatcherFailed:
		return "matcher failed"
	default:
		return "unknown"
	}
}

// difference represents a single difference between expected and actual JSON.
type difference struct {
	Path     string   // JSON path, e.g., "$.users[0].name"
	Expected any      // Expected value (or matcher description)
	Actual   any      // Actual value
	Type     diffType // Type of difference
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
