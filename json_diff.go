package testastic

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatJSONDiffInline generates a git-style inline diff between expected and actual JSON.
// Shows the full JSON with - prefix for removed lines and + prefix for added lines.
func formatJSONDiffInline(diagnostic treeDiagnostic) string {
	expJSON, err := json.MarshalIndent(diagnostic.expected, "", "  ")
	if err != nil {
		return fmt.Sprintf("error formatting expected: %v", err)
	}

	actJSON, err := json.MarshalIndent(diagnostic.actual, "", "  ")
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
