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

const nilTypeName = "nil"

// fail reports an assertion failure with expected and actual values.
func fail(tb testing.TB, name, expected, actual string) {
	tb.Helper()
	tb.Errorf(
		"testastic: assertion failed\n\n  %s\n    expected: %s\n    actual:   %s",
		name, red(expected), green(actual),
	)
}

// Equal asserts that expected and actual are equal using the == operator.
// Reports a non-fatal error on failure, allowing the test to continue.
func Equal[T comparable](tb testing.TB, expected, actual T) {
	tb.Helper()

	if expected != actual {
		fail(tb, "Equal", formatVal(expected), formatVal(actual))
	}
}

// NotEqual asserts that expected and actual are not equal using the == operator.
// Reports a non-fatal error on failure, allowing the test to continue.
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

// Nil asserts that value is nil. Handles interface nil correctly by
// checking the underlying pointer for pointer, map, slice, channel, and func types.
func Nil(tb testing.TB, value any) {
	tb.Helper()

	if !isNil(value) {
		fail(tb, "Nil", "nil", formatVal(value))
	}
}

// NotNil asserts that value is not nil. Handles interface nil correctly by
// checking the underlying pointer for pointer, map, slice, channel, and func types.
func NotNil(tb testing.TB, value any) {
	tb.Helper()

	if isNil(value) {
		fail(tb, "NotNil", "not nil", "nil")
	}
}

// True asserts that value is true.
// Reports a non-fatal error on failure, allowing the test to continue.
func True(tb testing.TB, value bool) {
	tb.Helper()

	if !value {
		fail(tb, "True", "true", "false")
	}
}

// False asserts that value is false.
// Reports a non-fatal error on failure, allowing the test to continue.
func False(tb testing.TB, value bool) {
	tb.Helper()

	if value {
		fail(tb, "False", "false", "true")
	}
}

// NoError asserts that err is nil.
// Reports a non-fatal error on failure, displaying the error message.
func NoError(tb testing.TB, err error) {
	tb.Helper()

	if err != nil {
		fail(tb, "NoError", "no error", err.Error())
	}
}

// Error asserts that err is not nil.
// Reports a non-fatal error on failure, allowing the test to continue.
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
		errStr := nilTypeName
		if err != nil {
			errStr = err.Error()
		}

		targetStr := nilTypeName
		if target != nil {
			targetStr = target.Error()
		}

		fail(tb, "ErrorIs", targetStr, errStr)
	}
}

// ErrorContains asserts that err is non-nil and its message contains the given substring.
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

// ErrorAs asserts that err matches the target type using errors.As.
// The target must be a non-nil pointer to either an error interface or a concrete type
// that implements error.
//
//	var pathErr *os.PathError
//	testastic.ErrorAs(t, err, &pathErr)
func ErrorAs(tb testing.TB, err error, target any) {
	tb.Helper()

	if !errors.As(err, target) {
		errStr := nilTypeName
		if err != nil {
			errStr = err.Error()
		}

		fail(tb, "ErrorAs", fmt.Sprintf("error assignable to %T", target), errStr)
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

// Greater asserts that a > b using [cmp.Ordered] comparison.
func Greater[T cmp.Ordered](tb testing.TB, a, b T) {
	tb.Helper()

	if a <= b {
		failCmp(tb, "Greater", ">", "<=", formatVal(a), formatVal(b))
	}
}

// GreaterOrEqual asserts that a >= b using [cmp.Ordered] comparison.
func GreaterOrEqual[T cmp.Ordered](tb testing.TB, a, b T) {
	tb.Helper()

	if a < b {
		failCmp(tb, "GreaterOrEqual", ">=", "<", formatVal(a), formatVal(b))
	}
}

// Less asserts that a < b using [cmp.Ordered] comparison.
func Less[T cmp.Ordered](tb testing.TB, a, b T) {
	tb.Helper()

	if a >= b {
		failCmp(tb, "Less", "<", ">=", formatVal(a), formatVal(b))
	}
}

// LessOrEqual asserts that a <= b using [cmp.Ordered] comparison.
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

// stringInputValue converts supported string assertion inputs to a string.
func stringInputValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	case fmt.Stringer:
		return v.String(), true
	default:
		return "", false
	}
}

