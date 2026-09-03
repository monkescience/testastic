//nolint:testpackage // The private matching algorithm is the intended test surface.
package testastic

import "testing"

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
