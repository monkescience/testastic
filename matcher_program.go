package testastic

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
)

var errEmptyMatcherExpression = errors.New("empty matcher expression")

type matcherSourcePreparer interface {
	prepare(source string) (preparedMatcherSource, error)
}

type preparedMatcherSource struct {
	source           string
	sites            []matcherSourceSite
	placeholder      string
	collides         func(string) bool
	embedded         bool
	trimWholeMatcher bool
}

type matcherSourceSite struct {
	start       int
	end         int
	expression  string
	original    string
	line        int
	column      int
	replacement func(string) string
}

type matcherSourceRules struct {
	includeWholeQuotes          bool
	unclosedBacktickConsumes    bool
	unclosedDoubleQuoteConsumes bool
}

type compiledMatcherSite struct {
	matcherSourceSite
	matcher     Matcher
	placeholder string
	coarse      string
}

type matcherProgram struct {
	original         string
	parserSource     string
	sites            []compiledMatcherSite
	embedded         bool
	trimWholeMatcher bool
}

type matcherCompileError struct {
	expression string
	line       int
	column     int
	cause      error
}

func (e *matcherCompileError) Error() string {
	if e.expression == "" {
		return fmt.Sprintf("line %d, column %d: %v", e.line, e.column, e.cause)
	}

	return fmt.Sprintf(
		"line %d, column %d, expression %q: %v",
		e.line,
		e.column,
		e.expression,
		e.cause,
	)
}

func (e *matcherCompileError) Unwrap() error {
	return e.cause
}

func compileMatcherProgram(source string, preparer matcherSourcePreparer) (matcherProgram, error) {
	prepared, err := preparer.prepare(source)
	if err != nil {
		return matcherProgram{}, err
	}

	sites, err := compileMatcherSites(prepared)
	if err != nil {
		return matcherProgram{}, err
	}

	return matcherProgram{
		original:         source,
		parserSource:     substituteMatcherSites(prepared.source, sites),
		sites:            sites,
		embedded:         prepared.embedded,
		trimWholeMatcher: prepared.trimWholeMatcher,
	}, nil
}

func compileMatcherSites(prepared preparedMatcherSource) ([]compiledMatcherSite, error) {
	sites := make([]compiledMatcherSite, 0, len(prepared.sites))
	nextPlaceholder := 0

	for _, site := range prepared.sites {
		compiled, next, err := compileMatcherSite(site, prepared, nextPlaceholder)
		if err != nil {
			return nil, err
		}

		nextPlaceholder = next

		sites = append(sites, compiled)
	}

	return sites, nil
}

func compileMatcherSite(
	site matcherSourceSite,
	prepared preparedMatcherSource,
	nextPlaceholder int,
) (compiledMatcherSite, int, error) {
	if site.expression == "" {
		return compiledMatcherSite{}, nextPlaceholder, &matcherCompileError{
			expression: site.expression,
			line:       site.line,
			column:     site.column,
			cause:      errEmptyMatcherExpression,
		}
	}

	matcher, err := parseMatcher(site.expression)
	if err != nil {
		return compiledMatcherSite{}, nextPlaceholder, &matcherCompileError{
			expression: site.expression,
			line:       site.line,
			column:     site.column,
			cause:      err,
		}
	}

	coarse, err := embeddedMatcherPattern(matcher)
	if err != nil {
		return compiledMatcherSite{}, nextPlaceholder, &matcherCompileError{
			expression: site.expression,
			line:       site.line,
			column:     site.column,
			cause:      fmt.Errorf("compile embedded matcher: %w", err),
		}
	}

	_, err = regexp.Compile(coarse)
	if err != nil {
		return compiledMatcherSite{}, nextPlaceholder, &matcherCompileError{
			expression: site.expression,
			line:       site.line,
			column:     site.column,
			cause:      fmt.Errorf("compile embedded matcher: %w", err),
		}
	}

	placeholder, nextPlaceholder := allocateMatcherPlaceholder(
		prepared.placeholder,
		nextPlaceholder,
		prepared.collides,
	)

	return compiledMatcherSite{
		matcherSourceSite: site,
		matcher:           matcher,
		placeholder:       placeholder,
		coarse:            coarse,
	}, nextPlaceholder, nil
}

