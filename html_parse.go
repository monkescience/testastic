package testastic

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/html"
)

type htmlNodeType string

const (
	htmlElement htmlNodeType = "element"
	htmlText    htmlNodeType = "text"
	htmlComment htmlNodeType = "comment"
	htmlDoctype htmlNodeType = "doctype"
)

// htmlNode represents a normalized HTML node for comparison.
type htmlNode struct {
	Type       htmlNodeType
	Tag        string
	Attributes map[string]any
	Children   []*htmlNode
	Text       any
	Path       string
}

type expectedHTML struct {
	Root *htmlNode
	Raw  string
}

const htmlMatcherPlaceholderPrefix = "__TESTASTIC_HTML_MATCHER_"

// htmlDocumentTag is the synthetic tag for the document root node.
const htmlDocumentTag = "#document"

type htmlMatcherPreparer struct{}

func (htmlMatcherPreparer) prepare(source string) (preparedMatcherSource, error) {
	values, err := parseHTMLValues(source)
	if err != nil {
		return preparedMatcherSource{}, fmt.Errorf("parse HTML literals: %w", err)
	}

	return preparedMatcherSource{
		source:           source,
		sites:            matcherSourceSites(source, matcherSourceRules{}, rawMatcherPlaceholder),
		placeholder:      htmlMatcherPlaceholderPrefix,
		collides:         func(candidate string) bool { return containsHTMLValue(values, candidate) },
		embedded:         true,
		trimWholeMatcher: true,
	}, nil
}

// parseExpectedHTMLFile reads and parses an expected HTML file, replacing template expressions with matchers.
func parseExpectedHTMLFile(path string) (*expectedHTML, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected HTML file: %w", err)
	}

	expected, err := parseExpectedHTMLString(string(content))
	if err != nil {
		_, isCompileError := errors.AsType[*matcherCompileError](err)
		if !isCompileError {
			return nil, err
		}

		if errors.Is(err, errUnknownMatcher) {
			return nil, fmt.Errorf(
				"failed to parse expected HTML file %s: %w, register custom matchers with RegisterMatcher",
				path,
				err,
			)
		}

		return nil, fmt.Errorf("failed to parse expected HTML file %s: %w", path, err)
	}

	return expected, nil
}

func parseExpectedHTMLString(content string) (*expectedHTML, error) {
	program, err := compileMatcherProgram(content, htmlMatcherPreparer{})
	if err != nil {
		_, isCompileError := errors.AsType[*matcherCompileError](err)
		if !isCompileError {
			return nil, fmt.Errorf("failed to parse expected HTML: %w", err)
		}

		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(program.sourceForParser()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected HTML: %w", err)
	}

	root, err := convertTohtmlNode(doc, &program, "")
	if err != nil {
		return nil, err
	}

	return &expectedHTML{Root: root, Raw: program.original}, nil
}

func parseHTMLValues(content string) ([]string, error) {
	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("parse HTML values: %w", err)
	}

	var values []string

	var collect func(*html.Node)

	collect = func(node *html.Node) {
		values = append(values, node.Data)

		for _, attr := range node.Attr {
			values = append(values, attr.Val)
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}

	collect(doc)

	return values, nil
}

func containsHTMLValue(values []string, candidate string) bool {
	for _, value := range values {
		if strings.Contains(value, candidate) {
			return true
		}
	}

	return false
}

func parseActualHTMLBytes(data []byte) (*htmlNode, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse actual HTML: %w", err)
	}

	return convertTohtmlNode(doc, nil, "")
}

