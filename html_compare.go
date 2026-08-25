package testastic

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const maxTextDisplayLen = 30

const nilDisplay = "(nil)"

var (
	htmlAnyIntRegex   = regexp.MustCompile(`^-?\d+$`)
	htmlAnyFloatRegex = regexp.MustCompile(`^-?\d+\.?\d*$`)
)

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
//nolint:funlen // Node-type dispatch is clearer in one switch.
func comparehtmlNodes(expected, actual *htmlNode, path string, cfg *config) []htmlDifference {
	if cfg.isHTMLFieldIgnored(path, expected.Tag) {
		return nil
	}

	if cfg.isElementIgnored(expected.Tag) {
		return nil
	}

	if diffs, handled := compareHTMLTextMatcher(expected, actual, path, cfg); handled {
		return diffs
	}

	if expected.Type != actual.Type {
		return htmlTypeMismatch(path, expected, actual)
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
		diffs = append(diffs, compareHTMLCommentNode(expected, actual, path, cfg)...)

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
			if !matchStringMatcher(m, actStr) {
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

	matches := findUnorderedMatches(expected, actual, func(expIdx int, act *htmlNode) bool {
		childPath := buildChildPath(path, expected[expIdx], expIdx)

		return len(comparehtmlNodes(expected[expIdx], act, childPath, cfg)) == 0
	})

	if len(matches.unmatchedExpected) == 0 {
		return nil
	}

	var diffs []htmlDifference

	for i, idx := range matches.unmatchedExpected {
		childPath := buildChildPath(path, expected[idx], idx)

		var actualDesc any
		if i < len(matches.unusedActual) {
			actualDesc = describeNode(actual[matches.unusedActual[i]])
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

		runes := []rune(text)
		if len(runes) > maxTextDisplayLen {
			return fmt.Sprintf("%q...", string(runes[:maxTextDisplayLen]))
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

// compareHTMLTextMatcher handles an expected text node whose content is a
// matcher or template. It returns handled=false when the expected node is not
// a text matcher so the caller falls through to literal comparison. A text
// matcher only applies to an actual text node, otherwise it is a type mismatch.
func compareHTMLTextMatcher(expected, actual *htmlNode, path string, cfg *config) ([]htmlDifference, bool) {
	if expected.Type != htmlText {
		return nil, false
	}

	if m, ok := expected.Text.(Matcher); ok {
		if isIgnore(m) {
			return nil, true
		}

		if actual.Type != htmlText {
			return htmlTypeMismatch(path, expected, actual), true
		}

		actualText := htmlActualText(actual, cfg)
		if !matchStringMatcher(m, actualText) {
			return matcherFailedDiff(path, m.String(), actualText), true
		}

		return nil, true
	}

	if ts, ok := expected.Text.(templateString); ok {
		if actual.Type != htmlText {
			return htmlTypeMismatch(path, expected, actual), true
		}

		actualText := htmlActualText(actual, cfg)
		if !ts.Match(actualText) {
			return matcherFailedDiff(path, ts.String(), actualText), true
		}

		return nil, true
	}

	return nil, false
}

// compareHTMLCommentNode compares two comment nodes, resolving any matcher or
// template embedded in the expected comment.
func compareHTMLCommentNode(expected, actual *htmlNode, path string, cfg *config) []htmlDifference {
	if cfg.IgnoreComments {
		return nil
	}

	if m, ok := expected.Text.(Matcher); ok {
		if isIgnore(m) {
			return nil
		}

		actComment := getString(actual.Text)
		if !matchStringMatcher(m, actComment) {
			return matcherFailedDiff(path, m.String(), actComment)
		}

		return nil
	}

	if ts, ok := expected.Text.(templateString); ok {
		actComment := getString(actual.Text)
		if !ts.Match(actComment) {
			return matcherFailedDiff(path, ts.String(), actComment)
		}

		return nil
	}

	expComment := getString(expected.Text)
	actComment := getString(actual.Text)

	if expComment != actComment {
		return []htmlDifference{{
			Path:     path,
			Expected: expComment,
			Actual:   actComment,
			Type:     diffChanged,
		}}
	}

	return nil
}

// matcherFailedDiff builds a single matcher-failed difference at path.
func matcherFailedDiff(path, expected, actual string) []htmlDifference {
	return []htmlDifference{{
		Path:     path,
		Expected: expected,
		Actual:   actual,
		Type:     diffMatcherFailed,
	}}
}

// htmlTypeMismatch reports a node-type mismatch between expected and actual.
func htmlTypeMismatch(path string, expected, actual *htmlNode) []htmlDifference {
	return []htmlDifference{{
		Path:     path,
		Expected: describeNodeType(expected.Type),
		Actual:   describeNodeType(actual.Type),
		Type:     diffTypeMismatch,
	}}
}

// htmlActualText returns the text content of a node, applying the same
// whitespace normalization the literal text path uses unless preserved.
func htmlActualText(node *htmlNode, cfg *config) string {
	text := getTextContent(node)
	if !cfg.PreserveWhitespace {
		return normalizeWhitespace(text)
	}

	return text
}

func matchStringMatcher(m Matcher, actual string) bool {
	if m.Match(actual) {
		return true
	}

	switch m.(type) {
	case anyIntMatcher:
		return htmlAnyIntRegex.MatchString(actual)
	case anyFloatMatcher:
		return htmlAnyFloatRegex.MatchString(actual)
	case anyBoolMatcher:
		return actual == strconv.FormatBool(true) || actual == strconv.FormatBool(false)
	default:
		return false
	}
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
