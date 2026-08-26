package testastic

import (
	"fmt"
	"strings"
)

// formatYAMLDiffInline generates a git-style inline diff between expected and actual YAML.
func formatYAMLDiffInline(expected, actual yamlDocuments, cfg *config) string {
	stream := newYAMLStreamContext(len(expected), len(actual), cfg)
	expClean := make(yamlDocuments, len(expected))

	for index, document := range expected {
		var actualDocument any
		if index < len(actual) {
			actualDocument = actual[index]
		}

		documentContext := stream.document(index)
		expClean[index] = substituteMatchedMatchers(
			document,
			actualDocument,
			documentContext.path,
			documentContext.config,
		)
	}

	actClean := make(yamlDocuments, len(actual))
	for index, document := range actual {
		actClean[index] = cleanMatchersForDisplay(document)
	}

	expYAML, err := renderYAMLDocuments(expClean)
	if err != nil {
		return fmt.Sprintf("error formatting expected: %v", err)
	}

	actYAML, err := renderYAMLDocuments(actClean)
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
