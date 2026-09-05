//nolint:testpackage // The private matching algorithm is the intended test surface.
package testastic

import (
	"encoding/json"
	"math"
	"testing"
)

func TestFindUnorderedMatchesFindsMaximumMatching(t *testing.T) {
	t.Parallel()

	edges := [][]bool{
		{true, true, false, false},
		{true, false, false, false},
		{false, true, true, false},
		{false, false, true, true},
	}
	values := []int{0, 1, 2, 3}

	result := findUnorderedMatches(values, values, func(expIdx int, actual int) bool {
		return edges[expIdx][actual]
	})

	if len(result.unmatchedExpected) != 0 || len(result.unusedActual) != 0 {
		t.Fatalf(
			"findUnorderedMatches() left expected %v and actual %v unmatched",
			result.unmatchedExpected,
			result.unusedActual,
		)
	}
}

//nolint:paralleltest // AllocsPerRun cannot run during a parallel test.
func TestFindUnorderedMatchesUsesBoundedAllocations(t *testing.T) {
	const size = 32

	values := make([]int, size)
	for i := range values {
		values[i] = i
	}

	allocations := testing.AllocsPerRun(5, func() {
		findUnorderedMatches(values, values, func(_ int, _ int) bool {
			return true
		})
	})

	if allocations > 10 {
		t.Fatalf("findUnorderedMatches() allocated %.0f times, want at most 10", allocations)
	}
}

func TestFindUnorderedMatchesReturnsMaximumCardinality(t *testing.T) {
	t.Parallel()

	const size = 4

	values := []int{0, 1, 2, 3}
	for graph := range uint(1 << (size * size)) {
		result := findUnorderedMatches(values, values, func(expIdx int, actual int) bool {
			return graph&(1<<uint(expIdx*size+actual)) != 0
		})
		matched := size - len(result.unmatchedExpected)
		want := maximumMatchingSize(graph, size, 0, 0)

		if matched != want {
			t.Fatalf("graph %016b matched %d elements, want %d", graph, matched, want)
		}
	}
}

func maximumMatchingSize(graph uint, size, expIdx int, usedActual uint) int {
	if expIdx == size {
		return 0
	}

	best := maximumMatchingSize(graph, size, expIdx+1, usedActual)

	for actual := range size {
		actualBit := uint(1) << uint(actual)

		edgeBit := uint(1) << uint(expIdx*size+actual)
		if usedActual&actualBit == 0 && graph&edgeBit != 0 {
			best = max(best, 1+maximumMatchingSize(graph, size, expIdx+1, usedActual|actualBit))
		}
	}

	return best
}

func BenchmarkFindUnorderedMatchesAmbiguous(b *testing.B) {
	const size = 2400

	values := make([]int, size)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		findUnorderedMatches(values, values, func(_ int, _ int) bool {
			return true
		})
	}
}

func TestUnorderedCandidatesPreserveComparisonSemantics(t *testing.T) {
	t.Parallel()

	values := []any{
		nil, true, false, "1", "value", 1, -1, uint64(math.MaxUint64),
		float64(1), float32(1.5), math.Inf(1), math.Inf(-1), math.NaN(),
		json.Number("1"), json.Number("1.0"), json.Number("1e0"),
		json.Number("0.5"), json.Number("1.50"), json.Number("15e-1"),
		json.Number("-0"), json.Number("0"), json.Number("9007199254740993"),
		json.Number("18446744073709551615"), json.Number("18446744073709551616"),
		json.Number("01"), json.Number("invalid"),
		anyStringMatcher{},
		map[string]any{"id": "a"},
		map[string]any{"id": "b"},
		[]any{1, "a"},
	}
	candidates := prepareUnorderedCandidates(values)

	for _, cfg := range []*config{{}, {IgnoredFields: []string{"$[0]"}}, {IgnoredFields: []string{"id"}}} {
		for expectedIndex, expected := range candidates {
			for actualIndex, actual := range candidates {
				want := len(compare(expected.value, actual.value, "$[0]", cfg)) == 0
				got := expected.matches(actual, "$[0]", cfg)

				if got != want {
					t.Fatalf("candidate %d versus %d: got %v, want %v", expectedIndex, actualIndex, got, want)
				}
			}
		}
	}
}
