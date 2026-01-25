package testastic

import (
	"regexp"
)

// lineMatcher represents a parsed line from an expected file.
type lineMatcher struct {
	original string         // raw line text
	pattern  *regexp.Regexp // nil if no matchers (exact match mode)
	matchers []Matcher      // matchers found in this line, in order
}

// parseLine parses a single line and extracts any matchers.
func parseLine(line string) (*lineMatcher, error) {
	return &lineMatcher{
		original: line,
		pattern:  nil,
		matchers: nil,
	}, nil
}
