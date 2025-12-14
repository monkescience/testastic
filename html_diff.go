package testastic

import (
	"fmt"
	"sort"
	"strings"
)

// nilValueDisplay is the string representation for nil values in output.
const nilValueDisplay = "(nil)"

// nilTypeName is the type name for nil values.
const nilTypeName = "nil"

// FormatHTMLDiff formats a slice of HTML differences into a human-readable string.
//
//nolint:dupl // Similar structure to FormatDiff is intentional for consistency.
func FormatHTMLDiff(diffs []HTMLDifference) string {
	if len(diffs) == 0 {
		return ""
	}

	var sb strings.Builder

	if len(diffs) == 1 {
		sb.WriteString("HTML mismatch at 1 path:\n")
	} else {
		sb.WriteString(fmt.Sprintf("HTML mismatch at %d paths:\n", len(diffs)))
	}

	for _, d := range diffs {
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", d.Path))

		switch d.Type {
		case DiffAdded:
			sb.WriteString("    expected: (missing)\n")
			sb.WriteString(fmt.Sprintf("    actual:   %s\n", formatHTMLValue(d.Actual)))

		case DiffRemoved:
			sb.WriteString(fmt.Sprintf("    expected: %s\n", formatHTMLValue(d.Expected)))
			sb.WriteString("    actual:   (missing)\n")

		case DiffTypeMismatch:
			sb.WriteString(fmt.Sprintf("    expected: %s (type: %s)\n", formatHTMLValue(d.Expected), typeOfHTML(d.Expected)))
			sb.WriteString(fmt.Sprintf("    actual:   %s (type: %s)\n", formatHTMLValue(d.Actual), typeOfHTML(d.Actual)))

		case DiffChanged, DiffMatcherFailed:
			sb.WriteString(fmt.Sprintf("    expected: %s\n", formatHTMLValue(d.Expected)))
			sb.WriteString(fmt.Sprintf("    actual:   %s\n", formatHTMLValue(d.Actual)))
		}
	}

	return sb.String()
}

// FormatHTMLDiffInline generates a git-style inline diff between expected and actual HTML.
// Uses the same format as JSON diff.
func FormatHTMLDiffInline(expected, actual *HTMLNode) string {
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

// renderPrettyHTML renders an HTMLNode tree as formatted HTML string.
//
//nolint:gocognit,funlen // HTML rendering requires handling multiple cases and statements.
func renderPrettyHTML(node *HTMLNode, indent int) string {
	if node == nil {
		return ""
	}

	var sb strings.Builder

	indentStr := strings.Repeat("  ", indent)

	switch node.Type {
	case HTMLElement:
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
				sb.WriteString(getString(val))
				sb.WriteString("\"")
			}
		}

		if isVoidElement(node.Tag) {
			sb.WriteString(">")

			return sb.String()
		}

		sb.WriteString(">")

		// Inline text content for single-text children.
		if len(node.Children) == 1 && node.Children[0].Type == HTMLText {
			text := getTextContent(node.Children[0])
			sb.WriteString(text)
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

	case HTMLText:
		text := getTextContent(node)
		if strings.TrimSpace(text) != "" {
			sb.WriteString(indentStr)
			sb.WriteString(strings.TrimSpace(text))
		}

	case HTMLComment:
		sb.WriteString(indentStr)
		sb.WriteString("<!-- ")
		sb.WriteString(getString(node.Text))
		sb.WriteString(" -->")

	case HTMLDoctype:
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

// formatHTMLValue formats a value for display in HTML diff output.
func formatHTMLValue(v any) string {
	if v == nil {
		return nilValueDisplay
	}

	switch val := v.(type) {
	case string:
		if len(val) > maxDisplayLineLen {
			return fmt.Sprintf("%q...", val[:maxDisplayLineLen-3])
		}

		return fmt.Sprintf("%q", val)

	case Matcher:
		return val.String()

	default:
		s := fmt.Sprintf("%v", val)
		if len(s) > maxDisplayLineLen {
			return s[:maxDisplayLineLen-3] + "..."
		}

		return s
	}
}

// typeOfHTML returns a human-readable type name for an HTML value.
func typeOfHTML(v any) string {
	if v == nil {
		return nilTypeName
	}

	switch v.(type) {
	case string:
		return "string"
	case Matcher:
		return "matcher"
	default:
		return fmt.Sprintf("%T", v)
	}
}