func allocateMatcherPlaceholder(prefix string, start int, collides func(string) bool) (string, int) {
	for index := start; ; index++ {
		placeholder := fmt.Sprintf("%s%d__", prefix, index)
		if !collides(placeholder) {
			return placeholder, index + 1
		}
	}
}

func substituteMatcherSites(source string, sites []compiledMatcherSite) string {
	var result strings.Builder

	lastEnd := 0

	for _, site := range sites {
		result.WriteString(source[lastEnd:site.start])
		result.WriteString(site.replacement(site.placeholder))
		lastEnd = site.end
	}

	result.WriteString(source[lastEnd:])

	return result.String()
}

func matcherSourceSites(
	source string,
	rules matcherSourceRules,
	replacement func(string) string,
) []matcherSourceSite {
	var sites []matcherSourceSite

	cursor := 0
	for cursor < len(source) {
		relativeStart := strings.Index(source[cursor:], "{{")
		if relativeStart < 0 {
			break
		}

		directiveStart := cursor + relativeStart

		directiveEnd := matcherDirectiveEnd(
			source,
			directiveStart+len("{{"),
			rules,
		)
		if directiveEnd < 0 {
			cursor = directiveStart + len("{{")

			continue
		}

		start := directiveStart
		end := directiveEnd

		if rules.includeWholeQuotes && matcherDirectiveIsWholeQuotedValue(source, start, end) {
			start--
			end++
		}

		line, column := matcherSourcePosition(source, directiveStart)
		sites = append(sites, matcherSourceSite{
			start:       start,
			end:         end,
			expression:  trimSpace(source[directiveStart+len("{{") : directiveEnd-len("}}")]),
			original:    source[directiveStart:directiveEnd],
			line:        line,
			column:      column,
			replacement: replacement,
		})
		cursor = directiveEnd
	}

	return sites
}

func rawMatcherPlaceholder(value string) string {
	return value
}

type matcherQuoteMode uint8

const (
	matcherUnquoted matcherQuoteMode = iota
	matcherBacktickQuoted
	matcherDoubleQuoted
	matcherJSONDoubleQuoted
)

func matcherDirectiveEnd(
	source string,
	start int,
	rules matcherSourceRules,
) int {
	mode := matcherUnquoted

	for index := start; index < len(source); index++ {
		character := source[index]
		slashes := precedingBackslashes(source, index)

		if mode != matcherUnquoted {
			mode = matcherQuoteModeAfterCharacter(mode, character, slashes)

			continue
		}

		if strings.HasPrefix(source[index:], "}}") {
			return index + len("}}")
		}

		mode = matcherQuoteModeAt(source, index, character, slashes, rules)
	}

	return -1
}

func matcherQuoteModeAfterCharacter(mode matcherQuoteMode, character byte, slashes int) matcherQuoteMode {
	switch mode {
	case matcherBacktickQuoted:
		if character == '`' {
			return matcherUnquoted
		}
	case matcherDoubleQuoted:
		if character == '"' && slashes%2 == 0 {
			return matcherUnquoted
		}
	case matcherJSONDoubleQuoted:
		if character == '"' && slashes%4 == 1 {
			return matcherUnquoted
		}
	case matcherUnquoted:
		return matcherUnquoted
	}

	return mode
}

