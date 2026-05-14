package testastic

import (
	"fmt"
	"os"
	"regexp"
	"sort"
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
	Root     *htmlNode
	Matchers map[string]string
	Raw      string
}

type templateSegment struct {
	Literal string  // Literal text (empty if Matcher is set).
	Matcher Matcher // Matcher (nil if Literal is set).
}

// templateString represents a string with embedded matchers.
type templateString struct {
	Segments []templateSegment
	Original string         // For display: "border-left: 6px solid {{anyString}}".
	regex    *regexp.Regexp // Pre-compiled regex for matching.
}

func (t *templateString) Match(actual string) bool {
	if t.regex == nil {
		return false
	}

	return t.regex.MatchString(actual)
}

func (t *templateString) String() string {
	return t.Original
}

func (t *templateString) buildRegex() error {
	var pattern strings.Builder

	pattern.WriteString("^")

	for _, seg := range t.Segments {
		if seg.Matcher != nil {
			pattern.WriteString(matcherToRegex(seg.Matcher))
		} else {
			pattern.WriteString(regexp.QuoteMeta(seg.Literal))
		}
	}

	pattern.WriteString("$")

	re, err := regexp.Compile(pattern.String())
	if err != nil {
		return fmt.Errorf("failed to compile template regex: %w", err)
	}

	t.regex = re

	return nil
}

func matcherToRegex(m Matcher) string {
	switch v := m.(type) {
	case anyStringMatcher:
		return ".*"
	case anyIntMatcher:
		return "-?\\d+"
	case anyFloatMatcher:
		return "-?\\d+\\.?\\d*"
	case anyBoolMatcher:
		return "(true|false)"
	case anyValueMatcher:
		return ".*"
	case ignoreMatcher:
		return ".*"
	case *regexMatcher:
		return v.pattern
	case *oneOfMatcher:
		return oneOfToRegex(v.values)
	default:
		return ".*"
	}
}

func oneOfToRegex(values []any) string {
	if len(values) == 0 {
		return "(?!)" // Match nothing.
	}

	parts := make([]string, 0, len(values))

	for _, v := range values {
		parts = append(parts, regexp.QuoteMeta(fmt.Sprintf("%v", v)))
	}

	return "(" + strings.Join(parts, "|") + ")"
}

const htmlMatcherPlaceholderPrefix = "__TESTASTIC_HTML_MATCHER_"

// htmlDocumentTag is the synthetic tag for the document root node.
const htmlDocumentTag = "#document"

// htmlTemplateExprRegex matches {{...}} expressions in HTML.
// Handles backtick-quoted content that may contain } characters.
// Effective pattern: \{\{((?:[^}`]+|`[^`]*`)+)\}\}.
var htmlTemplateExprRegex = regexp.MustCompile(`\{\{((?:[^}` + "`" + `]+|` + "`" + `[^` + "`" + `]*` + "`" + `)+)\}\}`)

// parseExpectedHTMLFile reads and parses an expected HTML file, replacing template expressions with matchers.
func parseExpectedHTMLFile(path string) (*expectedHTML, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is controlled by test code.
	if err != nil {
		return nil, fmt.Errorf("failed to read expected HTML file: %w", err)
	}

	return parseExpectedHTMLString(string(content))
}

func parseExpectedHTMLString(content string) (*expectedHTML, error) {
	expected := &expectedHTML{
		Matchers: make(map[string]string),
		Raw:      content,
	}

	matcherIndex := 0
	processedContent := htmlTemplateExprRegex.ReplaceAllStringFunc(content, func(match string) string {
		expr := match
		expr = strings.TrimPrefix(expr, "{{")
		expr = strings.TrimSuffix(expr, "}}")
		expr = trimSpace(expr)

		placeholder := fmt.Sprintf("%s%d__", htmlMatcherPlaceholderPrefix, matcherIndex)
		expected.Matchers[placeholder] = expr
		matcherIndex++

		return placeholder
	})

	doc, err := html.Parse(strings.NewReader(processedContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse expected HTML: %w", err)
	}

	expected.Root = convertTohtmlNode(doc, expected.Matchers, "")

	return expected, nil
}

func parseActualHTMLBytes(data []byte) (*htmlNode, error) {
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse actual HTML: %w", err)
	}

	return convertTohtmlNode(doc, nil, ""), nil
}

