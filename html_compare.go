package testastic

import (
	"fmt"
	"sort"
	"strings"
)

// maxTextDisplayLen is the maximum length for displaying text values.
const maxTextDisplayLen = 30

// nilDisplay is the string representation for nil values.
const nilDisplay = "(nil)"

// HTMLDifference represents a single difference between expected and actual HTML.
type HTMLDifference struct {
	Path     string
	Expected any
	Actual   any
	Type     DiffType
}

// compareHTML compares expected and actual HTML nodes.
// Returns a list of differences found.
func compareHTML(expected, actual *HTMLNode, cfg *HTMLConfig) []HTMLDifference {
	if expected == nil && actual == nil {
		return nil
	}

	if expected == nil {
		return []HTMLDifference{{
			Path:     actual.Path,
			Expected: nil,
			Actual:   describeNode(actual),
			Type:     DiffAdded,
		}}
	}

	if actual == nil {
		return []HTMLDifference{{
			Path:     expected.Path,
			Expected: describeNode(expected),
			Actual:   nil,
			Type:     DiffRemoved,
		}}
	}

	return compareHTMLNodes(expected, actual, expected.Path, cfg)
}

// compareHTMLNodes recursively compares two HTML nodes.
//
//nolint:funlen // Complex type dispatch is clearer in one function.
func compareHTMLNodes(expected, actual *HTMLNode, path string, cfg *HTMLConfig) []HTMLDifference {
	// Check if element should be ignored
	if cfg.isElementIgnored(expected.Tag) {
		return nil
	}

	if expected.Type == HTMLText { //nolint:nestif // Matcher handling requires nested conditions.
		if m, ok := expected.Text.(Matcher); ok {
			if IsIgnore(m) {
				return nil
			}

			actualText := getTextContent(actual)
			if !m.Match(actualText) {
				return []HTMLDifference{{
					Path:     path,
					Expected: m.String(),
					Actual:   actualText,
					Type:     DiffMatcherFailed,
				}}
			}

			return nil
		}

		if ts, ok := expected.Text.(TemplateString); ok {
			actualText := getTextContent(actual)
			if !ts.Match(actualText) {
				return []HTMLDifference{{
					Path:     path,
					Expected: ts.String(),
					Actual:   actualText,
					Type:     DiffMatcherFailed,
				}}
			}

			return nil
		}
	}

	// Compare node types
	if expected.Type != actual.Type {
		return []HTMLDifference{{
			Path:     path,
			Expected: describeNodeType(expected.Type),
			Actual:   describeNodeType(actual.Type),
			Type:     DiffTypeMismatch,
		}}
	}

	var diffs []HTMLDifference

	switch expected.Type {
	case HTMLElement:
		// Compare tag names
		if !strings.EqualFold(expected.Tag, actual.Tag) {
			diffs = append(diffs, HTMLDifference{
				Path:     path,
				Expected: fmt.Sprintf("<%s>", expected.Tag),
				Actual:   fmt.Sprintf("<%s>", actual.Tag),
				Type:     DiffChanged,
			})

			return diffs // Different tags, no point comparing further
		}

		// Compare attributes
		diffs = append(diffs, compareHTMLAttributes(expected.Attributes, actual.Attributes, path, cfg)...)

		// Compare children
		diffs = append(diffs, compareHTMLChildren(expected.Children, actual.Children, path, cfg)...)

	case HTMLText:
		expText := getTextContent(expected)
		actText := getTextContent(actual)

		// Normalize whitespace unless preserving
		if !cfg.PreserveWhitespace {
			expText = normalizeWhitespace(expText)
			actText = normalizeWhitespace(actText)
		}

		if expText != actText {
			diffs = append(diffs, HTMLDifference{
				Path:     path,
				Expected: expText,
				Actual:   actText,
				Type:     DiffChanged,
			})
		}

	case HTMLComment:
		if !cfg.IgnoreComments {
			expComment := getString(expected.Text)
			actComment := getString(actual.Text)

			if expComment != actComment {
				diffs = append(diffs, HTMLDifference{
					Path:     path,
					Expected: expComment,
					Actual:   actComment,
					Type:     DiffChanged,
				})
			}
		}

	case HTMLDoctype:
		if !strings.EqualFold(expected.Tag, actual.Tag) {
			diffs = append(diffs, HTMLDifference{
				Path:     path,
				Expected: expected.Tag,
				Actual:   actual.Tag,
				Type:     DiffChanged,
			})
		}
	}

	return diffs
}

