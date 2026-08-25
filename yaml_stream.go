package testastic

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
)

type yamlDocuments []any

func parseYAMLDocuments(data []byte) (yamlDocuments, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var documents yamlDocuments

	for {
		var document any

		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("decode YAML: %w", err)
		}

		documents = append(documents, normalizeYAMLData(document))
	}

	if len(documents) == 0 {
		return yamlDocuments{nil}, nil
	}

	return documents, nil
}

func compareYAMLDocuments(expected, actual yamlDocuments, cfg *config) []difference {
	multipleDocuments := len(expected) > 1 || len(actual) > 1

	var diffs []difference

	for index := range max(len(expected), len(actual)) {
		path := yamlDocumentPath(index, multipleDocuments)
		documentConfig := configForYAMLDocument(cfg, index, multipleDocuments)

		switch {
		case index >= len(expected):
			diffs = append(diffs, difference{
				Path:     path,
				Expected: nil,
				Actual:   actual[index],
				Type:     diffAdded,
			})
		case index >= len(actual):
			diffs = append(diffs, difference{
				Path:     path,
				Expected: expected[index],
				Actual:   nil,
				Type:     diffRemoved,
			})
		default:
			diffs = append(diffs, compare(expected[index], actual[index], path, documentConfig)...)
		}
	}

	return diffs
}

func yamlDocumentPath(index int, multipleDocuments bool) string {
	if !multipleDocuments {
		return "$"
	}

	return fmt.Sprintf("$[%d]", index)
}

func yamlMatcherDocumentPath(index int) string {
	return fmt.Sprintf("$[%d]", index)
}

func configForYAMLDocument(cfg *config, index int, multipleDocuments bool) *config {
	if !multipleDocuments {
		return cfg
	}

	documentConfig := *cfg
	documentPath := yamlDocumentPath(index, true)
	documentConfig.IgnoreArrayOrderPaths = qualifyYAMLDocumentPaths(cfg.IgnoreArrayOrderPaths, documentPath)
	documentConfig.IgnoredFields = qualifyYAMLDocumentPaths(cfg.IgnoredFields, documentPath)

	return &documentConfig
}

func qualifyYAMLDocumentPaths(paths []string, documentPath string) []string {
	qualified := make([]string, len(paths))

	for index, path := range paths {
		switch {
		case path == "$", strings.HasPrefix(path, "$."), strings.HasPrefix(path, "$["):
			qualified[index] = documentPath + path[1:]
		default:
			qualified[index] = path
		}
	}

	return qualified
}

func renderYAMLDocuments(documents yamlDocuments) (string, error) {
	var output strings.Builder

	for index, document := range documents {
		if index > 0 {
			output.WriteString("---\n")
		}

		formatted, err := yaml.Marshal(document)
		if err != nil {
			return "", fmt.Errorf("marshal YAML document %d: %w", index, err)
		}

		output.WriteString(strings.TrimSuffix(string(formatted), "\n"))
		output.WriteString("\n")
	}

	return output.String(), nil
}
