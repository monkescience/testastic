package testastic

import (
	"testing"
)

func TestCompareFileLines_ExactMatch(t *testing.T) {
	// given: identical lines
	expected := []string{"line 1", "line 2", "line 3"}
	actual := []string{"line 1", "line 2", "line 3"}

	// when: comparing
	diffs := compareFileLines(expected, actual)

	// then: no differences
	if len(diffs) != 0 {
		t.Errorf("expected no diffs, got %d: %v", len(diffs), diffs)
	}
}