func matcherQuoteModeAt(
	source string,
	index int,
	character byte,
	slashes int,
	rules matcherSourceRules,
) matcherQuoteMode {
	switch {
	case character == '`' &&
		(rules.unclosedBacktickConsumes || matcherQuoteCloses(source, index+1, matcherBacktickQuoted)):
		return matcherBacktickQuoted
	case character == '"' && slashes%2 == 0 &&
		(rules.unclosedDoubleQuoteConsumes || matcherQuoteCloses(source, index+1, matcherDoubleQuoted)):
		return matcherDoubleQuoted
	case character == '"' && slashes%4 == 1 &&
		(rules.unclosedDoubleQuoteConsumes || matcherQuoteCloses(source, index+1, matcherJSONDoubleQuoted)):
		return matcherJSONDoubleQuoted
	default:
		return matcherUnquoted
	}
}

func matcherQuoteCloses(source string, start int, mode matcherQuoteMode) bool {
	for index := start; index < len(source); index++ {
		if mode == matcherBacktickQuoted && source[index] == '`' {
			return true
		}

		if source[index] != '"' {
			continue
		}

		slashes := precedingBackslashes(source, index)
		if mode == matcherDoubleQuoted && slashes%2 == 0 {
			return true
		}

		if mode == matcherJSONDoubleQuoted && slashes%4 == 1 {
			return true
		}
	}

	return false
}

func precedingBackslashes(source string, index int) int {
	count := 0

	for index--; index >= 0 && source[index] == '\\'; index-- {
		count++
	}

	return count
}

func matcherDirectiveIsWholeQuotedValue(source string, start, end int) bool {
	if start == 0 || end >= len(source) || source[start-1] != '"' || source[end] != '"' {
		return false
	}

	return precedingBackslashes(source, start-1)%2 == 0 && precedingBackslashes(source, end)%2 == 0
}

func substituteMatcherSourceSites(source string, sites []matcherSourceSite, replacement string) string {
	var result strings.Builder

	lastEnd := 0

	for _, site := range sites {
		result.WriteString(source[lastEnd:site.start])
		result.WriteString(replacement)

		lastEnd = site.end
	}

	result.WriteString(source[lastEnd:])

	return result.String()
}

func matcherSourcePosition(source string, offset int) (int, int) {
	line := 1
	column := 1

	for index, character := range source {
		if index >= offset {
			break
		}

		if character == '\n' {
			line++
			column = 1

			continue
		}

		column++
	}

	return line, column
}

func (p matcherProgram) sourceForParser() string {
	return p.parserSource
}

func (p matcherProgram) resolve(value any) (any, error) {
	if len(p.sites) == 0 {
		return value, nil
	}

	switch typed := value.(type) {
	case map[string]any:
		return p.resolveMap(typed)
	case []any:
		return p.resolveSlice(typed)
	case string:
		return p.resolveString(typed)
	default:
		return value, nil
	}
}

func (p matcherProgram) resolveMap(value map[string]any) (map[string]any, error) {
	resolved := make(map[string]any, len(value))

	for key, child := range value {
		resolvedChild, err := p.resolve(child)
		if err != nil {
			return nil, fmt.Errorf("resolve key %q: %w", key, err)
		}

		resolved[key] = resolvedChild
	}

	return resolved, nil
}

func (p matcherProgram) resolveSlice(value []any) ([]any, error) {
	resolved := make([]any, len(value))

	for index, child := range value {
		resolvedChild, err := p.resolve(child)
		if err != nil {
			return nil, fmt.Errorf("resolve index %d: %w", index, err)
		}

		resolved[index] = resolvedChild
	}

	return resolved, nil
}

func (p matcherProgram) resolveString(value string) (any, error) {
	if matcher := p.wholeMatcher(value); matcher != nil {
		return matcher, nil
	}

	if !p.embedded {
		return value, nil
	}

	segments, original := p.embeddedSegments(value)
	if len(segments) == 0 {
		return value, nil
	}

	matcher, err := newEmbeddedMatcher(original, segments)
	if err != nil {
		return nil, err
	}

	return matcher, nil
}

func (p matcherProgram) wholeMatcher(value string) Matcher {
	for _, site := range p.sites {
		if value == site.placeholder {
			return site.matcher
		}
	}

	if !p.trimWholeMatcher {
		return nil
	}

	trimmed := strings.TrimSpace(value)
	for _, site := range p.sites {
		if trimmed == site.placeholder {
			return site.matcher
		}
	}

	return nil
}

