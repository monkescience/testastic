//nolint:testpackage // Benchmarks exercise the private comparison and diagnostic interface without parsing or file I/O.
package testastic

import "testing"

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
