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

// Equal asserts that expected and actual are equal.
func Equal[T comparable](t testing.TB, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Errorf("testastic: assertion failed\n\n  Equal\n    expected: %s\n    actual:   %s", red(formatVal(expected)), green(formatVal(actual)))
	}
}

// NotEqual asserts that expected and actual are not equal.
func NotEqual[T comparable](t testing.TB, unexpected, actual T) {
	t.Helper()
	if unexpected == actual {
		t.Errorf("testastic: assertion failed\n\n  NotEqual\n    unexpected: %s\n    actual:     %s", red(formatVal(unexpected)), green(formatVal(actual)))
	}
}

// DeepEqual asserts that expected and actual are deeply equal using reflect.DeepEqual.
func DeepEqual[T any](t testing.TB, expected, actual T) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("testastic: assertion failed\n\n  DeepEqual\n    expected: %s\n    actual:   %s", red(formatVal(expected)), green(formatVal(actual)))
	}
}

// Nil asserts that value is nil.
func Nil(t testing.TB, value any) {
	t.Helper()
	if !isNil(value) {
		t.Errorf("testastic: assertion failed\n\n  Nil\n    expected: %s\n    actual:   %s", red("nil"), green(formatVal(value)))
	}
}

// NotNil asserts that value is not nil.
func NotNil(t testing.TB, value any) {
	t.Helper()
	if isNil(value) {
		t.Errorf("testastic: assertion failed\n\n  NotNil\n    expected: %s\n    actual:   %s", red("not nil"), green("nil"))
	}
}

// True asserts that value is true.
func True(t testing.TB, value bool) {
	t.Helper()
	if !value {
		t.Errorf("testastic: assertion failed\n\n  True\n    expected: %s\n    actual:   %s", red("true"), green("false"))
	}
}

// False asserts that value is false.
func False(t testing.TB, value bool) {
	t.Helper()
	if value {
		t.Errorf("testastic: assertion failed\n\n  False\n    expected: %s\n    actual:   %s", red("false"), green("true"))
	}
}

// NoError asserts that err is nil.
func NoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("testastic: assertion failed\n\n  NoError\n    expected: %s\n    actual:   %s", red("no error"), green(err.Error()))
	}
}

// Error asserts that err is not nil.
func Error(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("testastic: assertion failed\n\n  Error\n    expected: %s\n    actual:   %s", red("an error"), green("nil"))
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
		t.Errorf("testastic: assertion failed\n\n  ErrorIs\n    expected: %s\n    actual:   %s", red(target.Error()), green(errStr))
	}
}

// ErrorContains asserts that err contains the given substring.
func ErrorContains(t testing.TB, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Errorf("testastic: assertion failed\n\n  ErrorContains\n    expected: %s\n    actual:   %s", red("error containing "+fmt.Sprintf("%q", substring)), green("nil"))
		return
	}
	if !strings.Contains(err.Error(), substring) {
		t.Errorf("testastic: assertion failed\n\n  ErrorContains\n    expected: %s\n    actual:   %s", red("error containing "+fmt.Sprintf("%q", substring)), green(err.Error()))
	}
}

// Greater asserts that a > b.
func Greater[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()
	if !(a > b) {
		t.Errorf("testastic: assertion failed\n\n  Greater\n    expected: %s > %s\n    actual:   %s <= %s", red(formatVal(a)), red(formatVal(b)), green(formatVal(a)), green(formatVal(b)))
	}
}

// GreaterOrEqual asserts that a >= b.
func GreaterOrEqual[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()
	if !(a >= b) {
		t.Errorf("testastic: assertion failed\n\n  GreaterOrEqual\n    expected: %s >= %s\n    actual:   %s < %s", red(formatVal(a)), red(formatVal(b)), green(formatVal(a)), green(formatVal(b)))
	}
}

// Less asserts that a < b.
func Less[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()
	if !(a < b) {
		t.Errorf("testastic: assertion failed\n\n  Less\n    expected: %s < %s\n    actual:   %s >= %s", red(formatVal(a)), red(formatVal(b)), green(formatVal(a)), green(formatVal(b)))
	}
}

// LessOrEqual asserts that a <= b.
func LessOrEqual[T cmp.Ordered](t testing.TB, a, b T) {
	t.Helper()
	if !(a <= b) {
		t.Errorf("testastic: assertion failed\n\n  LessOrEqual\n    expected: %s <= %s\n    actual:   %s > %s", red(formatVal(a)), red(formatVal(b)), green(formatVal(a)), green(formatVal(b)))
	}
}

// Between asserts that min <= value <= max.
func Between[T cmp.Ordered](t testing.TB, value, min, max T) {
	t.Helper()
	if value < min || value > max {
		t.Errorf("testastic: assertion failed\n\n  Between\n    expected: %s\n    actual:   %s", red(formatVal(min)+" <= value <= "+formatVal(max)), green(formatVal(value)))
	}
}

// Contains asserts that s contains substring.
func Contains(t testing.TB, s, substring string) {
	t.Helper()
	if !strings.Contains(s, substring) {
		t.Errorf("testastic: assertion failed\n\n  Contains\n    string:    %s\n    substring: %s (not found)", green(formatVal(s)), red(formatVal(substring)))
	}
}

// NotContains asserts that s does not contain substring.
func NotContains(t testing.TB, s, substring string) {
	t.Helper()
	if strings.Contains(s, substring) {
		t.Errorf("testastic: assertion failed\n\n  NotContains\n    string:    %s\n    substring: %s (found)", green(formatVal(s)), red(formatVal(substring)))
	}
}

// HasPrefix asserts that s has the given prefix.
func HasPrefix(t testing.TB, s, prefix string) {
	t.Helper()
	if !strings.HasPrefix(s, prefix) {
		t.Errorf("testastic: assertion failed\n\n  HasPrefix\n    string: %s\n    prefix: %s (not found)", green(formatVal(s)), red(formatVal(prefix)))
	}
}

// HasSuffix asserts that s has the given suffix.
func HasSuffix(t testing.TB, s, suffix string) {
	t.Helper()
	if !strings.HasSuffix(s, suffix) {
		t.Errorf("testastic: assertion failed\n\n  HasSuffix\n    string: %s\n    suffix: %s (not found)", green(formatVal(s)), red(formatVal(suffix)))
	}
}

// Matches asserts that s matches the given regular expression pattern.
func Matches(t testing.TB, s, pattern string) {
	t.Helper()
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Errorf("testastic: assertion failed\n\n  Matches\n    error: invalid pattern %q: %v", pattern, err)
		return
	}
	if !re.MatchString(s) {
		t.Errorf("testastic: assertion failed\n\n  Matches\n    string:  %s\n    pattern: %s (no match)", green(formatVal(s)), red(formatVal(pattern)))
	}
}

// StringEmpty asserts that s is an empty string.
func StringEmpty(t testing.TB, s string) {
	t.Helper()
	if s != "" {
		t.Errorf("testastic: assertion failed\n\n  StringEmpty\n    expected: %s\n    actual:   %s", red(`""`), green(formatVal(s)))
	}
}

// StringNotEmpty asserts that s is not an empty string.
func StringNotEmpty(t testing.TB, s string) {
	t.Helper()
	if s == "" {
		t.Errorf("testastic: assertion failed\n\n  StringNotEmpty\n    expected: %s\n    actual:   %s", red("non-empty string"), green(`""`))
	}
}

// isNil checks if a value is nil, handling interface nil correctly.
func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
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
