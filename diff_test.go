//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"strconv"
	"testing"
)

func TestComputeDiffFallsBackWhenMatrixExceedsLimit(t *testing.T) {
	t.Parallel()

	// given: line sets whose LCS matrix would exceed the cell limit.
	const lineCount = 1024

	expected := make([]string, lineCount)
	actual := make([]string, lineCount)
	expected[0] = "prefix"
	actual[0] = "prefix"
	expected[lineCount-1] = "suffix"
	actual[lineCount-1] = "suffix"
	actual[1] = "inserted"

	for i := 1; i < lineCount-1; i++ {
		expected[i] = "line-" + strconv.Itoa(i)
	}

	copy(actual[2:lineCount-1], expected[1:lineCount-2])

	want := make([]string, 0, 2*lineCount-2)

	want = append(want, "  prefix")
	for _, line := range expected[1 : lineCount-1] {
		want = append(want, red("- "+line))
	}

	for _, line := range actual[1 : lineCount-1] {
		want = append(want, green("+ "+line))
	}

	want = append(want, "  suffix")

	// when: computing the diff.
	got := computeDiff(expected, actual)

	// then: the fallback preserves the common ends and replaces the middle.
	if len(got) != len(want) {
		t.Fatalf("unexpected diff length: got %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected diff line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestComputeDiffFallsBackWhenDimensionExceedsLimit(t *testing.T) {
	t.Parallel()

	// given: a skewed line set whose larger dimension exceeds the line limit.
	const lineCount = 4097

	expected := make([]string, lineCount)
	expected[0] = "prefix"
	expected[lineCount-1] = "suffix"

	for i := 1; i < lineCount-1; i++ {
		expected[i] = "line-" + strconv.Itoa(i)
	}

	actual := []string{"prefix", expected[lineCount/2], "suffix"}
	want := make([]string, 0, lineCount+1)

	want = append(want, "  prefix")
	for _, line := range expected[1 : lineCount-1] {
		want = append(want, red("- "+line))
	}

	want = append(want, green("+ "+actual[1]), "  suffix")

	// when: computing the diff.
	got := computeDiff(expected, actual)

	// then: the fallback avoids the skewed matrix and preserves every changed line.
	if len(got) != len(want) {
		t.Fatalf("unexpected diff length: got %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected diff line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
