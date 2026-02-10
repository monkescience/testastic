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
var fileExprRegex = regexp.MustCompile(`\{\{((?:[^}` + "`" + `]+|` + "`" + `[^` + "`" + `]*` + "`" + `)+)\}\}`)

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

		matcher, err := parseMatcher(expr)
		if err != nil {
			return nil, fmt.Errorf("line %q: %w", line, err)
		}

		matchers = append(matchers, matcher)

		textPattern := getMatcherTextPattern(expr, matcher)
		patternBuilder.WriteString("(" + textPattern + ")")

		lastEnd = end
	}

	// Add any remaining literal text after the last matcher
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

		return pattern
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

	// Fallback: match anything
	return `.+`
}
