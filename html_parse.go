package testastic

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// HTMLNodeType represents the type of an HTML node.
type HTMLNodeType int

const (
	// HTMLElement represents an HTML element node.
	HTMLElement HTMLNodeType = iota
	// HTMLText represents a text node.
	HTMLText
	// HTMLComment represents a comment node.
	HTMLComment
	// HTMLDoctype represents a doctype node.
	HTMLDoctype
)

// HTMLNode represents a normalized HTML node for comparison.
type HTMLNode struct {
	Type       HTMLNodeType
	Tag        string
	Attributes map[string]any
	Children   []*HTMLNode
	Text       any
	Path       string
}

// ExpectedHTML represents a parsed expected HTML file with matchers.
type ExpectedHTML struct {
	Root     *HTMLNode
	Matchers map[string]string
	Raw      string
}

// htmlMatcherPlaceholderPrefix is the prefix used for HTML matcher placeholders.
const htmlMatcherPlaceholderPrefix = "__TESTASTIC_HTML_MATCHER_"

// htmlTemplateExprRegex matches {{...}} expressions in HTML.
var htmlTemplateExprRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ParseExpectedHTMLFile reads and parses an expected HTML file, replacing template expressions with matchers.
func ParseExpectedHTMLFile(path string) (*ExpectedHTML, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected HTML file: %w", err)
	}

	return ParseExpectedHTMLString(string(content))
}

// ParseExpectedHTMLString parses an expected HTML string with template expressions.
func ParseExpectedHTMLString(content string) (*ExpectedHTML, error) {
	expected := &ExpectedHTML{
		Matchers: make(map[string]string),
		Raw:      content,
	}

	// Find all template expressions and replace with placeholders
	matcherIndex := 0
	processedContent := htmlTemplateExprRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the expression (remove {{ and }})
		expr := match
		expr = strings.TrimPrefix(expr, "{{")
		expr = strings.TrimSuffix(expr, "}}")
		expr = trimSpace(expr)

		placeholder := fmt.Sprintf("%s%d__", htmlMatcherPlaceholderPrefix, matcherIndex)
		expected.Matchers[placeholder] = expr
		matcherIndex++

		return placeholder
	})

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(processedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected HTML: %w", err)
	}

	// Convert to HTMLNode tree with matchers
	expected.Root = convertToHTMLNode(doc, expected.Matchers, "")

	return expected, nil
}

// parseActualHTMLBytes parses actual HTML bytes into an HTMLNode tree.
func parseActualHTMLBytes(data []byte) (*HTMLNode, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse actual HTML: %w", err)
	}

	return convertToHTMLNode(doc, nil, ""), nil
}

// convertToHTMLNode converts an html.Node to an HTMLNode tree.
//
//nolint:gocognit,funlen // HTML DOM conversion requires handling multiple node types.
func convertToHTMLNode(n *html.Node, matchers map[string]string, parentPath string) *HTMLNode {
	if n == nil {
		return nil
	}

	switch n.Type { //nolint:exhaustive // Only handling relevant node types.
	case html.ElementNode:
		path := buildElementPath(parentPath, n.Data)
		node := &HTMLNode{
			Type:       HTMLElement,
			Tag:        n.Data,
			Path:       path,
			Attributes: make(map[string]any),
		}

		// Process attributes
		for _, attr := range n.Attr {
			node.Attributes[attr.Key] = resolveHTMLMatcherInValue(attr.Val, matchers)
		}

		// Process children
		childCounts := make(map[string]int)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := convertChildToHTMLNode(c, matchers, path, childCounts)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}

		return node

	case html.TextNode:
		text := n.Data
		resolved := resolveHTMLMatcherInValue(text, matchers)

		// Check if the text is only whitespace
		if s, ok := resolved.(string); ok && strings.TrimSpace(s) == "" {
			return nil
		}

		return &HTMLNode{
			Type: HTMLText,
			Text: resolved,
			Path: parentPath + " (text)",
		}

	case html.CommentNode:
		return &HTMLNode{
			Type: HTMLComment,
			Text: n.Data,
			Path: parentPath + " (comment)",
		}

	case html.DoctypeNode:
		return &HTMLNode{
			Type: HTMLDoctype,
			Tag:  n.Data,
			Path: "<!DOCTYPE>",
		}

	case html.DocumentNode:
		// For document nodes, find the root element
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				return convertToHTMLNode(c, matchers, parentPath)
			}
			// Also handle doctype
			if c.Type == html.DoctypeNode {
				// Create a wrapper that includes both doctype and root element
				root := &HTMLNode{
					Type: HTMLElement,
					Tag:  "#document",
					Path: "",
				}

				for child := n.FirstChild; child != nil; child = child.NextSibling {
					childNode := convertToHTMLNode(child, matchers, "")
					if childNode != nil {
						root.Children = append(root.Children, childNode)
					}
				}

				return root
			}
		}
		// No root element found, wrap children
		root := &HTMLNode{
			Type: HTMLElement,
			Tag:  "#document",
			Path: "",
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := convertToHTMLNode(c, matchers, "")
			if child != nil {
				root.Children = append(root.Children, child)
			}
		}

		return root

	default:
		return nil
	}
}

