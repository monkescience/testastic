package testastic

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Matcher parsing errors used internally by parseMatcher.
var (
	// errInvalidRegexSyntax is returned when a regex matcher has invalid syntax.
	// Example: {{regex `[invalid`}} - unclosed character class.
	errInvalidRegexSyntax = errors.New("invalid regex syntax")

	// errInvalidOneOfSyntax is returned when a oneOf matcher has invalid arguments.
	// Arguments must be quoted strings: {{oneOf "a" "b"}}, not {{oneOf a b}}.
	errInvalidOneOfSyntax = errors.New("invalid oneOf syntax")

	// errUnknownMatcher is returned when a matcher expression is not recognized.
	// This can occur if a custom matcher was not registered with [RegisterMatcher].
	errUnknownMatcher = errors.New("unknown matcher")
)

// Matcher defines the interface for custom value matching.
type Matcher interface {
	// Match returns true if the actual value matches the expected pattern.
	Match(actual any) bool
	// String returns a description of what this matcher expects.
	String() string
}

type anyStringMatcher struct{}

func (m anyStringMatcher) Match(actual any) bool {
	_, ok := actual.(string)

	return ok
}

func (m anyStringMatcher) String() string {
	return "{{anyString}}" //nolint:goconst // Canonical matcher representation, same pattern as other matchers.
}

// anyIntMatcher matches any integer value (including float64 with no decimal part).
type anyIntMatcher struct{}

func (m anyIntMatcher) Match(actual any) bool {
	switch v := actual.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float64:
		return v == float64(int64(v))
	case float32:
		return v == float32(int32(v))
	}

	return false
}

func (m anyIntMatcher) String() string {
	return "{{anyInt}}"
}

// anyFloatMatcher matches any numeric value.
type anyFloatMatcher struct{}

func (m anyFloatMatcher) Match(actual any) bool {
	switch actual.(type) {
	case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	}

	return false
}

func (m anyFloatMatcher) String() string {
	return "{{anyFloat}}"
}

type anyBoolMatcher struct{}

func (m anyBoolMatcher) Match(actual any) bool {
	_, ok := actual.(bool)

	return ok
}

func (m anyBoolMatcher) String() string {
	return "{{anyBool}}"
}

// anyValueMatcher matches any value including null.
type anyValueMatcher struct{}

func (m anyValueMatcher) Match(actual any) bool {
	return true
}

func (m anyValueMatcher) String() string {
	return "{{anyValue}}"
}

// ignoreMatcher indicates a field should be skipped during comparison.
type ignoreMatcher struct{}

func (m ignoreMatcher) Match(actual any) bool {
	return true
}

func (m ignoreMatcher) String() string {
	return "{{ignore}}"
}

func isIgnore(m Matcher) bool {
	_, ok := m.(ignoreMatcher)

	return ok
}

type regexMatcher struct {
	pattern string
	re      *regexp.Regexp
}

func (m *regexMatcher) Match(actual any) bool {
	s, ok := actual.(string)
	if !ok {
		return false
	}

	return m.re.MatchString(s)
}

func (m *regexMatcher) String() string {
	return fmt.Sprintf("{{regex `%s`}}", m.pattern)
}

type oneOfMatcher struct {
	values []any
}

func (m *oneOfMatcher) Match(actual any) bool {
	return slices.Contains(m.values, actual)
}

func (m *oneOfMatcher) String() string {
	return fmt.Sprintf("{{oneOf %v}}", m.values)
}

// anyUUIDMatcher matches UUID strings (RFC 4122).
type anyUUIDMatcher struct {
	re *regexp.Regexp
}

func (m *anyUUIDMatcher) Match(actual any) bool {
	s, ok := actual.(string)
	if !ok {
		return false
	}

	return m.re.MatchString(strings.ToLower(s))
}

func (m *anyUUIDMatcher) String() string {
	return "{{anyUUID}}"
}

// anyDateTimeMatcher matches ISO 8601 datetime strings.
type anyDateTimeMatcher struct {
	re *regexp.Regexp
}