// compareHTMLAttributes compares HTML element attributes.
//
//nolint:funlen // Attribute comparison needs explicit handling for all cases.
func compareHTMLAttributes(expected, actual map[string]any, path string, cfg *HTMLConfig) []HTMLDifference {
	var diffs []HTMLDifference

	// Check expected attributes
	for name, expVal := range expected {
		if cfg.isAttributeIgnored(path, name) {
			continue
		}

		// Check if expected value is an ignore matcher
		if m, ok := expVal.(Matcher); ok && IsIgnore(m) {
			continue
		}

		attrPath := path + " @" + name
		actVal, exists := actual[name]

		if !exists {
			diffs = append(diffs, HTMLDifference{
				Path:     attrPath,
				Expected: formatAttrValue(expVal),
				Actual:   nil,
				Type:     DiffRemoved,
			})

			continue
		}

		if m, ok := expVal.(Matcher); ok {
			actStr := getString(actVal)
			if !m.Match(actStr) {
				diffs = append(diffs, HTMLDifference{
					Path:     attrPath,
					Expected: m.String(),
					Actual:   actStr,
					Type:     DiffMatcherFailed,
				})
			}

			continue
		}

		if ts, ok := expVal.(TemplateString); ok {
			actStr := getString(actVal)
			if !ts.Match(actStr) {
				diffs = append(diffs, HTMLDifference{
					Path:     attrPath,
					Expected: ts.String(),
					Actual:   actStr,
					Type:     DiffMatcherFailed,
				})
			}

			continue
		}

		expStr := getString(expVal)
		actStr := getString(actVal)

		if expStr != actStr {
			diffs = append(diffs, HTMLDifference{
				Path:     attrPath,
				Expected: expStr,
				Actual:   actStr,
				Type:     DiffChanged,
			})
		}
	}

	// Check for extra attributes in actual
	for name, actVal := range actual {
		if cfg.isAttributeIgnored(path, name) {
			continue
		}

		if _, exists := expected[name]; !exists {
			diffs = append(diffs, HTMLDifference{
				Path:     path + " @" + name,
				Expected: nil,
				Actual:   formatAttrValue(actVal),
				Type:     DiffAdded,
			})
		}
	}

	return diffs
}

// compareHTMLChildren compares child nodes of an HTML element.
func compareHTMLChildren(expected, actual []*HTMLNode, path string, cfg *HTMLConfig) []HTMLDifference {
	// Filter out nodes that should be ignored
	expFiltered := filterSignificantChildren(expected, cfg)
	actFiltered := filterSignificantChildren(actual, cfg)

	if cfg.shouldIgnoreChildOrder(path) {
		return compareChildrenUnordered(expFiltered, actFiltered, path, cfg)
	}

	return compareChildrenOrdered(expFiltered, actFiltered, path, cfg)
}

// compareChildrenOrdered compares children where order matters.
func compareChildrenOrdered(expected, actual []*HTMLNode, path string, cfg *HTMLConfig) []HTMLDifference {
	var diffs []HTMLDifference

	maxLen := max(len(expected), len(actual))

	for i := range maxLen {
		switch {
		case i >= len(expected):
			childPath := buildChildPath(path, actual[i], i)
			diffs = append(diffs, HTMLDifference{
				Path:     childPath,
				Expected: nil,
				Actual:   describeNode(actual[i]),
				Type:     DiffAdded,
			})
		case i >= len(actual):
			childPath := buildChildPath(path, expected[i], i)
			diffs = append(diffs, HTMLDifference{
				Path:     childPath,
				Expected: describeNode(expected[i]),
				Actual:   nil,
				Type:     DiffRemoved,
			})
		default:
			childPath := buildChildPath(path, expected[i], i)
			diffs = append(diffs, compareHTMLNodes(expected[i], actual[i], childPath, cfg)...)
		}
	}

	return diffs
}

