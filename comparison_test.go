package testastic_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/monkescience/testastic"
)

type comparisonStateMatcher struct {
	accepted   bool
	calls      *atomic.Int32
	expression string
}

func (m comparisonStateMatcher) Match(any) bool {
	return (m.calls.Add(1) == 1) == m.accepted
}

func (m comparisonStateMatcher) String() string {
	return m.expression
}

type comparisonValueMatcher struct {
	calls      *atomic.Int32
	expression string
	want       string
}

func (m comparisonValueMatcher) Match(actual any) bool {
	m.calls.Add(1)

	value, ok := actual.(string)

	return ok && value == m.want
}

func (m comparisonValueMatcher) String() string {
	return m.expression
}

func TestAssertJSON_DiagnosticReusesDirectMatcherVerdict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		matcher    string
		accepted   bool
		wantInDiff bool
	}{
		{name: "accepted", matcher: "comparisonJSONAccept", accepted: true},
		{name: "rejected", matcher: "comparisonJSONReject", wantInDiff: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// given: a matcher whose result changes after its first evaluation
			calls := &atomic.Int32{}
			expression := "{{" + test.matcher + "}}"
			testastic.RegisterMatcher(test.matcher, func(string) (testastic.Matcher, error) {
				return comparisonStateMatcher{
					accepted:   test.accepted,
					calls:      calls,
					expression: expression,
				}, nil
			})

			expectedFile := writeComparisonExpected(t, "expected.json",
				`{"value":"`+expression+`","status":"expected"}`)
			mt := &mockT{}

			// when: another field forces failure diagnostics
			testastic.AssertJSON(mt, expectedFile, `{"value":"actual","status":"changed"}`)

			// then: diagnostics retain the first verdict without evaluating the matcher again
			if !mt.failed || mt.fatal {
				t.Fatalf("expected a nonfatal mismatch, got: %s", mt.message)
			}

			if got := calls.Load(); got != 1 {
				t.Errorf("Matcher.Match calls = %d, want 1", got)
			}

			if got := strings.Contains(mt.message, expression); got != test.wantInDiff {
				t.Errorf("diagnostic contains matcher expression = %t, want %t: %s",
					got, test.wantInDiff, mt.message)
			}
		})
	}
}

func TestAssertYAML_DiagnosticReusesDirectMatcherVerdict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		matcher    string
		accepted   bool
		wantInDiff bool
	}{
		{name: "accepted", matcher: "comparisonYAMLAccept", accepted: true},
		{name: "rejected", matcher: "comparisonYAMLReject", wantInDiff: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// given: a matcher whose result changes after its first evaluation
			calls := &atomic.Int32{}
			expression := "{{" + test.matcher + "}}"
			testastic.RegisterMatcher(test.matcher, func(string) (testastic.Matcher, error) {
				return comparisonStateMatcher{
					accepted:   test.accepted,
					calls:      calls,
					expression: expression,
				}, nil
			})

			expectedFile := writeComparisonExpected(t, "expected.yaml",
				"value: "+expression+"\nstatus: expected\n")
			mt := &mockT{}

			// when: another field forces failure diagnostics
			testastic.AssertYAML(mt, expectedFile, "value: actual\nstatus: changed\n")

			// then: diagnostics retain the first verdict without evaluating the matcher again
			if !mt.failed || mt.fatal {
				t.Fatalf("expected a nonfatal mismatch, got: %s", mt.message)
			}

			if got := calls.Load(); got != 1 {
				t.Errorf("Matcher.Match calls = %d, want 1", got)
			}

			if got := strings.Contains(mt.message, expression); got != test.wantInDiff {
				t.Errorf("diagnostic contains matcher expression = %t, want %t: %s",
					got, test.wantInDiff, mt.message)
			}
		})
	}
}

