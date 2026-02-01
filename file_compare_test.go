//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"testing"
)

func TestCompareFileLines(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		// given: identical lines
		expected := []string{"line 1", "line 2", "line 3"}
		actual := []string{"line 1", "line 2", "line 3"}

		// when: comparing
		diffs := compareFileLines(expected, actual)

		// then: no differences
		if len(diffs) != 0 {
			t.Errorf("expected no diffs, got %d: %v", len(diffs), diffs)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		// given: lines with a difference
		expected := []string{"line 1", "expected text", "line 3"}
		actual := []string{"line 1", "actual text", "line 3"}

		// when: comparing
		diffs := compareFileLines(expected, actual)

		// then: one difference at line 2
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		if diffs[0].Path != "line 2" {
			t.Errorf("expected path 'line 2', got %q", diffs[0].Path)
		}

		if diffs[0].Expected != "expected text" {
			t.Errorf("expected %q, got %q", "expected text", diffs[0].Expected)
		}

		if diffs[0].Actual != "actual text" {
			t.Errorf("expected actual %q, got %q", "actual text", diffs[0].Actual)
		}
	})

	t.Run("extra lines", func(t *testing.T) {
		// given: actual has more lines
		expected := []string{"line 1"}
		actual := []string{"line 1", "extra line"}

		// when: comparing
		diffs := compareFileLines(expected, actual)

		// then: one added difference
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		if diffs[0].Type != diffAdded {
			t.Errorf("expected diffAdded, got %v", diffs[0].Type)
		}

		if diffs[0].Actual != "extra line" {
			t.Errorf("expected actual 'extra line', got %q", diffs[0].Actual)
		}
	})

	t.Run("missing lines", func(t *testing.T) {
		// given: actual has fewer lines
		expected := []string{"line 1", "line 2"}
		actual := []string{"line 1"}

		// when: comparing
		diffs := compareFileLines(expected, actual)

		// then: one removed difference
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		if diffs[0].Type != diffRemoved {
			t.Errorf("expected diffRemoved, got %v", diffs[0].Type)
		}

		if diffs[0].Expected != "line 2" {
			t.Errorf("expected 'line 2', got %q", diffs[0].Expected)
		}
	})
}

func TestCompareFileLinesWithMatchers(t *testing.T) {
	t.Run("match", func(t *testing.T) {
		// given: expected with matcher, matching actual
		expected := []string{"Name: {{anyString}}"}
		actual := []string{"Name: Alice"}

		// when: comparing with matchers
		diffs := compareFileLinesWithMatchers(expected, actual)

		// then: no differences (matcher matches)
		if len(diffs) != 0 {
			t.Errorf("expected no diffs, got %d: %v", len(diffs), diffs)
		}
	})

	t.Run("no match", func(t *testing.T) {
		// given: expected with int matcher, non-matching actual
		expected := []string{"Age: {{anyInt}}"}
		actual := []string{"Age: not-a-number"}

		// when: comparing with matchers
		diffs := compareFileLinesWithMatchers(expected, actual)

		// then: one difference (matcher failed)
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		if diffs[0].Type != diffMatcherFailed {
			t.Errorf("expected diffMatcherFailed, got %v", diffs[0].Type)
		}
	})
}
