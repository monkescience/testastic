package testastic

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

// Len asserts that the collection has the expected length.
// Works with slices, maps, strings, arrays, and channels.
func Len(t testing.TB, collection any, expected int) {
	t.Helper()

	actual := getLen(collection)
	if actual == -1 {
		t.Errorf(
			"testastic: assertion failed\n\n  Len\n    error: cannot get length of %T",
			collection,
		)

		return
	}

	if actual != expected {
		t.Errorf(
			"testastic: assertion failed\n\n  Len\n    expected: %s\n    actual:   %s",
			red(strconv.Itoa(expected)), green(strconv.Itoa(actual)),
		)
	}
}

// Empty asserts that the collection is empty.
// Works with slices, maps, strings, arrays, and channels.
func Empty(t testing.TB, collection any) {
	t.Helper()

	length := getLen(collection)
	if length == -1 {
		t.Errorf(
			"testastic: assertion failed\n\n  Empty\n    error: cannot get length of %T",
			collection,
		)

		return
	}

	if length != 0 {
		t.Errorf(
			"testastic: assertion failed\n\n  Empty\n    expected: %s\n    actual:   %s",
			red("empty (length 0)"), green(fmt.Sprintf("length %d", length)),
		)
	}
}

// NotEmpty asserts that the collection is not empty.
// Works with slices, maps, strings, arrays, and channels.
func NotEmpty(t testing.TB, collection any) {
	t.Helper()

	length := getLen(collection)
	if length == -1 {
		t.Errorf(
			"testastic: assertion failed\n\n  NotEmpty\n    error: cannot get length of %T",
			collection,
		)

		return
	}

	if length == 0 {
		t.Errorf(
			"testastic: assertion failed\n\n  NotEmpty\n    expected: %s\n    actual:   %s",
			red("non-empty"), green("empty (length 0)"),
		)
	}
}

// SliceContains asserts that slice contains element.
func SliceContains[T comparable](t testing.TB, slice []T, element T) {
	t.Helper()

	for _, v := range slice {
		if v == element {
			return
		}
	}

	t.Errorf(
		"testastic: assertion failed\n\n  SliceContains\n    slice:   %s\n    element: %s (not found)",
		green(formatSlice(slice)), red(formatVal(element)),
	)
}

// SliceNotContains asserts that slice does not contain element.
func SliceNotContains[T comparable](t testing.TB, slice []T, element T) {
	t.Helper()

	for _, v := range slice {
		if v == element {
			t.Errorf(
				"testastic: assertion failed\n\n  SliceNotContains\n    slice:   %s\n    element: %s (found)",
				green(formatSlice(slice)), red(formatVal(element)),
			)

			return
		}
	}
}

// SliceEqual asserts that two slices are equal (same length and elements in same order).
func SliceEqual[T comparable](t testing.TB, expected, actual []T) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf(
			"testastic: assertion failed\n\n  SliceEqual\n    expected: %s (len %d)\n    actual:   %s (len %d)",
			red(formatSlice(expected)), len(expected), green(formatSlice(actual)), len(actual),
		)

		return
	}

	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf(
				"testastic: assertion failed\n\n  SliceEqual\n    diff at [%d]: %s != %s",
				i, red(formatVal(expected[i])), green(formatVal(actual[i])),
			)

			return
		}
	}
}

// MapHasKey asserts that the map contains the given key.
func MapHasKey[K comparable, V any](t testing.TB, m map[K]V, key K) {
	t.Helper()

	if _, ok := m[key]; !ok {
		t.Errorf(
			"testastic: assertion failed\n\n  MapHasKey\n    map: %s\n    key: %s (not found)",
			green(formatMap(m)), red(formatVal(key)),
		)
	}
}

// MapNotHasKey asserts that the map does not contain the given key.
func MapNotHasKey[K comparable, V any](t testing.TB, m map[K]V, key K) {
	t.Helper()

	if _, ok := m[key]; ok {
		t.Errorf(
			"testastic: assertion failed\n\n  MapNotHasKey\n    map: %s\n    key: %s (found)",
			green(formatMap(m)), red(formatVal(key)),
		)
	}
}

// MapEqual asserts that two maps are equal.
func MapEqual[K comparable, V comparable](t testing.TB, expected, actual map[K]V) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf(
			"testastic: assertion failed\n\n  MapEqual\n    expected: %s (len %d)\n    actual:   %s (len %d)",
			red(formatMap(expected)), len(expected), green(formatMap(actual)), len(actual),
		)

		return
	}

	for k, ev := range expected {
		av, ok := actual[k]
		if !ok {
			t.Errorf(
				"testastic: assertion failed\n\n  MapEqual\n    missing key: %s",
				red(formatVal(k)),
			)

			return
		}

		if ev != av {
			t.Errorf(
				"testastic: assertion failed\n\n  MapEqual\n    diff at key %s: %s != %s",
				formatVal(k), red(formatVal(ev)), green(formatVal(av)),
			)

			return
		}
	}
}

// getLen returns the length of a collection, or -1 if not a collection type.
func getLen(collection any) int {
	if collection == nil {
		return 0
	}

	v := reflect.ValueOf(collection)
	switch v.Kind() { //nolint:exhaustive // Only collection types have length.
	case reflect.Slice, reflect.Map, reflect.String, reflect.Array, reflect.Chan:
		return v.Len()
	}

	return -1
}

// formatSlice formats a slice for display, truncating if too long.
func formatSlice[T any](s []T) string {
	if len(s) <= 5 {
		return fmt.Sprintf("%v", s)
	}

	return fmt.Sprintf("[%v %v %v ... (%d total)]", s[0], s[1], s[2], len(s))
}

// formatMap formats a map for display, truncating if too many entries.
func formatMap[K comparable, V any](m map[K]V) string {
	if len(m) <= 3 {
		return fmt.Sprintf("%v", m)
	}

	return fmt.Sprintf("map[...] (%d entries)", len(m))
}
