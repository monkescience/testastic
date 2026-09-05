package testastic

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
)

type yamlDocuments []any

type yamlStreamContext struct {
	multipleDocuments bool
	base              *config
}

type yamlDocumentContext struct {
	path   string
	config *config
}

func newYAMLStreamContext(expectedCount, actualCount int, base *config) yamlStreamContext {
	return yamlStreamContext{
		multipleDocuments: expectedCount > 1 || actualCount > 1,
		base:              base,
	}
}

func (c yamlStreamContext) document(index int) yamlDocumentContext {
	if !c.multipleDocuments {
		return yamlDocumentContext{path: "$", config: c.base}
	}

	documentConfig := *c.base
	documentPath := yamlDocumentPath(index, true)
	documentConfig.IgnoreArrayOrderPaths = qualifyYAMLDocumentPaths(
		c.base.IgnoreArrayOrderPaths,
		documentPath,
	)
	documentConfig.IgnoredFields = qualifyYAMLDocumentPaths(c.base.IgnoredFields, documentPath)

	return yamlDocumentContext{path: documentPath, config: &documentConfig}
}

func parseYAMLDocuments(data []byte) (yamlDocuments, error) {
	tokens := preserveEmptyYAMLDocumentTokens(lexer.Tokenize(string(data)))

	file, err := parser.Parse(tokens, 0)
	if err != nil {
		return nil, fmt.Errorf("decode YAML: %w", err)
	}

	documents := make(yamlDocuments, 0, len(file.Docs))
	decoder := yaml.NewDecoder(bytes.NewReader(nil))

	for _, documentNode := range file.Docs {
		if documentNode.Body == nil {
			documents = append(documents, nil)

			continue
		}

		var document any

		err = decoder.DecodeFromNode(documentNode.Body, &document)
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

func preserveEmptyYAMLDocumentTokens(tokens token.Tokens) token.Tokens {
	var preserved token.Tokens

	for index, current := range tokens {
		preserved.Add(current)

		if current.Type != token.DocumentHeaderType || !emptyYAMLDocumentFollows(tokens[index+1:]) {
			continue
		}

		// The upstream parser stops grouping at adjacent empty documents.
		position := *current.Position
		position.Line++
		preserved.Add(token.New("null", "null", &position))
	}

	return preserved
}

func emptyYAMLDocumentFollows(tokens token.Tokens) bool {
	for _, current := range tokens {
		if current.Type == token.CommentType {
			continue
		}

		return current.Type == token.DocumentHeaderType || current.Type == token.DocumentEndType
	}

	return true
}

type yamlComparison struct {
	differences []difference
	documents   []treeComparison
	actual      yamlDocuments
}

func compareYAMLDocuments(expected, actual yamlDocuments, cfg *config) yamlComparison {
	stream := newYAMLStreamContext(len(expected), len(actual), cfg)
	result := yamlComparison{documents: make([]treeComparison, len(expected)), actual: actual}

	for index := range max(len(expected), len(actual)) {
		document := stream.document(index)
		switch {
		case index >= len(expected):
			result.differences = append(result.differences, difference{
				Path: document.path, Expected: nil, Actual: actual[index], Type: diffAdded,
			})
		case index >= len(actual):
			result.differences = append(result.differences, difference{
				Path: document.path, Expected: expected[index], Actual: nil, Type: diffRemoved,
			})
			result.documents[index] = treeComparison{
				expected: expected[index], path: document.path, config: document.config,
			}
		default:
			comparison := compareTree(expected[index], actual[index], document.path, document.config)
			result.differences = append(result.differences, comparison.differences...)
			result.documents[index] = comparison
		}
	}

	return result
}

func (c yamlComparison) diagnostic() (yamlDocuments, yamlDocuments) {
	expected := make(yamlDocuments, len(c.documents))
	for index, document := range c.documents {
		expected[index] = document.diagnostic().expected
	}

	actual := make(yamlDocuments, len(c.actual))
	for index, document := range c.actual {
		actual[index] = cleanMatchersForDisplay(document)
	}

	return expected, actual
}

func yamlDocumentPath(index int, multipleDocuments bool) string {
	if !multipleDocuments {
		return "$"
	}

	return fmt.Sprintf("$[%d]", index)
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