func (p matcherProgram) embeddedSegments(value string) ([]embeddedMatcherSegment, string) {
	segments := make([]embeddedMatcherSegment, 0, len(p.sites)*2+1)

	var original strings.Builder

	cursor := 0
	found := false

	for cursor < len(value) {
		nextIndex, site := p.nextEmbeddedSite(value, cursor)
		if site == nil {
			literal := value[cursor:]
			segments = append(segments, embeddedMatcherSegment{literal: literal})
			original.WriteString(literal)

			break
		}

		if nextIndex > cursor {
			literal := value[cursor:nextIndex]
			segments = append(segments, embeddedMatcherSegment{literal: literal})
			original.WriteString(literal)
		}

		segments = append(segments, embeddedMatcherSegment{
			matcher:  site.matcher,
			original: site.original,
			coarse:   site.coarse,
		})
		original.WriteString(site.original)
		cursor = nextIndex + len(site.placeholder)
		found = true
	}

	if !found {
		return nil, ""
	}

	return segments, original.String()
}

func (p matcherProgram) nextEmbeddedSite(value string, cursor int) (int, *compiledMatcherSite) {
	nextIndex := -1

	var nextSite *compiledMatcherSite

	for index := range p.sites {
		site := &p.sites[index]
		relative := strings.Index(value[cursor:], site.placeholder)

		if relative < 0 {
			continue
		}

		absolute := cursor + relative
		if nextIndex < 0 || absolute < nextIndex {
			nextIndex = absolute
			nextSite = site
		}
	}

	return nextIndex, nextSite
}

type embeddedMatcherSegment struct {
	literal      string
	matcher      Matcher
	original     string
	coarse       string
	captureIndex int
}

type embeddedMatcher struct {
	original string
	pattern  *regexp.Regexp
	segments []embeddedMatcherSegment
}

type matcherCapture struct {
	expected string
	actual   string
	accepted bool
}

type matcherVerdict struct {
	matched  bool
	captures []matcherCapture
}

func newEmbeddedMatcher(original string, segments []embeddedMatcherSegment) (*embeddedMatcher, error) {
	pattern, segments, err := compileEmbeddedPattern(segments)
	if err != nil {
		return nil, err
	}

	return &embeddedMatcher{
		original: original,
		pattern:  pattern,
		segments: segments,
	}, nil
}

func compileEmbeddedPattern(
	segments []embeddedMatcherSegment,
) (*regexp.Regexp, []embeddedMatcherSegment, error) {
	var pattern strings.Builder

	pattern.WriteString("^")

	nextCapture := 1

	for index := range segments {
		segment := &segments[index]
		if segment.matcher == nil {
			pattern.WriteString(regexp.QuoteMeta(segment.literal))

			continue
		}

		coarse, err := regexp.Compile(segment.coarse)
		if err != nil {
			return nil, nil, fmt.Errorf("compile embedded matcher %q: %w", segment.original, err)
		}

		segment.captureIndex = nextCapture
		nextCapture += 1 + coarse.NumSubexp()

		pattern.WriteString("(")
		pattern.WriteString(segment.coarse)
		pattern.WriteString(")")
	}

	pattern.WriteString("$")

	compiled, err := regexp.Compile(pattern.String())
	if err != nil {
		return nil, nil, fmt.Errorf("compile embedded matcher program: %w", err)
	}

	return compiled, segments, nil
}

func (m *embeddedMatcher) Match(actual any) bool {
	value, ok := actual.(string)
	if !ok {
		return false
	}

	return m.match(value).matched
}

func (m *embeddedMatcher) String() string {
	return m.original
}