// convertTohtmlNode converts an html.Node to an htmlNode tree.
//
//nolint:gocognit,funlen,nilnil // DOM conversion ignores node types with a nil node.
func convertTohtmlNode(n *html.Node, program *matcherProgram, parentPath string) (*htmlNode, error) {
	if n == nil {
		return nil, nil
	}

	switch n.Type { //nolint:exhaustive // Only handling relevant node types.
	case html.ElementNode:
		path := buildElementPath(parentPath, n.Data)
		node := &htmlNode{
			Type:       htmlElement,
			Tag:        n.Data,
			Path:       path,
			Attributes: make(map[string]any),
		}

		for _, attr := range n.Attr {
			resolved, err := resolveHTMLValue(attr.Val, program)
			if err != nil {
				return nil, err
			}

			node.Attributes[attr.Key] = resolved
		}

		childCounts := make(map[string]int)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child, err := convertChildTohtmlNode(c, program, path, childCounts)
			if err != nil {
				return nil, err
			}

			if child != nil {
				node.Children = append(node.Children, child)
			}
		}

		return node, nil

	case html.TextNode:
		text := n.Data

		resolved, err := resolveHTMLValue(text, program)
		if err != nil {
			return nil, err
		}

		return &htmlNode{
			Type: htmlText,
			Text: resolved,
			Path: parentPath + " (text)",
		}, nil

	case html.CommentNode:
		resolved, err := resolveHTMLValue(n.Data, program)
		if err != nil {
			return nil, err
		}

		return &htmlNode{
			Type: htmlComment,
			Text: resolved,
			Path: parentPath + " (comment)",
		}, nil

	case html.DoctypeNode:
		return &htmlNode{
			Type: htmlDoctype,
			Tag:  n.Data,
			Path: "<!DOCTYPE>",
		}, nil

	case html.DocumentNode:
		// For document nodes, find the root element
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				return convertTohtmlNode(c, program, parentPath)
			}

			if c.Type != html.DoctypeNode {
				continue
			}

			// Create a wrapper that includes both doctype and root element
			root := &htmlNode{
				Type: htmlElement,
				Tag:  htmlDocumentTag,
				Path: "",
			}

			for child := n.FirstChild; child != nil; child = child.NextSibling {
				childNode, err := convertTohtmlNode(child, program, "")
				if err != nil {
					return nil, err
				}

				if childNode != nil {
					root.Children = append(root.Children, childNode)
				}
			}

			return root, nil
		}
		// No root element found, wrap children
		root := &htmlNode{
			Type: htmlElement,
			Tag:  htmlDocumentTag,
			Path: "",
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child, err := convertTohtmlNode(c, program, "")
			if err != nil {
				return nil, err
			}

			if child != nil {
				root.Children = append(root.Children, child)
			}
		}

		return root, nil

	default:
		return nil, nil
	}
}

// convertChildTohtmlNode handles child node conversion with proper path indexing.
//
//nolint:nilnil // DOM conversion ignores child node types with a nil node.
func convertChildTohtmlNode(
	n *html.Node, program *matcherProgram, parentPath string, childCounts map[string]int,
) (*htmlNode, error) {
	if n == nil {
		return nil, nil
	}

	if n.Type == html.ElementNode {
		tag := n.Data
		index := childCounts[tag]
		childCounts[tag]++

		path := buildElementPathWithIndex(parentPath, tag, index)
		node := &htmlNode{
			Type:       htmlElement,
			Tag:        tag,
			Path:       path,
			Attributes: make(map[string]any),
		}

		for _, attr := range n.Attr {
			resolved, err := resolveHTMLValue(attr.Val, program)
			if err != nil {
				return nil, err
			}

			node.Attributes[attr.Key] = resolved
		}

		nestedCounts := make(map[string]int)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child, err := convertChildTohtmlNode(c, program, path, nestedCounts)
			if err != nil {
				return nil, err
			}

			if child != nil {
				node.Children = append(node.Children, child)
			}
		}

		return node, nil
	}

	return convertTohtmlNode(n, program, parentPath)
}

func buildElementPath(parentPath, tag string) string {
	if parentPath == "" {
		return tag
	}

	return parentPath + " > " + tag
}

// buildElementPathWithIndex builds an HTML path with index for repeated elements.
func buildElementPathWithIndex(parentPath, tag string, index int) string {
	if parentPath == "" {
		if index == 0 {
			return tag
		}

		return fmt.Sprintf("%s[%d]", tag, index)
	}

	if index == 0 {
		return parentPath + " > " + tag
	}

	return fmt.Sprintf("%s > %s[%d]", parentPath, tag, index)
}

func resolveHTMLValue(value string, program *matcherProgram) (any, error) {
	if program == nil {
		return value, nil
	}

	return program.resolve(value)
}