func (m *anyDateTimeMatcher) Match(actual any) bool {
	s, ok := actual.(string)
	if !ok {
		return false
	}

	return m.re.MatchString(s)
}

func (m *anyDateTimeMatcher) String() string {
	return "{{anyDateTime}}"
}

type anyURLMatcher struct {
	re *regexp.Regexp
}

func (m *anyURLMatcher) Match(actual any) bool {
	s, ok := actual.(string)
	if !ok {
		return false
	}

	return m.re.MatchString(s)
}

func (m *anyURLMatcher) String() string {
	return "{{anyURL}}"
}

// Template function constructors for creating matchers.
// These are used by the template parser.

// AnyString returns a [Matcher] that accepts any string value.
func AnyString() Matcher {
	return anyStringMatcher{}
}

// AnyInt returns a [Matcher] that accepts any integer value,
// including float64 values with no fractional part (as produced by JSON unmarshaling).
func AnyInt() Matcher {
	return anyIntMatcher{}
}

// AnyFloat returns a [Matcher] that accepts any numeric value (int or float types).
func AnyFloat() Matcher {
	return anyFloatMatcher{}
}

// AnyBool returns a [Matcher] that accepts any boolean value.
func AnyBool() Matcher {
	return anyBoolMatcher{}
}

// AnyValue returns a matcher that matches any value including null.
func AnyValue() Matcher {
	return anyValueMatcher{}
}

// Ignore returns a matcher that causes the field to be skipped.
func Ignore() Matcher {
	return ignoreMatcher{}
}

// Regex returns a [Matcher] that accepts strings matching the given regular expression.
// Returns an error if the pattern fails to compile.
func Regex(pattern string) (Matcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	return &regexMatcher{pattern: pattern, re: re}, nil
}

// OneOf returns a [Matcher] that accepts values equal to one of the given values.
func OneOf(values ...any) Matcher {
	return &oneOfMatcher{values: values}
}

