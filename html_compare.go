package testastic

import (
	"fmt"
	"sort"
	"strings"
)

const maxTextDisplayLen = 30

const nilDisplay = "(nil)"

type htmlDifference struct {
	Path     string
	Expected any
	Actual   any
	Type     diffType
}

func compareHTML(expected, actual *htmlNode, cfg *config) []htmlDifference {
	if expected == nil && actual == nil {
		return nil
	}

	if expected == nil {
		return []htmlDifference{{
			Path:     actual.Path,
			Expected: nil,
			Actual:   describeNode(actual),
			Type:     diffAdded,
		}}
	}

	if actual == nil {
		return []htmlDifference{{
			Path:     expected.Path,
			Expected: describeNode(expected),
			Actual:   nil,
			Type:     diffRemoved,
		}}
	}

	return comparehtmlNodes(expected, actual, expected.Path, cfg)
}

// comparehtmlNodes recursively compares two HTML nodes.
//
//nolint:gocognit,funlen // Complex type dispatch is clearer in one function.
func comparehtmlNodes(expected, actual *htmlNode, path string, cfg *config) []htmlDifference {
	if cfg.isHTMLFieldIgnored(path, expected.Tag) {
		return nil
	}

	if cfg.isElementIgnored(expected.Tag) {
		return nil
	}

	if expected.Type == htmlText { //nolint:nestif // Matcher handling requires nested conditions.
		if m, ok := expected.Text.(Matcher); ok {
			if isIgnore(m) {
				return nil
			}

			actualText := getTextContent(actual)
			if !m.Match(actualText) {
				return []htmlDifference{{
					Path:     path,
					Expected: m.String(),
					Actual:   actualText,
					Type:     diffMatcherFailed,
				}}
			}

			return nil
		}

		if ts, ok := expected.Text.(templateString); ok {
			actualText := getTextContent(actual)
			if !ts.Match(actualText) {
				return []htmlDifference{{
					Path:     path,
					Expected: ts.String(),
					Actual:   actualText,
					Type:     diffMatcherFailed,
				}}
			}

			return nil
		}
	}

	if expected.Type != actual.Type {
		return []htmlDifference{{
			Path:     path,
			Expected: describeNodeType(expected.Type),
			Actual:   describeNodeType(actual.Type),
			Type:     diffTypeMismatch,
		}}
	}

	var diffs []htmlDifference

	switch expected.Type {
	case htmlElement:
		if !strings.EqualFold(expected.Tag, actual.Tag) {
			diffs = append(diffs, htmlDifference{
				Path:     path,
				Expected: fmt.Sprintf("<%s>", expected.Tag),
				Actual:   fmt.Sprintf("<%s>", actual.Tag),
				Type:     diffChanged,
			})

			return diffs // Different tags, no point comparing further
		}

		diffs = append(diffs, compareHTMLAttributes(expected.Attributes, actual.Attributes, path, cfg)...)
		diffs = append(diffs, compareHTMLChildren(expected.Children, actual.Children, path, cfg)...)

	case htmlText:
		expText := getTextContent(expected)
		actText := getTextContent(actual)

		if !cfg.PreserveWhitespace {
			expText = normalizeWhitespace(expText)
			actText = normalizeWhitespace(actText)
		}

		if expText != actText {
			diffs = append(diffs, htmlDifference{
				Path:     path,
				Expected: expText,
				Actual:   actText,
				Type:     diffChanged,
			})
		}

	case htmlComment:
		if !cfg.IgnoreComments {
			expComment := getString(expected.Text)
			actComment := getString(actual.Text)

			if expComment != actComment {
				diffs = append(diffs, htmlDifference{
					Path:     path,
					Expected: expComment,
					Actual:   actComment,
					Type:     diffChanged,
				})
			}
		}

	case htmlDoctype:
		if !strings.EqualFold(expected.Tag, actual.Tag) {
			diffs = append(diffs, htmlDifference{
				Path:     path,
				Expected: expected.Tag,
				Actual:   actual.Tag,
				Type:     diffChanged,
			})
		}
	}

	return diffs
}

