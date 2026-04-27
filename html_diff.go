package testastic

import (
	"html"
	"sort"
	"strings"
)

// formatHTMLDiffInline generates a git-style inline diff between expected and actual HTML.
// Uses the same format as JSON diff.
func formatHTMLDiffInline(expected, actual *htmlNode) string {
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
		if node.Tag == "#document" {
			for i, child := range node.Children {
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

		// Inline text content for single-text children.
		if len(node.Children) == 1 && node.Children[0].Type == htmlText {
			text := getTextContent(node.Children[0])
			sb.WriteString(html.EscapeString(text))
			sb.WriteString("</")
			sb.WriteString(node.Tag)
			sb.WriteString(">")

			return sb.String()
		}

		if len(node.Children) > 0 {
			for _, child := range node.Children {
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
