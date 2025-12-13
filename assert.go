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
func fail(t testing.TB, name, expected, actual string) {
	t.Helper()
	t.Errorf(
		"testastic: assertion failed\n\n  %s\n    expected: %s\n    actual:   %s",
		name, red(expected), green(actual),
	)
}

// Equal asserts that expected and actual are equal.
func Equal[T comparable](t testing.TB, expected, actual T) {
	t.Helper()

	if expected != actual {
		fail(t, "Equal", formatVal(expected), formatVal(actual))
	}
}

// NotEqual asserts that expected and actual are not equal.
func NotEqual[T comparable](t testing.TB, unexpected, actual T) {
	t.Helper()

	if unexpected == actual {
		t.Errorf(
			"testastic: assertion failed\n\n  NotEqual\n    unexpected: %s\n    actual:     %s",
			red(formatVal(unexpected)), green(formatVal(actual)),
		)
	}
}

// DeepEqual asserts that expected and actual are deeply equal using reflect.DeepEqual.
func DeepEqual[T any](t testing.TB, expected, actual T) {
	t.Helper()

	if !reflect.DeepEqual(expected, actual) {
		fail(t, "DeepEqual", formatVal(expected), formatVal(actual))
	}
}

// Nil asserts that value is nil.
func Nil(t testing.TB, value any) {
	t.Helper()

	if !isNil(value) {
		fail(t, "Nil", "nil", formatVal(value))
	}
}

// NotNil asserts that value is not nil.
func NotNil(t testing.TB, value any) {
	t.Helper()

	if isNil(value) {
		fail(t, "NotNil", "not nil", "nil")
	}
}

// True asserts that value is true.
func True(t testing.TB, value bool) {
	t.Helper()

	if !value {
		fail(t, "True", "true", "false")
	}
}

// False asserts that value is false.
func False(t testing.TB, value bool) {
	t.Helper()

	if value {
		fail(t, "False", "false", "true")
	}
}

// NoError asserts that err is nil.
func NoError(t testing.TB, err error) {
	t.Helper()

	if err != nil {
		fail(t, "NoError", "no error", err.Error())
	}
}

// Error asserts that err is not nil.
func Error(t testing.TB, err error) {
	t.Helper()

	if err == nil {
		fail(t, "Error", "an error", "nil")
	}
}

// ErrorIs asserts that err matches target using errors.Is.
func ErrorIs(t testing.TB, err, target error) {
	t.Helper()

	if !errors.Is(err, target) {
		errStr := "nil"
		if err != nil {
			errStr = err.Error()
		}

		fail(t, "ErrorIs", target.Error(), errStr)
	}
}

// ErrorContains asserts that err contains the given substring.
func ErrorContains(t testing.TB, err error, substring string) {
	t.Helper()

	wantMsg := "error containing " + fmt.Sprintf("%q", substring)

	if err == nil {
		fail(t, "ErrorContains", wantMsg, "nil")

		return
	}

	if !strings.Contains(err.Error(), substring) {
		fail(t, "ErrorContains", wantMsg, err.Error())
	}
}

// failCmp reports a comparison assertion failure.
func failCmp(t testing.TB, name, expectOp, actualOp, a, b string) {
	t.Helper()
	t.Errorf(
		"testastic: assertion failed\n\n  %s\n    expected: %s %s %s\n    actual:   %s %s %s",
		name, red(a), expectOp, red(b), green(a), actualOp, green(b),
	)
}

// Greater asserts that a > b.
func Greater[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()

	if a <= b {
		failCmp(t, "Greater", ">", "<=", formatVal(a), formatVal(b))
	}
}

// GreaterOrEqual asserts that a >= b.
func GreaterOrEqual[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()

	if a < b {
		failCmp(t, "GreaterOrEqual", ">=", "<", formatVal(a), formatVal(b))
	}
}

// Less asserts that a < b.
func Less[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()

	if a >= b {
		failCmp(t, "Less", "<", ">=", formatVal(a), formatVal(b))
	}
}

// LessOrEqual asserts that a <= b.
func LessOrEqual[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()

	if a > b {
		failCmp(t, "LessOrEqual", "<=", ">", formatVal(a), formatVal(b))
	}
}

// Between asserts that minVal <= value <= maxVal.
func Between[T cmp.Ordered](t testing.TB, value, minVal, maxVal T) {
	t.Helper()

	if value < minVal || value > maxVal {
		expected := formatVal(minVal) + " <= value <= " + formatVal(maxVal)
		fail(t, "Between", expected, formatVal(value))
	}
}

// failStr reports a string assertion failure.
func failStr(t testing.TB, name, label, s, search, status string) {
	t.Helper()
	t.Errorf(
		"testastic: assertion failed\n\n  %s\n    string: %s\n    %s: %s (%s)",
		name, green(formatVal(s)), label, red(formatVal(search)), status,
	)
}

// Contains asserts that s contains substring.
func Contains(t testing.TB, s, substring string) {
	t.Helper()

	if !strings.Contains(s, substring) {
		failStr(t, "Contains", "substring", s, substring, "not found")
	}
}

// NotContains asserts that s does not contain substring.
func NotContains(t testing.TB, s, substring string) {
	t.Helper()

	if strings.Contains(s, substring) {
		failStr(t, "NotContains", "substring", s, substring, "found")
	}
}

// HasPrefix asserts that s has the given prefix.
func HasPrefix(t testing.TB, s, prefix string) {
	t.Helper()

	if !strings.HasPrefix(s, prefix) {
		failStr(t, "HasPrefix", "prefix", s, prefix, "not found")
	}
}

// HasSuffix asserts that s has the given suffix.
func HasSuffix(t testing.TB, s, suffix string) {
	t.Helper()

	if !strings.HasSuffix(s, suffix) {
		failStr(t, "HasSuffix", "suffix", s, suffix, "not found")
	}
}

// Matches asserts that s matches the given regular expression pattern.
func Matches(t testing.TB, s, pattern string) {
	t.Helper()

	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Errorf(
			"testastic: assertion failed\n\n  Matches\n    error: invalid pattern %q: %v",
			pattern, err,
		)

		return
	}

	if !re.MatchString(s) {
		failStr(t, "Matches", "pattern", s, pattern, "no match")
	}
}

// StringEmpty asserts that s is an empty string.
func StringEmpty(t testing.TB, s string) {
	t.Helper()

	if s != "" {
		fail(t, "StringEmpty", `""`, formatVal(s))
	}
}

// StringNotEmpty asserts that s is not an empty string.
func StringNotEmpty(t testing.TB, s string) {
	t.Helper()

	if s == "" {
		fail(t, "StringNotEmpty", "non-empty string", `""`)
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
