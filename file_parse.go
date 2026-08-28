package testastic

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type lineMatcher struct {
	original       string         // raw line text
	pattern        *regexp.Regexp // nil if no matchers (exact match mode)
	matchers       []Matcher      // matchers found in this line, in order
	captureIndexes []int
	embedded       *embeddedMatcher
}

type fileMatcherPreparer struct{}

func (fileMatcherPreparer) prepare(source string) (preparedMatcherSource, error) {
	return preparedMatcherSource{
		source: source,
		sites: matcherSourceSites(
			source,
			matcherSourceRules{
				unclosedBacktickConsumes:    true,
				unclosedDoubleQuoteConsumes: true,
			},
			rawMatcherPlaceholder,
		),
		placeholder:      "__TESTASTIC_TEXT_MATCHER_",
		collides:         func(candidate string) bool { return strings.Contains(source, candidate) },
		embedded:         true,
		trimWholeMatcher: false,
	}, nil
}

func parseLine(line string) (*lineMatcher, error) {
	program, err := compileMatcherProgram(line, fileMatcherPreparer{})
	if err != nil {
		compileErr, ok := errors.AsType[*matcherCompileError](err)
		if ok {
			return nil, fmt.Errorf("line %q: %w", line, compileErr.cause)
		}

		return nil, fmt.Errorf("line %q: %w", line, err)
	}

	if len(program.sites) == 0 {
		return &lineMatcher{original: line}, nil
	}

	segments, original := program.embeddedSegments(program.sourceForParser())

	embedded, err := newEmbeddedMatcher(original, segments)
	if err != nil {
		return nil, fmt.Errorf("line %q: %w", line, err)
	}

	matchers := make([]Matcher, 0, len(program.sites))
	for _, site := range program.sites {
		matchers = append(matchers, site.matcher)
	}

	captureIndexes := make([]int, 0, len(program.sites))

	for _, segment := range embedded.segments {
		if segment.matcher != nil {
			captureIndexes = append(captureIndexes, segment.captureIndex)
		}
	}

	return &lineMatcher{
		original:       line,
		pattern:        embedded.pattern,
		matchers:       matchers,
		captureIndexes: captureIndexes,
		embedded:       embedded,
	}, nil
}
