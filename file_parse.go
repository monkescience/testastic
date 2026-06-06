package testastic

import (
	"fmt"
	"regexp"
	"strings"
)

type lineMatcher struct {
	original string         // raw line text
	pattern  *regexp.Regexp // nil if no matchers (exact match mode)
	matchers []Matcher      // matchers found in this line, in order
}

var matcherTextPatterns = map[string]string{
	"anyString":   `.+`,
	"anyInt":      `-?\d+`,
	"anyFloat":    `-?\d+\.?\d*`,
	"anyBool":     `(?:true|false)`,
	"anyUUID":     `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
	"anyDateTime": `\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?)?`,
	"anyURL":      `https?://[^\s]+`,
	"anyValue":    `.+`,
	"ignore":      `.*`,
}

// fileExprRegex matches {{...}} expressions in text (without JSON quote handling).
// The backtick and double-quote alternatives let a quoted/backticked argument
// contain } (e.g. a {n} regex quantifier) without prematurely closing the {{}}.
var fileExprRegex = regexp.MustCompile(
	`\{\{((?:[^}` + "`" + `"]+|` + "`" + `[^` + "`" + `]*` + "`" + `|"[^"]*")+)\}\}`,
)

//nolint:funlen // Sequential pattern builder is clearer as one pass.
func parseLine(line string) (*lineMatcher, error) {
	matches := fileExprRegex.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		return &lineMatcher{
			original: line,
			pattern:  nil,
			matchers: nil,
		}, nil
	}

	var matchers []Matcher

	var patternBuilder strings.Builder

	patternBuilder.WriteString("^")

	lastEnd := 0

	for _, match := range matches {
		// match[0]:match[1] is the full match {{...}}
		// match[2]:match[3] is the captured group (content inside {{}})
		start, end := match[0], match[1]
		exprStart, exprEnd := match[2], match[3]

		if start > lastEnd {
			patternBuilder.WriteString(regexp.QuoteMeta(line[lastEnd:start]))
		}

		expr := trimSpace(line[exprStart:exprEnd])

		if expr == "" {
			// Empty {{ }} is literal text (e.g. documentation about template
			// syntax), not a matcher directive.
			patternBuilder.WriteString(regexp.QuoteMeta(line[start:end]))
			lastEnd = end

			continue
		}

		matcher, err := parseMatcher(expr)
		if err != nil {
			return nil, fmt.Errorf("line %q: %w", line, err)
		}

		matchers = append(matchers, matcher)

		textPattern := getMatcherTextPattern(expr, matcher)
		patternBuilder.WriteString("(" + textPattern + ")")

		lastEnd = end
	}

	if lastEnd < len(line) {
		patternBuilder.WriteString(regexp.QuoteMeta(line[lastEnd:]))
	}

	patternBuilder.WriteString("$")

	pattern, err := regexp.Compile(patternBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("failed to compile pattern for line %q: %w", line, err)
	}

	return &lineMatcher{
		original: line,
		pattern:  pattern,
		matchers: matchers,
	}, nil
}

func getMatcherTextPattern(expr string, _ Matcher) string {
	if strings.HasPrefix(expr, "regex ") {
		pattern := extractBacktickArg(expr[6:])
		if pattern == "" {
			pattern = extractQuotedArg(expr[6:])
		}

		// The whole line is already anchored with ^...$, so a user-supplied
		// leading ^ or trailing $ would become an impossible mid-line anchor.
		return stripLineAnchors(pattern)
	}

	if strings.HasPrefix(expr, "oneOf ") {
		values := extractQuotedArgs(expr[6:])

		var escaped []string

		for _, v := range values {
			if s, ok := v.(string); ok {
				escaped = append(escaped, regexp.QuoteMeta(s))
			}
		}

		return "(?:" + strings.Join(escaped, "|") + ")"
	}

	if p, ok := matcherTextPatterns[expr]; ok {
		return p
	}

	return `.+`
}

// stripLineAnchors removes a single leading ^ and trailing $ from a regex
// pattern. The text line is anchored as a whole, so embedded anchors would be
// unsatisfiable. A trailing escaped \$ is a literal dollar and is preserved.
func stripLineAnchors(pattern string) string {
	pattern = strings.TrimPrefix(pattern, "^")

	if strings.HasSuffix(pattern, "$") && !strings.HasSuffix(pattern, `\$`) {
		pattern = pattern[:len(pattern)-1]
	}

	return pattern
}