func TestAssertJSON_DiagnosticReusesUnorderedAlignment(t *testing.T) {
	t.Parallel()

	// given: a passing unordered comparison establishes the matcher evaluation count

	calls := &atomic.Int32{}

	const (
		matcher    = "comparisonJSONValue"
		expression = "{{comparisonJSONValue}}"
	)

	testastic.RegisterMatcher(matcher, func(string) (testastic.Matcher, error) {
		return comparisonValueMatcher{calls: calls, expression: expression, want: "z"}, nil
	})

	expectedFile := writeComparisonExpected(t, "expected.json",
		`{"items":["a","`+expression+`"],"status":"expected"}`)
	passing := &mockT{}
	testastic.AssertJSON(passing, expectedFile, `{"items":["z","a"],"status":"expected"}`,
		testastic.IgnoreArrayOrderAt("$.items"))

	if passing.failed {
		t.Fatalf("expected the reference comparison to pass, got: %s", passing.message)
	}

	wantCalls := calls.Load()
	calls.Store(0)

	mt := &mockT{}

	// when: another field changes and forces diagnostics for the same array
	testastic.AssertJSON(mt, expectedFile, `{"items":["z","a"],"status":"changed"}`,
		testastic.IgnoreArrayOrderAt("$.items"))

	// then: the diagnostic reuses the reference alignment and adds no matcher evaluations
	if !mt.failed || mt.fatal {
		t.Fatalf("expected a nonfatal mismatch, got: %s", mt.message)
	}

	if got := calls.Load(); got != wantCalls {
		t.Errorf("Matcher.Match calls = %d, want reference count %d", got, wantCalls)
	}

	if strings.Contains(mt.message, expression) ||
		strings.Contains(mt.message, `-     "a"`) ||
		strings.Contains(mt.message, `+     "z"`) {
		t.Errorf("diagnostic changed the matched unordered array: %s", mt.message)
	}
}

func TestAssertYAML_StreamDiagnosticReusesExpectedIndexAlignment(t *testing.T) {
	t.Parallel()

	// given: an unordered array in the second document uses document-relative options

	calls := &atomic.Int32{}

	const (
		matcher    = "comparisonYAMLStreamValue"
		expression = "{{comparisonYAMLStreamValue}}"
	)

	testastic.RegisterMatcher(matcher, func(string) (testastic.Matcher, error) {
		return comparisonValueMatcher{calls: calls, expression: expression, want: "z"}, nil
	})

	expectedFile := writeComparisonExpected(t, "expected.yaml",
		"name: first\n---\nitems: [a, "+expression+"]\nstatus: expected\n")
	passing := &mockT{}
	testastic.AssertYAML(passing, expectedFile,
		"name: first\n---\nitems: [z, a]\nstatus: expected\n",
		testastic.IgnoreArrayOrderAt("$.items"))

	if passing.failed {
		t.Fatalf("expected the reference comparison to pass, got: %s", passing.message)
	}

	wantCalls := calls.Load()
	calls.Store(0)

	mt := &mockT{}

	// when: another field changes and forces diagnostics for the same array
	testastic.AssertYAML(mt, expectedFile,
		"name: first\n---\nitems: [z, a]\nstatus: changed\n",
		testastic.IgnoreArrayOrderAt("$.items"))

	// then: the diagnostic reuses the reference alignment and adds no matcher evaluations
	if !mt.failed || mt.fatal {
		t.Fatalf("expected a nonfatal mismatch, got: %s", mt.message)
	}

	if got := calls.Load(); got != wantCalls {
		t.Errorf("Matcher.Match calls = %d, want reference count %d", got, wantCalls)
	}

	if strings.Contains(mt.message, expression) ||
		strings.Contains(mt.message, "- - a") || strings.Contains(mt.message, "+ - z") {
		t.Errorf("diagnostic changed the matched unordered stream value: %s", mt.message)
	}
}