func (m *embeddedMatcher) match(actual string) matcherVerdict {
	submatches := m.pattern.FindStringSubmatch(actual)
	if submatches == nil {
		return matcherVerdict{}
	}

	verdict := matcherVerdict{
		matched:  true,
		captures: make([]matcherCapture, 0, len(m.segments)),
	}

	for _, segment := range m.segments {
		if segment.matcher == nil {
			continue
		}

		capture := matcherCapture{expected: segment.original}
		if segment.captureIndex < len(submatches) {
			capture.actual = submatches[segment.captureIndex]
			capture.accepted = matchStringMatcher(segment.matcher, capture.actual)
		}

		verdict.captures = append(verdict.captures, capture)
		verdict.matched = verdict.matched && capture.accepted
	}

	return verdict
}

func (m *embeddedMatcher) display(actual string, verdict matcherVerdict) string {
	if verdict.matched {
		return actual
	}

	return m.original
}

func stripLineAnchors(pattern string) string {
	pattern = strings.TrimPrefix(pattern, "^")

	if strings.HasSuffix(pattern, "$") && !strings.HasSuffix(pattern, `\$`) {
		pattern = pattern[:len(pattern)-1]
	}

	return pattern
}

func embeddedMatcherPattern(matcher Matcher) (string, error) {
	switch typed := matcher.(type) {
	case anyIntMatcher:
		return `-?\d+`, nil
	case anyFloatMatcher:
		return `-?\d+\.?\d*`, nil
	case anyBoolMatcher:
		return `(?:true|false)`, nil
	case *regexMatcher:
		return captureRelativeRegexPattern(typed.pattern)
	case *oneOfMatcher:
		return oneOfMatcherPattern(typed.values), nil
	case *anyUUIDMatcher:
		return `(?i:` + stripLineAnchors(typed.re.String()) + `)`, nil
	case *anyDateTimeMatcher:
		return stripLineAnchors(typed.re.String()), nil
	case *anyURLMatcher:
		return stripLineAnchors(typed.re.String()), nil
	default:
		return `(?s:.*)`, nil
	}
}

func captureRelativeRegexPattern(pattern string) (string, error) {
	expression, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", fmt.Errorf("parse regex pattern: %w", err)
	}

	removeMatcherAnchors(expression)

	return expression.String(), nil
}

func removeMatcherAnchors(expression *syntax.Regexp) {
	if expression.Op == syntax.OpBeginLine ||
		expression.Op == syntax.OpEndLine ||
		expression.Op == syntax.OpBeginText ||
		expression.Op == syntax.OpEndText {
		expression.Op = syntax.OpEmptyMatch

		return
	}

	for _, child := range expression.Sub {
		removeMatcherAnchors(child)
	}
}

func oneOfMatcherPattern(values []any) string {
	if len(values) == 0 {
		return `(?s:.*)`
	}

	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, regexp.QuoteMeta(fmt.Sprintf("%v", value)))
	}

	return "(?:" + strings.Join(parts, "|") + ")"
}

var (
	matcherStringIntPattern   = regexp.MustCompile(`^-?\d+$`)
	matcherStringFloatPattern = regexp.MustCompile(`^-?\d+\.?\d*$`)
)

func matchStringMatcher(matcher Matcher, actual string) bool {
	if matcher.Match(actual) {
		return true
	}

	switch matcher.(type) {
	case anyIntMatcher:
		return matcherStringIntPattern.MatchString(actual)
	case anyFloatMatcher:
		return matcherStringFloatPattern.MatchString(actual)
	case anyBoolMatcher:
		return actual == strconv.FormatBool(true) || actual == strconv.FormatBool(false)
	default:
		return false
	}
}

func stringValues(data any) map[string]struct{} {
	values := make(map[string]struct{})
	collectStringValues(data, values)

	return values
}

func collectStringValues(data any, values map[string]struct{}) {
	switch value := data.(type) {
	case map[string]any:
		for _, child := range value {
			collectStringValues(child, values)
		}
	case []any:
		for _, child := range value {
			collectStringValues(child, values)
		}
	case string:
		values[value] = struct{}{}
	}
}