// convertChildToHTMLNode handles child node conversion with proper path indexing.
func convertChildToHTMLNode(
	n *html.Node, matchers map[string]string, parentPath string, childCounts map[string]int,
) *HTMLNode {
	if n == nil {
		return nil
	}

	if n.Type == html.ElementNode {
		// Track element index for path building
		tag := n.Data
		index := childCounts[tag]
		childCounts[tag]++

		path := buildElementPathWithIndex(parentPath, tag, index)
		node := &HTMLNode{
			Type:       HTMLElement,
			Tag:        tag,
			Path:       path,
			Attributes: make(map[string]any),
		}

		// Process attributes
		for _, attr := range n.Attr {
			node.Attributes[attr.Key] = resolveHTMLMatcherInValue(attr.Val, matchers)
		}

		// Process children recursively
		nestedCounts := make(map[string]int)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := convertChildToHTMLNode(c, matchers, path, nestedCounts)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}

		return node
	}

	// For non-element nodes, delegate to standard conversion
	return convertToHTMLNode(n, matchers, parentPath)
}

// buildElementPath builds an HTML path for an element.
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

// resolveHTMLMatcherInValue checks if a string contains a matcher placeholder and returns the Matcher.
func resolveHTMLMatcherInValue(value string, matchers map[string]string) any {
	if matchers == nil {
		return value
	}

	// Check if the entire value is a single matcher placeholder
	if strings.HasPrefix(value, htmlMatcherPlaceholderPrefix) && strings.HasSuffix(value, "__") {
		if expr, ok := matchers[value]; ok {
			matcher, err := ParseMatcher(expr)
			if err == nil {
				return matcher
			}
		}
	}

	// Check if value contains any matcher placeholders (partial match)
	for placeholder, expr := range matchers {
		if strings.Contains(value, placeholder) {
			// For partial matches, we need to handle it as a pattern
			// For now, if the entire trimmed value is the placeholder, return matcher
			if strings.TrimSpace(value) == placeholder {
				matcher, err := ParseMatcher(expr)
				if err == nil {
					return matcher
				}
			}
			// Otherwise, replace placeholder back with original expression for display
			value = strings.ReplaceAll(value, placeholder, "{{"+expr+"}}")
		}
	}

	return value
}

// ExtractMatcherPositions returns a map of HTML paths to their original template expressions.
func (e *ExpectedHTML) ExtractMatcherPositions() map[string]string {
	positions := make(map[string]string)
	extractHTMLMatcherPaths(e.Root, positions)

	return positions
}

// extractHTMLMatcherPaths recursively finds all Matcher positions in the HTML tree.
func extractHTMLMatcherPaths(node *HTMLNode, positions map[string]string) {
	if node == nil {
		return
	}

	// Check text content
	if m, ok := node.Text.(Matcher); ok {
		positions[node.Path] = m.String()
	}

	// Check attributes
	for attr, val := range node.Attributes {
		if m, ok := val.(Matcher); ok {
			positions[node.Path+"@"+attr] = m.String()
		}
	}

	// Recurse into children
	for _, child := range node.Children {
		extractHTMLMatcherPaths(child, positions)
	}
}