var (
	// uuidRegex matches lowercase hex UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
	uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	// dateTimeRegex matches ISO 8601 dates with optional time: YYYY-MM-DD[T]HH:MM:SS[.fractional][Z|+HH:MM].
	dateTimeRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}([T ]\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?)?$`)
	// urlRegex matches HTTP/HTTPS URLs: scheme://host with optional path.
	urlRegex = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
)

// AnyUUID returns a matcher that matches UUID strings (RFC 4122).
func AnyUUID() Matcher {
	return &anyUUIDMatcher{re: uuidRegex}
}

// AnyDateTime returns a matcher that matches ISO 8601 datetime strings.
func AnyDateTime() Matcher {
	return &anyDateTimeMatcher{re: dateTimeRegex}
}

// AnyURL returns a [Matcher] that accepts strings matching common HTTP/HTTPS URL patterns.
func AnyURL() Matcher {
	return &anyURLMatcher{re: urlRegex}
}

// MatcherFactory creates a [Matcher] from arguments extracted from a template expression.
// The args parameter contains everything after the matcher name in the template.
//
// For {{myMatcher}}, args is an empty string.
// For {{myMatcher foo bar}}, args is "foo bar".
//
// Return an error if the arguments are invalid. The error will be wrapped
// and reported during template parsing.
type MatcherFactory func(args string) (Matcher, error)

var customMatchers = make(map[string]MatcherFactory)

// RegisterMatcher registers a custom matcher factory with the given name.
// Once registered, the matcher can be used in expected files as {{name}} or {{name args}}.
//
// Registration should typically happen in TestMain or an init function
// to ensure matchers are available before tests run.
//
// Example without arguments:
//
//	testastic.RegisterMatcher("orderID", func(args string) (testastic.Matcher, error) {
//	    return &orderIDMatcher{}, nil
//	})
//	// Use in expected file: "id": "{{orderID}}"
//
// Example with arguments:
//
//	testastic.RegisterMatcher("minLength", func(args string) (testastic.Matcher, error) {
//	    n, err := strconv.Atoi(strings.TrimSpace(args))
//	    if err != nil {
//	        return nil, fmt.Errorf("minLength requires integer argument: %w", err)
//	    }
//	    return &minLengthMatcher{min: n}, nil
//	})
//	// Use in expected file: "name": "{{minLength 3}}"
//
// Custom matchers must implement the [Matcher] interface.
func RegisterMatcher(name string, factory MatcherFactory) {
	customMatchers[name] = factory
}

// parseMatcher creates a Matcher from a template expression.
// The expression is the content between {{ and }}, without the braces.
func parseMatcher(expr string) (Matcher, error) {
	switch expr {
	case "anyString":
		return AnyString(), nil
	case "anyInt":
		return AnyInt(), nil
	case "anyFloat":
		return AnyFloat(), nil
	case "anyBool":
		return AnyBool(), nil
	case "anyValue":
		return AnyValue(), nil
	case "ignore":
		return Ignore(), nil
	case "anyUUID":
		return AnyUUID(), nil
	case "anyDateTime":
		return AnyDateTime(), nil
	case "anyURL":
		return AnyURL(), nil
	}

	if factory, ok := customMatchers[expr]; ok {
		return factory("")
	}

	const matcherArgParts = 2 // Splits "name args" into at most [name, args].

	parts := strings.SplitN(expr, " ", matcherArgParts)
	if len(parts) == matcherArgParts {
		if factory, ok := customMatchers[parts[0]]; ok {
			return factory(parts[1])
		}
	}

	// Handle regex `pattern`
	if len(expr) > 6 && expr[:6] == "regex " {
		pattern := extractBacktickArg(expr[6:])
		if pattern != "" {
			return Regex(pattern)
		}
		// Try quoted string
		pattern = extractQuotedArg(expr[6:])
		if pattern != "" {
			return Regex(pattern)
		}

		return nil, fmt.Errorf("%w: %s", errInvalidRegexSyntax, expr)
	}

	// Handle oneOf "a" "b" "c"
	if len(expr) > 6 && expr[:6] == "oneOf " {
		values := extractQuotedArgs(expr[6:])
		if len(values) > 0 {
			return OneOf(values...), nil
		}

		return nil, fmt.Errorf("%w: %s", errInvalidOneOfSyntax, expr)
	}

	return nil, fmt.Errorf("%w: %s", errUnknownMatcher, expr)
}

func extractBacktickArg(s string) string {
	s = trimSpace(s)
	if len(s) >= 2 && s[0] == '`' {
		end := indexOf(s[1:], '`')
		if end >= 0 {
			return s[1 : end+1]
		}
	}

	return ""
}

func extractQuotedArg(s string) string {
	s = trimSpace(s)
	if len(s) >= 2 && s[0] == '"' {
		end := indexOf(s[1:], '"')
		if end >= 0 {
			unquoted, err := strconv.Unquote(s[:end+2])
			if err == nil {
				return unquoted
			}

			return s[1 : end+1]
		}
	}

	return ""
}

// extractQuotedArgs extracts multiple quoted strings.
// Handles both regular quotes and JSON-escaped quotes (\" or \\").
func extractQuotedArgs(s string) []any {
	var result []any

	s = trimSpace(s)

	// Handle JSON-escaped quotes: \" or \\"
	if strings.Contains(s, `\"`) || strings.Contains(s, `\\"`) {
		s = strings.ReplaceAll(s, `\\"`, `"`)
		s = strings.ReplaceAll(s, `\"`, `"`)
	}

	for len(s) > 0 && s[0] == '"' {
		end := indexOf(s[1:], '"')
		if end < 0 {
			break
		}

		unquoted, err := strconv.Unquote(s[:end+2])
		if err == nil {
			result = append(result, unquoted)
		} else {
			result = append(result, s[1:end+1])
		}

		s = trimSpace(s[end+2:])
	}

	return result
}

func trimSpace(s string) string {
	start := 0

	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}

func indexOf(s string, c byte) int {
	for i := range len(s) {
		if s[i] == c {
			return i
		}
	}

	return -1
}
