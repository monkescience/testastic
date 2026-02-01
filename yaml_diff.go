package testastic

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// formatYAMLDiffInline generates a git-style inline diff between expected and actual YAML.
func formatYAMLDiffInline(expected, actual any) string {
	expClean := cleanMatchersForDisplay(expected)
	actClean := cleanMatchersForDisplay(actual)

	expYAML, err := yaml.Marshal(expClean)
	if err != nil {
		return fmt.Sprintf("error formatting expected: %v", err)
	}

	actYAML, err := yaml.Marshal(actClean)
	if err != nil {
		return fmt.Sprintf("error formatting actual: %v", err)
	}

	expLines := strings.Split(strings.TrimSuffix(string(expYAML), "\n"), "\n")
	actLines := strings.Split(strings.TrimSuffix(string(actYAML), "\n"), "\n")
	diff := computeDiff(expLines, actLines)

	var sb strings.Builder

	for _, line := range diff {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}