// failStrType reports an unsupported input type for string assertions.
func failStrType(tb testing.TB, name string, value any) {
	tb.Helper()
	tb.Errorf(
		"testastic: assertion failed\n\n  %s\n    error: unsupported type %T (want string, []byte, or fmt.Stringer)",
		name, value,
	)
}

// failStr reports a string assertion failure.
func failStr(tb testing.TB, name, label, s, search, status string) {
	tb.Helper()
	tb.Errorf(
		"testastic: assertion failed\n\n  %s\n    string: %s\n    %s: %s (%s)",
		name, green(formatVal(s)), label, red(formatVal(search)), status,
	)
}

// Contains asserts that a string, byte slice, or Stringer contains the given substring.
// Reports a non-fatal error on failure, displaying the string and missing substring.
func Contains(tb testing.TB, value any, substring string) {
	tb.Helper()

	s, ok := stringInputValue(value)
	if !ok {
		failStrType(tb, "Contains", value)

		return
	}

	if !strings.Contains(s, substring) {
		failStr(tb, "Contains", "substring", s, substring, "not found")
	}
}

// NotContains asserts that a string, byte slice, or Stringer does not contain the given substring.
func NotContains(tb testing.TB, value any, substring string) {
	tb.Helper()

	s, ok := stringInputValue(value)
	if !ok {
		failStrType(tb, "NotContains", value)

		return
	}

	if strings.Contains(s, substring) {
		failStr(tb, "NotContains", "substring", s, substring, "found")
	}
}

// HasPrefix asserts that a string, byte slice, or Stringer starts with the given prefix.
func HasPrefix(tb testing.TB, value any, prefix string) {
	tb.Helper()

	s, ok := stringInputValue(value)
	if !ok {
		failStrType(tb, "HasPrefix", value)

		return
	}

	if !strings.HasPrefix(s, prefix) {
		failStr(tb, "HasPrefix", "prefix", s, prefix, "not found")
	}
}

// HasSuffix asserts that a string, byte slice, or Stringer ends with the given suffix.
func HasSuffix(tb testing.TB, value any, suffix string) {
	tb.Helper()

	s, ok := stringInputValue(value)
	if !ok {
		failStrType(tb, "HasSuffix", value)

		return
	}

	if !strings.HasSuffix(s, suffix) {
		failStr(tb, "HasSuffix", "suffix", s, suffix, "not found")
	}
}

// Matches asserts that a string, byte slice, or Stringer matches the given regular expression pattern.
func Matches(tb testing.TB, value any, pattern string) {
	tb.Helper()

	s, ok := stringInputValue(value)
	if !ok {
		failStrType(tb, "Matches", value)

		return
	}

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

// StringEmpty asserts that s is an empty string ("").
func StringEmpty(tb testing.TB, s string) {
	tb.Helper()

	if s != "" {
		fail(tb, "StringEmpty", `""`, formatVal(s))
	}
}

// StringNotEmpty asserts that s is not an empty string ("").
func StringNotEmpty(tb testing.TB, s string) {
	tb.Helper()

	if s == "" {
		fail(tb, "StringNotEmpty", "non-empty string", `""`)
	}
}

// Panics asserts that the function panics when called.
//
//	testastic.Panics(t, func() {
//	    panic("something went wrong")
//	})
func Panics(tb testing.TB, fn func()) {
	tb.Helper()

	if !didPanic(fn) {
		tb.Errorf("testastic: assertion failed\n\n  Panics\n    expected: %s\n    actual:   %s",
			red("function to panic"), green("no panic"))
	}
}

// NotPanics asserts that the function does not panic when called.
//
//	testastic.NotPanics(t, func() {
//	    doSomethingSafe()
//	})
func NotPanics(tb testing.TB, fn func()) {
	tb.Helper()

	if didPanic(fn) {
		tb.Errorf("testastic: assertion failed\n\n  NotPanics\n    expected: %s\n    actual:   %s",
			red("no panic"), green("function panicked"))
	}
}

// didPanic returns true if the function panics when called.
func didPanic(fn func()) bool {
	panicked := true

	func() {
		defer func() {
			_ = recover()
		}()

		fn()

		panicked = false
	}()

	return panicked
}

// isNil checks if a value is nil, handling interface nil correctly.
func isNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	//nolint:exhaustive // Only nil-able types need checking.
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
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
