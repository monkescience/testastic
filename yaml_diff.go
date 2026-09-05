package testastic

import (
	"fmt"
	"strings"
)

// formatYAMLDiffInline generates a git-style inline diff between expected and actual YAML.
func formatYAMLDiffInline(expected, actual yamlDocuments) string {
	expYAML, err := renderYAMLDocuments(expected)
	if err != nil {
		return fmt.Sprintf("error formatting expected: %v", err)
	}

	actYAML, err := renderYAMLDocuments(actual)
	if err != nil {
		return fmt.Sprintf("error formatting actual: %v", err)
	}

	expLines := strings.Split(strings.TrimSuffix(expYAML, "\n"), "\n")
	actLines := strings.Split(strings.TrimSuffix(actYAML, "\n"), "\n")
	diff := computeDiff(expLines, actLines)

	var sb strings.Builder

	for _, line := range diff {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}