// compareChildrenUnordered compares children where order doesn't matter.
//
//nolint:funlen // Unordered comparison requires explicit matching logic.
func compareChildrenUnordered(expected, actual []*HTMLNode, path string, cfg *HTMLConfig) []HTMLDifference {
	if len(expected) != len(actual) {
		return []HTMLDifference{{
			Path:     path,
			Expected: fmt.Sprintf("%d children", len(expected)),
			Actual:   fmt.Sprintf("%d children", len(actual)),
			Type:     DiffChanged,
		}}
	}

	// Try to find a matching element for each expected element
	used := make([]bool, len(actual))

	var unmatched []int

	for i, exp := range expected {
		found := false

		for j, act := range actual {
			if used[j] {
				continue
			}

			if len(compareHTMLNodes(exp, act, path, cfg)) == 0 {
				used[j] = true
				found = true

				break
			}
		}

		if !found {
			unmatched = append(unmatched, i)
		}
	}

	if len(unmatched) > 0 {
		var unusedActual []int

		for i, u := range used {
			if !u {
				unusedActual = append(unusedActual, i)
			}
		}

		var diffs []HTMLDifference

		for i, idx := range unmatched {
			childPath := buildChildPath(path, expected[idx], idx)

			var actualDesc any
			if i < len(unusedActual) {
				actualDesc = describeNode(actual[unusedActual[i]])
			}

			diffs = append(diffs, HTMLDifference{
				Path:     childPath,
				Expected: describeNode(expected[idx]),
				Actual:   actualDesc,
				Type:     DiffChanged,
			})
		}

		return diffs
	}

	return nil
}

// filterSignificantChildren filters out insignificant nodes.
func filterSignificantChildren(nodes []*HTMLNode, cfg *HTMLConfig) []*HTMLNode {
	result := make([]*HTMLNode, 0, len(nodes))

	for _, node := range nodes {
		if node == nil {
			continue
		}

		// Skip ignored elements
		if node.Type == HTMLElement && cfg.isElementIgnored(node.Tag) {
			continue
		}

		// Skip comments if ignored
		if node.Type == HTMLComment && cfg.IgnoreComments {
			continue
		}

		// Skip whitespace-only text nodes unless preserving whitespace
		if node.Type == HTMLText && !cfg.PreserveWhitespace {
			text := getTextContent(node)
			if strings.TrimSpace(text) == "" {
				continue
			}
		}

		result = append(result, node)
	}

	return result
}

// buildChildPath builds a path for a child node.
func buildChildPath(parentPath string, node *HTMLNode, _ int) string {
	if node.Type == HTMLText {
		return parentPath + " (text)"
	}

	if node.Type == HTMLComment {
		return parentPath + " (comment)"
	}

	if parentPath == "" {
		return node.Tag
	}

	return fmt.Sprintf("%s > %s", parentPath, node.Tag)
}

// describeNode returns a human-readable description of a node.
func describeNode(node *HTMLNode) string {
	if node == nil {
		return nilDisplay
	}

	switch node.Type {
	case HTMLElement:
		return fmt.Sprintf("<%s>", node.Tag)
	case HTMLText:
		text := getTextContent(node)
		if len(text) > maxTextDisplayLen {
			return fmt.Sprintf("%q...", text[:maxTextDisplayLen])
		}

		return fmt.Sprintf("%q", text)
	case HTMLComment:
		return "<!-- comment -->"
	case HTMLDoctype:
		return "<!DOCTYPE>"
	default:
		return "(unknown)"
	}
}

// describeNodeType returns a human-readable type name.
func describeNodeType(t HTMLNodeType) string {
	switch t {
	case HTMLElement:
		return "element"
	case HTMLText:
		return "text"
	case HTMLComment:
		return "comment"
	case HTMLDoctype:
		return "doctype"
	default:
		return "unknown"
	}
}

// getTextContent extracts text content from a node.
func getTextContent(node *HTMLNode) string {
	if node == nil {
		return ""
	}

	if s, ok := node.Text.(string); ok {
		return s
	}

	if m, ok := node.Text.(Matcher); ok {
		return m.String()
	}

	if ts, ok := node.Text.(TemplateString); ok {
		return ts.String()
	}

	return ""
}

// getString converts a value to string.
func getString(v any) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	if m, ok := v.(Matcher); ok {
		return m.String()
	}

	if ts, ok := v.(TemplateString); ok {
		return ts.String()
	}

	return fmt.Sprintf("%v", v)
}

// formatAttrValue formats an attribute value for display.
func formatAttrValue(v any) string {
	if v == nil {
		return nilDisplay
	}

	if m, ok := v.(Matcher); ok {
		return m.String()
	}

	return fmt.Sprintf("%q", getString(v))
}

// normalizeWhitespace collapses whitespace in text.
func normalizeWhitespace(s string) string {
	// Collapse multiple whitespace to single space
	fields := strings.Fields(s)

	return strings.Join(fields, " ")
}

// sortHTMLDiffs sorts differences by path for consistent output.
func sortHTMLDiffs(diffs []HTMLDifference) {
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})
}
