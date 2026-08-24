package testastic

import (
	"html"
	"sort"
	"strings"
)

// formatHTMLDiffInline generates a git-style inline diff between expected and actual HTML.
// Uses the same format as JSON diff.
func formatHTMLDiffInline(expected, actual *htmlNode, cfg *config) string {
	if cfg.IgnoreComments {
		expected = cloneWithoutHTMLComments(expected)
		actual = cloneWithoutHTMLComments(actual)
	}

	expHTML := renderPrettyHTML(expected, 0)
	actHTML := renderPrettyHTML(actual, 0)

	expLines := strings.Split(expHTML, "\n")
	actLines := strings.Split(actHTML, "\n")
	diff := computeDiff(expLines, actLines)

	var sb strings.Builder

	for _, line := range diff {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

func cloneWithoutHTMLComments(node *htmlNode) *htmlNode {
	if node == nil || node.Type == htmlComment {
		return nil
	}

	clone := *node
	clone.Children = make([]*htmlNode, 0, len(node.Children))

	for _, child := range node.Children {
		filtered := cloneWithoutHTMLComments(child)
		if filtered != nil {
			clone.Children = append(clone.Children, filtered)
		}
	}

	return &clone
}

// renderPrettyHTML renders an htmlNode tree as formatted HTML string.
//
//nolint:gocognit,funlen // HTML rendering requires handling multiple cases and statements.
func renderPrettyHTML(node *htmlNode, indent int) string {
	if node == nil {
		return ""
	}

	var sb strings.Builder

	indentStr := strings.Repeat("  ", indent)

	switch node.Type {
	case htmlElement:
		children := filterHTMLRenderChildren(node.Children)

		if node.Tag == htmlDocumentTag {
			for i, child := range children {
				if i > 0 {
					sb.WriteString("\n")
				}

				sb.WriteString(renderPrettyHTML(child, indent))
			}

			return sb.String()
		}

		sb.WriteString(indentStr)
		sb.WriteString("<")
		sb.WriteString(node.Tag)

		// Sort attributes for consistent output.
		if len(node.Attributes) > 0 {
			attrs := make([]string, 0, len(node.Attributes))

			for name := range node.Attributes {
				attrs = append(attrs, name)
			}

			sort.Strings(attrs)

			for _, name := range attrs {
				val := node.Attributes[name]

				sb.WriteString(" ")
				sb.WriteString(name)
				sb.WriteString("=\"")
				sb.WriteString(html.EscapeString(getString(val)))
				sb.WriteString("\"")
			}
		}

		if isVoidElement(node.Tag) {
			sb.WriteString(">")

			return sb.String()
		}

		sb.WriteString(">")

		if len(children) == 1 && children[0].Type == htmlText {
			text := getTextContent(children[0])
			sb.WriteString(html.EscapeString(text))
			sb.WriteString("</")
			sb.WriteString(node.Tag)
			sb.WriteString(">")

			return sb.String()
		}

		if len(children) > 0 {
			for _, child := range children {
				sb.WriteString("\n")
				sb.WriteString(renderPrettyHTML(child, indent+1))
			}

			sb.WriteString("\n")
			sb.WriteString(indentStr)
		}

		sb.WriteString("</")
		sb.WriteString(node.Tag)
		sb.WriteString(">")

	case htmlText:
		text := getTextContent(node)
		if strings.TrimSpace(text) != "" {
			sb.WriteString(indentStr)
			sb.WriteString(html.EscapeString(strings.TrimSpace(text)))
		}

	case htmlComment:
		sb.WriteString(indentStr)
		sb.WriteString("<!-- ")
		sb.WriteString(getString(node.Text))
		sb.WriteString(" -->")

	case htmlDoctype:
		sb.WriteString("<!DOCTYPE ")
		sb.WriteString(node.Tag)
		sb.WriteString(">")
	}

	return sb.String()
}

func filterHTMLRenderChildren(children []*htmlNode) []*htmlNode {
	result := make([]*htmlNode, 0, len(children))

	for _, child := range children {
		if child.Type == htmlText && strings.TrimSpace(getTextContent(child)) == "" {
			continue
		}

		result = append(result, child)
	}

	return result
}

// isVoidElement returns true if the tag is a void element (self-closing).
func isVoidElement(tag string) bool {
	switch strings.ToLower(tag) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input",
		"link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}