//nolint:funlen // Attribute comparison needs explicit handling for all cases.
func compareHTMLAttributes(expected, actual map[string]any, path string, cfg *config) []htmlDifference {
	var diffs []htmlDifference

	for name, expVal := range expected {
		if cfg.isAttributeIgnored(path, name) {
			continue
		}

		// Check if expected value is an ignore matcher
		if m, ok := expVal.(Matcher); ok && isIgnore(m) {
			continue
		}

		attrPath := path + " @" + name
		actVal, exists := actual[name]

		if !exists {
			diffs = append(diffs, htmlDifference{
				Path:     attrPath,
				Expected: formatAttrValue(expVal),
				Actual:   nil,
				Type:     diffRemoved,
			})

			continue
		}

		if m, ok := expVal.(Matcher); ok {
			actStr := getString(actVal)
			if !m.Match(actStr) {
				diffs = append(diffs, htmlDifference{
					Path:     attrPath,
					Expected: m.String(),
					Actual:   actStr,
					Type:     diffMatcherFailed,
				})
			}

			continue
		}

		if ts, ok := expVal.(templateString); ok {
			actStr := getString(actVal)
			if !ts.Match(actStr) {
				diffs = append(diffs, htmlDifference{
					Path:     attrPath,
					Expected: ts.String(),
					Actual:   actStr,
					Type:     diffMatcherFailed,
				})
			}

			continue
		}

		expStr := getString(expVal)
		actStr := getString(actVal)

		if expStr != actStr {
			diffs = append(diffs, htmlDifference{
				Path:     attrPath,
				Expected: expStr,
				Actual:   actStr,
				Type:     diffChanged,
			})
		}
	}

	// Check for extra attributes in actual
	for name, actVal := range actual {
		if cfg.isAttributeIgnored(path, name) {
			continue
		}

		if _, exists := expected[name]; !exists {
			diffs = append(diffs, htmlDifference{
				Path:     path + " @" + name,
				Expected: nil,
				Actual:   formatAttrValue(actVal),
				Type:     diffAdded,
			})
		}
	}

	return diffs
}

func compareHTMLChildren(expected, actual []*htmlNode, path string, cfg *config) []htmlDifference {
	expFiltered := filterSignificantChildren(expected, cfg)
	actFiltered := filterSignificantChildren(actual, cfg)

	if cfg.shouldIgnoreChildOrder(path) {
		return compareChildrenUnordered(expFiltered, actFiltered, path, cfg)
	}

	return compareChildrenOrdered(expFiltered, actFiltered, path, cfg)
}

func compareChildrenOrdered(expected, actual []*htmlNode, path string, cfg *config) []htmlDifference {
	var diffs []htmlDifference

	maxLen := max(len(expected), len(actual))

	for i := range maxLen {
		switch {
		case i >= len(expected):
			childPath := buildChildPath(path, actual[i], i)
			diffs = append(diffs, htmlDifference{
				Path:     childPath,
				Expected: nil,
				Actual:   describeNode(actual[i]),
				Type:     diffAdded,
			})
		case i >= len(actual):
			childPath := buildChildPath(path, expected[i], i)
			diffs = append(diffs, htmlDifference{
				Path:     childPath,
				Expected: describeNode(expected[i]),
				Actual:   nil,
				Type:     diffRemoved,
			})
		default:
			childPath := buildChildPath(path, expected[i], i)
			diffs = append(diffs, comparehtmlNodes(expected[i], actual[i], childPath, cfg)...)
		}
	}

	return diffs
}