func TestAssertJSON_UpdateUnequalUnorderedArrayDoesNotMatchCandidates(t *testing.T) {
	t.Parallel()

	// given: an expected matcher occupies the first position of an unordered array

	calls := &atomic.Int32{}

	const (
		matcher    = "comparisonJSONUpdateValue"
		expression = "{{comparisonJSONUpdateValue}}"
	)

	testastic.RegisterMatcher(matcher, func(string) (testastic.Matcher, error) {
		return comparisonValueMatcher{calls: calls, expression: expression, want: "kept"}, nil
	})

	expectedFile := writeComparisonExpected(t, "expected.json", `{"items":["`+expression+`"]}`)
	mt := &mockT{}

	// when: update mode receives an additional actual array element
	testastic.AssertJSON(mt, expectedFile, `{"items":["kept","added"]}`,
		testastic.IgnoreArrayOrderAt("$.items"), testastic.Update())

	// then: the update preserves the matcher without changing the existing update policy
	if mt.failed {
		t.Fatalf("expected update to succeed, got: %s", mt.message)
	}

	if got := calls.Load(); got != 0 {
		t.Errorf("Matcher.Match calls = %d, want 0", got)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("read updated expected file: %v", err)
	}

	if !strings.Contains(string(content), expression) || !strings.Contains(string(content), `"added"`) {
		t.Errorf("updated JSON did not preserve the positional matcher: %s", content)
	}
}

func TestAssertYAML_UpdateUnequalUnorderedArrayPreservesMatcher(t *testing.T) {
	t.Parallel()

	// given: an unordered YAML array contains a matcher to preserve

	const (
		matcher    = "comparisonYAMLUpdateValue"
		expression = "{{comparisonYAMLUpdateValue}}"
	)

	testastic.RegisterMatcher(matcher, func(string) (testastic.Matcher, error) {
		return comparisonValueMatcher{calls: &atomic.Int32{}, expression: expression, want: "kept"}, nil
	})

	expectedFile := writeComparisonExpected(t, "expected.yaml", "items: ["+expression+"]\n")
	mt := &mockT{}

	// when: update mode receives an additional actual array element
	testastic.AssertYAML(mt, expectedFile, "items: [kept, added]\n",
		testastic.IgnoreArrayOrderAt("$.items"), testastic.Update())

	// then: the update preserves the matcher without changing the existing update policy
	if mt.failed {
		t.Fatalf("expected update to succeed, got: %s", mt.message)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("read updated expected file: %v", err)
	}

	if !strings.Contains(string(content), expression) || !strings.Contains(string(content), "added") {
		t.Errorf("updated YAML did not preserve the matched matcher: %s", content)
	}
}

func writeComparisonExpected(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("write expected file: %v", err)
	}

	return path
}

func TestStructuredDiagnosticUnequalArrays(t *testing.T) {
	t.Parallel()

	formats := []struct {
		name   string
		assert func(testing.TB, string, string, ...testastic.Option)
	}{
		{name: "JSON", assert: testastic.AssertJSON[string]},
		{name: "YAML", assert: testastic.AssertYAML[string]},
	}
	cases := []struct {
		name       string
		expected   string
		actual     string
		unordered  bool
		wantMarker bool
	}{
		{name: "extra actual", expected: `["{{anyInt}}"]`, actual: `[1,"added"]`, unordered: true},
		{name: "missing literal", expected: `["{{anyInt}}","tail"]`, actual: `[1]`, unordered: true},
		{name: "missing matcher", expected: `[1,"{{anyInt}}"]`, actual: `[1]`, unordered: true, wantMarker: true},
		{name: "ordered missing matcher", expected: `[1,"{{anyInt}}"]`, actual: `[1]`, wantMarker: true},
		{name: "ordered extra actual", expected: `["{{anyInt}}"]`, actual: `[1,"added"]`},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			t.Parallel()

			for _, test := range cases {
				t.Run(test.name, func(t *testing.T) {
					t.Parallel()

					// given: expected and actual arrays have different lengths
					expectedFile := writeComparisonExpected(t, "expected", test.expected)
					mt := &mockT{}

					var opts []testastic.Option
					if test.unordered {
						opts = append(opts, testastic.IgnoreArrayOrder())
					}

					// when: the assertion reports the length mismatch
					format.assert(mt, expectedFile, test.actual, opts...)

					// then: only matchers without a matching actual value remain in the diagnostic
					if !mt.failed || mt.fatal {
						t.Fatalf("expected a nonfatal length mismatch, got: %s", mt.message)
					}

					if got := strings.Contains(mt.message, "{{anyInt}}"); got != test.wantMarker {
						t.Errorf("diagnostic contains matcher = %t, want %t: %s", got, test.wantMarker, mt.message)
					}
				})
			}
		})
	}
}
