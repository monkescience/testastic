package testastic

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// fail reports an assertion failure with expected and actual values.
func fail(tb testing.TB, name, expected, actual string) {
	tb.Helper()
	tb.Errorf(
		"testastic: assertion failed\n\n  %s\n    expected: %s\n    actual:   %s",
		name, red(expected), green(actual),
	)
}

// Equal asserts that expected and actual are equal.
func Equal[T comparable](tb testing.TB, expected, actual T) {
	tb.Helper()

	if expected != actual {
		fail(tb, "Equal", formatVal(expected), formatVal(actual))
	}
}

// NotEqual asserts that expected and actual are not equal.
func NotEqual[T comparable](tb testing.TB, unexpected, actual T) {
	tb.Helper()

	if unexpected == actual {
		tb.Errorf(
			"testastic: assertion failed\n\n  NotEqual\n    unexpected: %s\n    actual:     %s",
			red(formatVal(unexpected)), green(formatVal(actual)),
		)
	}
}

// DeepEqual asserts that expected and actual are deeply equal using reflect.DeepEqual.
func DeepEqual[T any](tb testing.TB, expected, actual T) {
	tb.Helper()

	if !reflect.DeepEqual(expected, actual) {
		fail(tb, "DeepEqual", formatVal(expected), formatVal(actual))
	}
}

// Nil asserts that value is nil.
func Nil(tb testing.TB, value any) {
	tb.Helper()

	if !isNil(value) {
		fail(tb, "Nil", "nil", formatVal(value))
	}
}

// NotNil asserts that value is not nil.
func NotNil(tb testing.TB, value any) {
	tb.Helper()

	if isNil(value) {
		fail(tb, "NotNil", "not nil", "nil")
	}
}

// True asserts that value is true.
func True(tb testing.TB, value bool) {
	tb.Helper()

	if !value {
		fail(tb, "True", "true", "false")
	}
}

// False asserts that value is false.
func False(tb testing.TB, value bool) {
	tb.Helper()

	if value {
		fail(tb, "False", "false", "true")
	}
}

// NoError asserts that err is nil.
func NoError(tb testing.TB, err error) {
	tb.Helper()

	if err != nil {
		fail(tb, "NoError", "no error", err.Error())
	}
}

// Error asserts that err is not nil.
func Error(tb testing.TB, err error) {
	tb.Helper()

	if err == nil {
		fail(tb, "Error", "an error", "nil")
	}
}

// ErrorIs asserts that err matches target using errors.Is.
func ErrorIs(tb testing.TB, err, target error) {
	tb.Helper()

	if !errors.Is(err, target) {
		errStr := "nil"
		if err != nil {
			errStr = err.Error()
		}

		fail(tb, "ErrorIs", target.Error(), errStr)
	}
}

// ErrorContains asserts that err contains the given substring.
func ErrorContains(tb testing.TB, err error, substring string) {
	tb.Helper()

	wantMsg := "error containing " + fmt.Sprintf("%q", substring)

	if err == nil {
		fail(tb, "ErrorContains", wantMsg, "nil")

		return
	}

	if !strings.Contains(err.Error(), substring) {
		fail(tb, "ErrorContains", wantMsg, err.Error())
	}
}

// failCmp reports a comparison assertion failure.
func failCmp(tb testing.TB, name, expectOp, actualOp, a, b string) {
	tb.Helper()
	tb.Errorf(
		"testastic: assertion failed\n\n  %s\n    expected: %s %s %s\n    actual:   %s %s %s",
		name, red(a), expectOp, red(b), green(a), actualOp, green(b),
	)
}

// Greater asserts that a > b.
func Greater[T cmp.Ordered](tb testing.TB, a, b T) {
	tb.Helper()

	if a <= b {
		failCmp(tb, "Greater", ">", "<=", formatVal(a), formatVal(b))
	}
}

// GreaterOrEqual asserts that a >= b.
func GreaterOrEqual[T cmp.Ordered](tb testing.TB, a, b T) {
	tb.Helper()

	if a < b {
		failCmp(tb, "GreaterOrEqual", ">=", "<", formatVal(a), formatVal(b))
	}
}

// Less asserts that a < b.
func Less[T cmp.Ordered](tb testing.TB, a, b T) {
	tb.Helper()

	if a >= b {
		failCmp(tb, "Less", "<", ">=", formatVal(a), formatVal(b))
	}
}

// LessOrEqual asserts that a <= b.
func LessOrEqual[T cmp.Ordered](tb testing.TB, a, b T) {
	tb.Helper()

	if a > b {
		failCmp(tb, "LessOrEqual", "<=", ">", formatVal(a), formatVal(b))
	}
}

// Between asserts that minVal <= value <= maxVal.
func Between[T cmp.Ordered](tb testing.TB, value, minVal, maxVal T) {
	tb.Helper()

	if value < minVal || value > maxVal {
		expected := formatVal(minVal) + " <= value <= " + formatVal(maxVal)
		fail(tb, "Between", expected, formatVal(value))
	}
}

// failStr reports a string assertion failure.
func failStr(tb testing.TB, name, label, s, search, status string) {
	tb.Helper()
	tb.Errorf(
		"testastic: assertion failed\n\n  %s\n    string: %s\n    %s: %s (%s)",
		name, green(formatVal(s)), label, red(formatVal(search)), status,
	)
}

// Contains asserts that s contains substring.
func Contains(tb testing.TB, s, substring string) {
	tb.Helper()

	if !strings.Contains(s, substring) {
		failStr(tb, "Contains", "substring", s, substring, "not found")
	}
}

// NotContains asserts that s does not contain substring.
func NotContains(tb testing.TB, s, substring string) {
	tb.Helper()

	if strings.Contains(s, substring) {
		failStr(tb, "NotContains", "substring", s, substring, "found")
	}
}

// HasPrefix asserts that s has the given prefix.
func HasPrefix(tb testing.TB, s, prefix string) {
	tb.Helper()

	if !strings.HasPrefix(s, prefix) {
		failStr(tb, "HasPrefix", "prefix", s, prefix, "not found")
	}
}

// HasSuffix asserts that s has the given suffix.
func HasSuffix(tb testing.TB, s, suffix string) {
	tb.Helper()

	if !strings.HasSuffix(s, suffix) {
		failStr(tb, "HasSuffix", "suffix", s, suffix, "not found")
	}
}

// Matches asserts that s matches the given regular expression pattern.
func Matches(tb testing.TB, s, pattern string) {
	tb.Helper()

	re, err := regexp.Compile(pattern)
	if err != nil {
		tb.Errorf(
			"testastic: assertion failed\n\n  Matches\n    error: invalid pattern %q: %v",
			pattern, err,
		)

		return
	}

	if !re.MatchString(s) {
		failStr(tb, "Matches", "pattern", s, pattern, "no match")
	}
}

// StringEmpty asserts that s is an empty string.
func StringEmpty(tb testing.TB, s string) {
	tb.Helper()

	if s != "" {
		fail(tb, "StringEmpty", `""`, formatVal(s))
	}
}

// StringNotEmpty asserts that s is not an empty string.
func StringNotEmpty(tb testing.TB, s string) {
	tb.Helper()

	if s == "" {
		fail(tb, "StringNotEmpty", "non-empty string", `""`)
	}
}

// isNil checks if a value is nil, handling interface nil correctly.
func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() { //nolint:exhaustive // Only nil-able types need checking.
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}

	return false
}

// formatVal formats a value for display in error messages.
func formatVal(v any) string {
	if v == nil {
		return "nil"
	}

	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
