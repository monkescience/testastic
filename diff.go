package testastic

import (
	"slices"
	"strings"
)

type diffType string

const (
	// diffChanged indicates a value was modified.
	diffChanged diffType = "changed"
	// diffAdded indicates a value was added (exists in actual but not expected).
	diffAdded diffType = "added"
	// diffRemoved indicates a value was removed (exists in expected but not actual).
	diffRemoved diffType = "removed"
	// diffTypeMismatch indicates the types don't match.
	diffTypeMismatch diffType = "type mismatch"
	// diffMatcherFailed indicates a matcher didn't match the actual value.
	diffMatcherFailed diffType = "matcher failed"
)

func (d diffType) String() string {
	return string(d)
}

// difference represents a single difference between expected and actual JSON.
type difference struct {
	Path     string // JSON path, e.g., "$.users[0].name"
	Expected any    // Expected value (or matcher description)
	Actual   any
	Type     diffType
}

func formatFileDiffInline(expected, actual []string) string {
	diff := computeDiff(expected, actual)

	var sb strings.Builder
	for _, line := range diff {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

type diffOp string

const (
	maxDiffMatrixCells        = 1 << 20
	maxDiffMatrixLines        = 1 << 12
	diffEqual          diffOp = "equal"
	diffDelete         diffOp = "delete"
	diffInsert         diffOp = "insert"
)

// computeDiff generates a unified diff between two sets of lines.
// Uses a bounded LCS-based algorithm with a linear fallback.
//
//nolint:funlen // LCS algorithm requires sequential steps.
func computeDiff(expected, actual []string) []string {
	if len(expected) > maxDiffMatrixLines || len(actual) > maxDiffMatrixLines {
		return computeFallbackDiff(expected, actual)
	}

	// Build the longest common subsequence (LCS) matrix.
	m, n := len(expected), len(actual)

	rows, columns := m+1, n+1
	if rows > maxDiffMatrixCells/columns {
		return computeFallbackDiff(expected, actual)
	}

	dp := make([][]int, rows)
	for i := range dp {
		dp[i] = make([]int, columns)
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

	for _, op := range slices.Backward(ops) {
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

func computeFallbackDiff(expected, actual []string) []string {
	commonPrefix := 0
	for commonPrefix < min(len(expected), len(actual)) && expected[commonPrefix] == actual[commonPrefix] {
		commonPrefix++
	}

	commonSuffix := 0
	maxCommonSuffix := min(len(expected), len(actual)) - commonPrefix

	for commonSuffix < maxCommonSuffix &&
		expected[len(expected)-1-commonSuffix] == actual[len(actual)-1-commonSuffix] {
		commonSuffix++
	}

	result := make([]string, 0, max(len(expected), len(actual)))
	for _, line := range expected[:commonPrefix] {
		result = append(result, "  "+line)
	}

	for _, line := range expected[commonPrefix : len(expected)-commonSuffix] {
		result = append(result, red("- "+line))
	}

	for _, line := range actual[commonPrefix : len(actual)-commonSuffix] {
		result = append(result, green("+ "+line))
	}

	for _, line := range expected[len(expected)-commonSuffix:] {
		result = append(result, "  "+line)
	}

	return result
}
