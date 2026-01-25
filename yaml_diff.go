package testastic

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func FormatYAMLDiff(diffs []Difference) string {
	return formatDiffList(diffs, "YAML", formatYAMLValue, typeOfYAML)
}

// FormatYAMLDiffInline generates a git-style inline diff between expected and actual YAML.
func FormatYAMLDiffInline(expected, actual any) string {
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

// yamlNullDisplay is the string representation for null values in YAML.
const yamlNullDisplay = "null"

// formatYAMLValue formats a value for display in YAML diff output.
func formatYAMLValue(v any) string {
	if v == nil {
		return yamlNullDisplay
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
		data, err := yaml.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}

		s := strings.TrimSpace(string(data))
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

// YAML type name constants.
const (
	yamlTypeNull     = "null"
	yamlTypeString   = "string"
	yamlTypeNumber   = "number"
	yamlTypeInteger  = "integer"
	yamlTypeBoolean  = "boolean"
	yamlTypeMapping  = "mapping"
	yamlTypeSequence = "sequence"
	yamlTypeMatcher  = "matcher"
)

// typeOfYAML returns a human-readable type name for a YAML value.
func typeOfYAML(v any) string {
	if v == nil {
		return yamlTypeNull
	}

	switch v.(type) {
	case string:
		return yamlTypeString
	case float64:
		return yamlTypeNumber
	case int, int64, int32:
		return yamlTypeInteger
	case bool:
		return yamlTypeBoolean
	case map[string]any:
		return yamlTypeMapping
	case []any:
		return yamlTypeSequence
	case Matcher:
		return yamlTypeMatcher
	default:
		return fmt.Sprintf("%T", v)
	}
}