func compareChildrenUnordered(expected, actual []*htmlNode, path string, cfg *config) []htmlDifference {
	if len(expected) != len(actual) {
		return []htmlDifference{{
			Path:     path,
			Expected: fmt.Sprintf("%d children", len(expected)),
			Actual:   fmt.Sprintf("%d children", len(actual)),
			Type:     diffChanged,
		}}
	}

	unmatched, unusedActual := findUnorderedMatches(expected, actual, func(exp, act *htmlNode) bool {
		return len(comparehtmlNodes(exp, act, path, cfg)) == 0
	})

	if len(unmatched) == 0 {
		return nil
	}

	var diffs []htmlDifference

	for i, idx := range unmatched {
		childPath := buildChildPath(path, expected[idx], idx)

		var actualDesc any
		if i < len(unusedActual) {
			actualDesc = describeNode(actual[unusedActual[i]])
		}

		diffs = append(diffs, htmlDifference{
			Path:     childPath,
			Expected: describeNode(expected[idx]),
			Actual:   actualDesc,
			Type:     diffChanged,
		})
	}

	return diffs
}

func filterSignificantChildren(nodes []*htmlNode, cfg *config) []*htmlNode {
	result := make([]*htmlNode, 0, len(nodes))

	for _, node := range nodes {
		if node == nil {
			continue
		}

		if node.Type == htmlElement && cfg.isElementIgnored(node.Tag) {
			continue
		}

		if cfg.isHTMLFieldIgnored(node.Path, node.Tag) {
			continue
		}

		if node.Type == htmlComment && cfg.IgnoreComments {
			continue
		}

		// Skip whitespace-only text nodes unless preserving whitespace
		if node.Type == htmlText && !cfg.PreserveWhitespace {
			text := getTextContent(node)
			if strings.TrimSpace(text) == "" {
				continue
			}
		}

		result = append(result, node)
	}

	return result
}

func buildChildPath(parentPath string, node *htmlNode, _ int) string {
	if node.Type == htmlText {
		return parentPath + " (text)"
	}

	if node.Type == htmlComment {
		return parentPath + " (comment)"
	}

	if parentPath == "" {
		return node.Tag
	}

	return fmt.Sprintf("%s > %s", parentPath, node.Tag)
}

func describeNode(node *htmlNode) string {
	if node == nil {
		return nilDisplay
	}

	switch node.Type {
	case htmlElement:
		return fmt.Sprintf("<%s>", node.Tag)
	case htmlText:
		text := getTextContent(node)
		if len(text) > maxTextDisplayLen {
			return fmt.Sprintf("%q...", text[:maxTextDisplayLen])
		}

		return fmt.Sprintf("%q", text)
	case htmlComment:
		return "<!-- comment -->"
	case htmlDoctype:
		return "<!DOCTYPE>"
	default:
		return "(unknown)"
	}
}

func describeNodeType(t htmlNodeType) string {
	switch t {
	case htmlElement:
		return "element"
	case htmlText:
		return "text"
	case htmlComment:
		return "comment"
	case htmlDoctype:
		return "doctype"
	default:
		return "unknown"
	}
}

func getTextContent(node *htmlNode) string {
	if node == nil {
		return ""
	}

	if s, ok := node.Text.(string); ok {
		return s
	}

	if m, ok := node.Text.(Matcher); ok {
		return m.String()
	}

	if ts, ok := node.Text.(templateString); ok {
		return ts.String()
	}

	return ""
}

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

	if ts, ok := v.(templateString); ok {
		return ts.String()
	}

	return fmt.Sprintf("%v", v)
}

func formatAttrValue(v any) string {
	if v == nil {
		return nilDisplay
	}

	if m, ok := v.(Matcher); ok {
		return m.String()
	}

	return fmt.Sprintf("%q", getString(v))
}

// normalizeWhitespace collapses runs of whitespace into a single space and trims edges.
func normalizeWhitespace(s string) string {
	fields := strings.Fields(s)

	return strings.Join(fields, " ")
}

func sortHTMLDiffs(diffs []htmlDifference) {
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})
}