// convertTohtmlNode converts an html.Node to an htmlNode tree.
//
//nolint:gocognit,funlen // HTML DOM conversion requires handling multiple node types.
func convertTohtmlNode(n *html.Node, matchers map[string]string, parentPath string) *htmlNode {
	if n == nil {
		return nil
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
			node.Attributes[attr.Key] = resolveHTMLMatcherInValue(attr.Val, matchers)
		}

		childCounts := make(map[string]int)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := convertChildTohtmlNode(c, matchers, path, childCounts)
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

		return &htmlNode{
			Type: htmlText,
			Text: resolved,
			Path: parentPath + " (text)",
		}

	case html.CommentNode:
		return &htmlNode{
			Type: htmlComment,
			Text: n.Data,
			Path: parentPath + " (comment)",
		}

	case html.DoctypeNode:
		return &htmlNode{
			Type: htmlDoctype,
			Tag:  n.Data,
			Path: "<!DOCTYPE>",
		}

	case html.DocumentNode:
		// For document nodes, find the root element
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				return convertTohtmlNode(c, matchers, parentPath)
			}
			// Also handle doctype
			if c.Type == html.DoctypeNode {
				// Create a wrapper that includes both doctype and root element
				root := &htmlNode{
					Type: htmlElement,
					Tag:  htmlDocumentTag,
					Path: "",
				}

				for child := n.FirstChild; child != nil; child = child.NextSibling {
					childNode := convertTohtmlNode(child, matchers, "")
					if childNode != nil {
						root.Children = append(root.Children, childNode)
					}
				}

				return root
			}
		}
		// No root element found, wrap children
		root := &htmlNode{
			Type: htmlElement,
			Tag:  htmlDocumentTag,
			Path: "",
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := convertTohtmlNode(c, matchers, "")
			if child != nil {
				root.Children = append(root.Children, child)
			}
		}

		return root

	default:
		return nil
	}
}

// convertChildTohtmlNode handles child node conversion with proper path indexing.
func convertChildTohtmlNode(
	n *html.Node, matchers map[string]string, parentPath string, childCounts map[string]int,
) *htmlNode {
	if n == nil {
		return nil
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
			node.Attributes[attr.Key] = resolveHTMLMatcherInValue(attr.Val, matchers)
		}

		nestedCounts := make(map[string]int)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			child := convertChildTohtmlNode(c, matchers, path, nestedCounts)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}

		return node
	}

	return convertTohtmlNode(n, matchers, parentPath)
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

// resolveHTMLMatcherInValue checks if a string contains a matcher placeholder and returns the Matcher.
func resolveHTMLMatcherInValue(value string, matchers map[string]string) any {
	if matchers == nil {
		return value
	}

	if m := tryParseSingleMatcher(value, matchers); m != nil {
		return m
	}

	hasPlaceholder := false

	for placeholder := range matchers {
		if !strings.Contains(value, placeholder) {
			continue
		}

		hasPlaceholder = true

		// Handle whitespace around placeholder: "  {{anyString}}  " should match as single matcher.
		if m := tryParseSingleMatcher(strings.TrimSpace(value), matchers); m != nil {
			return m
		}
	}

	if hasPlaceholder {
		return parsetemplateString(value, matchers)
	}

	return value
}

func tryParseSingleMatcher(value string, matchers map[string]string) Matcher {
	if !strings.HasPrefix(value, htmlMatcherPlaceholderPrefix) || !strings.HasSuffix(value, "__") {
		return nil
	}

	expr, ok := matchers[value]
	if !ok {
		return nil
	}

	matcher, err := parseMatcher(expr)
	if err != nil {
		return nil
	}

	return matcher
}

type placeholderPos struct {
	start int
	end   int
	expr  string
}

func parsetemplateString(value string, matchers map[string]string) templateString {
	original := buildOriginalDisplay(value, matchers)
	positions := findPlaceholderPositions(value, matchers)
	segments := buildSegments(value, positions)

	ts := templateString{
		Segments: segments,
		Original: original,
	}

	// Pre-compile regex for performance.
	_ = ts.buildRegex()

	return ts
}

func buildOriginalDisplay(value string, matchers map[string]string) string {
	original := value
	for placeholder, expr := range matchers {
		original = strings.ReplaceAll(original, placeholder, "{{"+expr+"}}")
	}

	return original
}

// findPlaceholderPositions finds all placeholder positions in a value, sorted by start index.
func findPlaceholderPositions(value string, matchers map[string]string) []placeholderPos {
	var positions []placeholderPos

	for placeholder, expr := range matchers {
		idx := 0

		for {
			pos := strings.Index(value[idx:], placeholder)
			if pos == -1 {
				break
			}

			absPos := idx + pos
			positions = append(positions, placeholderPos{
				start: absPos,
				end:   absPos + len(placeholder),
				expr:  expr,
			})
			idx = absPos + len(placeholder)
		}
	}

	sort.Slice(positions, func(i, j int) bool {
		return positions[i].start < positions[j].start
	})

	return positions
}

func buildSegments(value string, positions []placeholderPos) []templateSegment {
	var segments []templateSegment

	lastEnd := 0

	for _, pos := range positions {
		if pos.start > lastEnd {
			segments = append(segments, templateSegment{
				Literal: value[lastEnd:pos.start],
			})
		}

		matcher, err := parseMatcher(pos.expr)
		if err == nil {
			segments = append(segments, templateSegment{
				Matcher: matcher,
			})
		}

		lastEnd = pos.end
	}

	if lastEnd < len(value) {
		segments = append(segments, templateSegment{
			Literal: value[lastEnd:],
		})
	}

	return segments
}
