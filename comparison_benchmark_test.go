//nolint:testpackage // Benchmarks exercise the private comparison and diagnostic interface without parsing or file I/O.
package testastic

import (
	"encoding/json"
	"strconv"
	"testing"
)

func BenchmarkStructuredComparison(b *testing.B) {
	for _, format := range []string{"JSON", "YAML"} {
		b.Run(format, func(b *testing.B) {
			for _, kind := range []string{"plain", "matcher", "unordered"} {
				b.Run(kind, func(b *testing.B) {
					for _, outcome := range []string{"pass", "fail"} {
						b.Run(outcome, func(b *testing.B) {
							benchmarkStructuredComparison(b, format, kind, outcome == "fail")
						})
					}
				})
			}
		})
	}
}

func benchmarkStructuredComparison(b *testing.B, format, kind string, failing bool) {
	b.Helper()

	const size = 64

	expectedItems := make([]any, size)
	actualItems := make([]any, size)

	for index := range expectedItems {
		expectedItems[index] = "value"
		actualItems[index] = "value"
	}

	if kind == "matcher" {
		expectedItems[0] = anyStringMatcher{}
	}

	expected := map[string]any{"items": expectedItems, "status": "expected"}
	actual := map[string]any{"items": actualItems, "status": "expected"}

	if failing {
		actual["status"] = "changed"
	}

	cfg := &config{IgnoreArrayOrder: kind == "unordered"}

	if format == "JSON" {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			comparison := compareTree(expected, actual, "$", cfg)
			if len(comparison.differences) > 0 {
				formatJSONDiffInline(comparison.diagnostic())
			}
		}

		return
	}

	expectedDocuments, actualDocuments := benchmarkYAMLDocuments(b, expected, actual)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		comparison := compareYAMLDocuments(expectedDocuments, actualDocuments, cfg)
		if len(comparison.differences) > 0 {
			formatYAMLDiffInline(comparison.diagnostic())
		}
	}
}

func benchmarkYAMLDocuments(b *testing.B, expected, actual any) (yamlDocuments, yamlDocuments) {
	b.Helper()

	expectedText, err := renderYAMLDocuments(yamlDocuments{cleanMatchersForDisplay(expected)})
	if err != nil {
		b.Fatal(err)
	}

	actualText, err := renderYAMLDocuments(yamlDocuments{actual})
	if err != nil {
		b.Fatal(err)
	}

	parsedExpected, err := parseExpectedYAMLString(expectedText)
	if err != nil {
		b.Fatal(err)
	}

	parsedActual, err := parseActualYAML([]byte(actualText))
	if err != nil {
		b.Fatal(err)
	}

	return parsedExpected.Documents, parsedActual
}

func BenchmarkComparisonScaling(b *testing.B) {
	for _, kind := range []string{"strings", "numbers", "objects"} {
		b.Run(kind, func(b *testing.B) {
			for _, unordered := range []bool{false, true} {
				b.Run("unordered="+strconv.FormatBool(unordered), func(b *testing.B) {
					for _, size := range []int{16, 64, 256} {
						b.Run(strconv.Itoa(size), func(b *testing.B) {
							benchmarkComparisonScaling(b, kind, unordered, size)
						})
					}
				})
			}
		})
	}
}

func benchmarkComparisonScaling(b *testing.B, kind string, unordered bool, size int) {
	b.Helper()

	expected := make([]any, size)
	actual := make([]any, size)

	for index := range expected {
		value := strconv.Itoa(index)

		switch kind {
		case "strings":
			expected[index] = value
			actual[index] = value
		case "numbers":
			expected[index] = json.Number(value)
			actual[index] = json.Number(value)
		case "objects":
			expected[index] = map[string]any{"id": value, "active": true}
			actual[index] = map[string]any{"id": value, "active": true}
		}
	}

	if unordered {
		for left, right := 0, len(actual)-1; left < right; left, right = left+1, right-1 {
			actual[left], actual[right] = actual[right], actual[left]
		}
	}

	cfg := &config{IgnoreArrayOrder: unordered}
	comparison := compareTree(expected, actual, "$", cfg)

	if len(comparison.differences) != 0 {
		b.Fatal("expected equal values")
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		comparison = compareTree(expected, actual, "$", cfg)
		if len(comparison.differences) != 0 {
			b.Fatal("expected equal values")
		}
	}
}
