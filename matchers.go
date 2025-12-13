package testastic

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Matcher defines the interface for custom value matching.
type Matcher interface {
	// Match returns true if the actual value matches the expected pattern.
	Match(actual any) bool
	// String returns a description of what this matcher expects.
	String() string
}

// anyStringMatcher matches any string value.
type anyStringMatcher struct{}

func (m anyStringMatcher) Match(actual any) bool {
	_, ok := actual.(string)
	return ok
}

func (m anyStringMatcher) String() string {
	return "{{anyString}}"
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

// anyBoolMatcher matches any boolean value.
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

// IsIgnore returns true if the matcher is an ignore matcher.
func IsIgnore(m Matcher) bool {
	_, ok := m.(ignoreMatcher)
	return ok
}

// regexMatcher matches string values against a regular expression.
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

// oneOfMatcher matches if the value equals one of the allowed values.
type oneOfMatcher struct {
	values []any
}

func (m *oneOfMatcher) Match(actual any) bool {
	for _, v := range m.values {
		if actual == v {
			return true
		}
	}
	return false
}

func (m *oneOfMatcher) String() string {
	return fmt.Sprintf("{{oneOf %v}}", m.values)
}

// Template function constructors for creating matchers.
// These are used by the template parser.

// AnyString returns a matcher that matches any string value.
func AnyString() Matcher {
	return anyStringMatcher{}
}

// AnyInt returns a matcher that matches any integer value.
func AnyInt() Matcher {
	return anyIntMatcher{}
}

// AnyFloat returns a matcher that matches any numeric value.
func AnyFloat() Matcher {
	return anyFloatMatcher{}
}

// AnyBool returns a matcher that matches any boolean value.
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

// Regex returns a matcher that matches strings against a regex pattern.
func Regex(pattern string) (Matcher, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	return &regexMatcher{pattern: pattern, re: re}, nil
}

// OneOf returns a matcher that matches if the value equals one of the given values.
func OneOf(values ...any) Matcher {
	return &oneOfMatcher{values: values}
}

// parseMatcher creates a Matcher from a template expression.
// The expression is the content between {{ and }}.
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
		return nil, fmt.Errorf("invalid regex syntax: %s", expr)
	}

	// Handle oneOf "a" "b" "c"
	if len(expr) > 6 && expr[:6] == "oneOf " {
		values := extractQuotedArgs(expr[6:])
		if len(values) > 0 {
			return OneOf(values...), nil
		}
		return nil, fmt.Errorf("invalid oneOf syntax: %s", expr)
	}

	return nil, fmt.Errorf("unknown matcher: %s", expr)
}

// extractBacktickArg extracts content from backticks: `content`
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

// extractQuotedArg extracts content from quotes: "content"
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
		// Replace escaped quotes with regular quotes
		s = strings.ReplaceAll(s, `\\"`, `"`)
		s = strings.ReplaceAll(s, `\"`, `"`)
	}

	for len(s) > 0 {
		if s[0] == '"' {
			end := indexOf(s[1:], '"')
			if end >= 0 {
				unquoted, err := strconv.Unquote(s[:end+2])
				if err == nil {
					result = append(result, unquoted)
				} else {
					result = append(result, s[1:end+1])
				}
				s = trimSpace(s[end+2:])
			} else {
				break
			}
		} else {
			break
		}
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
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
