package testastic_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

// Test data constants.
const (
	testJSONAliceAge30     = `{"name": "Alice", "age": 30}`
	testJSONAliceOnly      = `{"name": "Alice"}`
	testJSONAliceAge30Full = `{"name": "Alice", "age": 30, "active": true}`
)

func TestAssertJSON_ExactMatch(t *testing.T) {
	// GIVEN: an expected JSON file with exact values
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "exact.expected.json")

	expected := `{
  "name": "Alice",
  "age": 30,
  "active": true
}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with matching JSON
	// THEN: the test passes without failure
	testastic.AssertJSON(t, expectedFile, testJSONAliceAge30Full)
}

func TestAssertJSON_Mismatch(t *testing.T) {
	// GIVEN: an expected JSON file and non-matching actual JSON
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "mismatch.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceAge30)

	mt := &mockT{}
	actual := `{"name": "Bob", "age": 25}`

	// WHEN: asserting with mismatched JSON
	testastic.AssertJSON(mt, expectedFile, actual)

	// THEN: the test fails and diff mentions the differing fields
	if !mt.failed {
		t.Error("expected test to fail")
	}

	if !strings.Contains(mt.output, `"name"`) {
		t.Errorf("expected diff to mention name field, got: %s", mt.output)
	}

	if !strings.Contains(mt.output, `"age"`) {
		t.Errorf("expected diff to mention age field, got: %s", mt.output)
	}
}

func TestAssertJSON_WithAnyStringMatcher(t *testing.T) {
	// GIVEN: an expected JSON file with anyString matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "any_string.expected.json")

	expected := `{
  "id": "{{anyString}}",
  "name": "Alice"
}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with any string value for id
	actual := `{"id": "abc-123-xyz", "name": "Alice"}`

	// THEN: the test passes (matcher accepts any string)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithAnyIntMatcher(t *testing.T) {
	// GIVEN: an expected JSON file with anyInt matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "any_int.expected.json")

	expected := `{
  "count": "{{anyInt}}",
  "name": "test"
}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with any integer value for count
	actual := `{"count": 42, "name": "test"}`

	// THEN: the test passes (matcher accepts any integer)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithIgnoreMatcher(t *testing.T) {
	// GIVEN: an expected JSON file with ignore matchers
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "ignore.expected.json")

	expected := `{
  "id": "{{ignore}}",
  "timestamp": "{{ignore}}",
  "name": "Alice"
}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with any values for ignored fields
	actual := `{"id": 12345, "timestamp": "2024-01-15T10:30:00Z", "name": "Alice"}`

	// THEN: the test passes (ignored fields are not compared)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithRegexMatcher(t *testing.T) {
	// GIVEN: an expected JSON file with regex matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "regex.expected.json")

	expected := "{\"email\": \"{{regex `^[a-z]+@example\\.com$`}}\"}"
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with a value matching the regex pattern
	actual := `{"email": "alice@example.com"}`

	// THEN: the test passes (value matches regex)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithOneOfMatcher(t *testing.T) {
	// GIVEN: an expected JSON file with oneOf matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "one_of.expected.json")

	expected := "{\"status\": \"{{oneOf \\\"pending\\\" \\\"active\\\" \\\"completed\\\"}}\"}"
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with a value from the allowed set
	actual := `{"status": "active"}`

	// THEN: the test passes (value is one of the allowed values)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_NestedObjects(t *testing.T) {
	// GIVEN: an expected JSON file with nested objects and matchers
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "nested.expected.json")

	expected := `{
  "user": {
    "id": "{{anyString}}",
    "profile": {
      "name": "Alice",
      "age": 30
    }
  }
}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with matching nested structure
	actual := `{"user": {"id": "usr-123", "profile": {"name": "Alice", "age": 30}}}`

	// THEN: the test passes (nested structure matches)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_Arrays(t *testing.T) {
	// GIVEN: an expected JSON file with arrays
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "arrays.expected.json")

	expected := `{
  "items": [
    {"id": 1, "name": "first"},
    {"id": 2, "name": "second"}
  ]
}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with matching array content and order
	actual := `{"items": [{"id": 1, "name": "first"}, {"id": 2, "name": "second"}]}`

	// THEN: the test passes (array matches exactly)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_IgnoreArrayOrder(t *testing.T) {
	// GIVEN: an expected JSON file with an array
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "array_order.expected.json")

	expected := `{"tags": ["a", "b", "c"]}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with same elements in different order using IgnoreArrayOrder
	actual := `{"tags": ["c", "a", "b"]}`

	// THEN: the test passes (order is ignored)
	testastic.AssertJSON(t, expectedFile, actual, testastic.IgnoreArrayOrder())
}

func TestAssertJSON_IgnoreArrayOrderAt(t *testing.T) {
	// GIVEN: an expected JSON file with ordered and unordered arrays
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "array_order_at.expected.json")

	expected := `{"ordered": [1, 2, 3], "unordered": ["a", "b", "c"]}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with different order only in the unordered array
	actual := `{"ordered": [1, 2, 3], "unordered": ["c", "a", "b"]}`

	// THEN: the test passes (order ignored only at specified path)
	testastic.AssertJSON(t, expectedFile, actual, testastic.IgnoreArrayOrderAt("$.unordered"))
}

func TestAssertJSON_IgnoreFields(t *testing.T) {
	// GIVEN: an expected JSON file with fields to ignore
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "ignore_fields.expected.json")

	expected := `{"id": "fixed", "name": "Alice", "timestamp": "2024-01-01"}`
	writeTestFile(t, expectedFile, expected)

	// WHEN: asserting with different values for ignored fields
	actual := `{"id": "different", "name": "Alice", "timestamp": "2024-12-15"}`

	// THEN: the test passes (specified fields are ignored)
	testastic.AssertJSON(t, expectedFile, actual, testastic.IgnoreFields("id", "timestamp"))
}

func TestAssertJSON_FromStruct(t *testing.T) {
	// GIVEN: an expected JSON file and a Go struct with matching data
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "struct.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceAge30)

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	actual := User{Name: "Alice", Age: 30}

	// WHEN: asserting with the struct as actual value
	// THEN: the test passes (struct is serialized and matches)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_FromReader(t *testing.T) {
	// GIVEN: an expected JSON file and an io.Reader with matching content
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "reader.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceOnly)

	actual := bytes.NewReader([]byte(testJSONAliceOnly))

	// WHEN: asserting with the io.Reader as actual value
	// THEN: the test passes (reader content matches)
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_ExtraField(t *testing.T) {
	// GIVEN: an expected JSON file without an extra field
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "extra.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceOnly)

	mt := &mockT{}
	actual := `{"name": "Alice", "extra": "field"}`

	// WHEN: asserting with JSON containing an extra field
	testastic.AssertJSON(mt, expectedFile, actual)

	// THEN: the test fails and diff mentions the extra field
	if !mt.failed {
		t.Error("expected test to fail due to extra field")
	}

	if !strings.Contains(mt.output, `"extra"`) {
		t.Errorf("expected diff to mention extra field, got: %s", mt.output)
	}
}

func TestAssertJSON_MissingField(t *testing.T) {
	// GIVEN: an expected JSON file with a field that actual lacks
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "missing.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceAge30)

	mt := &mockT{}

	// WHEN: asserting with JSON missing the age field
	testastic.AssertJSON(mt, expectedFile, testJSONAliceOnly)

	// THEN: the test fails and diff mentions the missing field
	if !mt.failed {
		t.Error("expected test to fail due to missing field")
	}

	if !strings.Contains(mt.output, `"age"`) {
		t.Errorf("expected diff to mention age field, got: %s", mt.output)
	}
}

func TestParseMatcher(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"anyString", false},
		{"anyInt", false},
		{"anyFloat", false},
		{"anyBool", false},
		{"anyValue", false},
		{"ignore", false},
		{"regex `^test$`", false},
		{`oneOf "a" "b"`, false},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			// GIVEN: a matcher expression
			// WHEN: parsing the matcher expression
			_, err := testastic.ParseMatcher(tt.expr)

			// THEN: error status matches expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMatcher(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestMatchers(t *testing.T) {
	t.Run("AnyString", func(t *testing.T) {
		// GIVEN: an AnyString matcher
		m := testastic.AnyString()

		// WHEN: matching against a string
		// THEN: it matches
		if !m.Match("hello") {
			t.Error("expected to match string")
		}

		// WHEN: matching against an int
		// THEN: it does not match
		if m.Match(123) {
			t.Error("expected not to match int")
		}
	})

	t.Run("AnyInt", func(t *testing.T) {
		// GIVEN: an AnyInt matcher
		m := testastic.AnyInt()

		// WHEN: matching against an integer float64
		// THEN: it matches
		if !m.Match(float64(42)) {
			t.Error("expected to match integer float64")
		}

		// WHEN: matching against a non-integer float
		// THEN: it does not match
		if m.Match(42.5) {
			t.Error("expected not to match non-integer float")
		}

		// WHEN: matching against a string
		// THEN: it does not match
		if m.Match("42") {
			t.Error("expected not to match string")
		}
	})

	t.Run("AnyFloat", func(t *testing.T) {
		// GIVEN: an AnyFloat matcher
		m := testastic.AnyFloat()

		// WHEN: matching against a float
		// THEN: it matches
		if !m.Match(float64(42.5)) {
			t.Error("expected to match float")
		}

		// WHEN: matching against an integer (as float64)
		// THEN: it also matches
		if !m.Match(float64(42)) {
			t.Error("expected to match integer")
		}
	})

	t.Run("AnyBool", func(t *testing.T) {
		// GIVEN: an AnyBool matcher
		m := testastic.AnyBool()

		// WHEN: matching against a bool
		// THEN: it matches
		if !m.Match(true) {
			t.Error("expected to match bool")
		}

		// WHEN: matching against a string "true"
		// THEN: it does not match
		if m.Match("true") {
			t.Error("expected not to match string")
		}
	})

	t.Run("AnyValue", func(t *testing.T) {
		// GIVEN: an AnyValue matcher
		m := testastic.AnyValue()

		// WHEN: matching against any type
		// THEN: it always matches
		if !m.Match("hello") {
			t.Error("expected to match string")
		}

		if !m.Match(123) {
			t.Error("expected to match int")
		}

		if !m.Match(nil) {
			t.Error("expected to match nil")
		}
	})

	t.Run("Regex", func(t *testing.T) {
		// GIVEN: a Regex matcher for date format
		m, err := testastic.Regex(`^\d{4}-\d{2}-\d{2}$`)
		if err != nil {
			t.Fatal(err)
		}

		// WHEN: matching against a valid date string
		// THEN: it matches
		if !m.Match("2024-01-15") {
			t.Error("expected to match date format")
		}

		// WHEN: matching against an invalid format
		// THEN: it does not match
		if m.Match("invalid") {
			t.Error("expected not to match invalid format")
		}
	})

	t.Run("OneOf", func(t *testing.T) {
		// GIVEN: a OneOf matcher with allowed values
		m := testastic.OneOf("a", "b", "c")

		// WHEN: matching against an allowed value
		// THEN: it matches
		if !m.Match("a") {
			t.Error("expected to match 'a'")
		}

		// WHEN: matching against a non-allowed value
		// THEN: it does not match
		if m.Match("d") {
			t.Error("expected not to match 'd'")
		}
	})
}

func TestFormatDiff(t *testing.T) {
	// GIVEN: a list of differences
	diffs := []testastic.Difference{
		{Path: "$.name", Expected: "Alice", Actual: "Bob", Type: testastic.DiffChanged},
		{Path: "$.age", Expected: float64(30), Actual: nil, Type: testastic.DiffRemoved},
		{Path: "$.extra", Expected: nil, Actual: "value", Type: testastic.DiffAdded},
	}

	// WHEN: formatting the diff
	output := testastic.FormatDiff(diffs)

	// THEN: the output contains all expected information
	if !strings.Contains(output, "$.name") {
		t.Error("expected output to contain $.name")
	}

	if !strings.Contains(output, "Alice") {
		t.Error("expected output to contain Alice")
	}

	if !strings.Contains(output, "Bob") {
		t.Error("expected output to contain Bob")
	}

	if !strings.Contains(output, "(missing)") {
		t.Error("expected output to contain (missing)")
	}
}

// writeTestFile writes content to a file, failing the test on error.
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	err := os.WriteFile(path, []byte(content), 0o600)
	if err != nil {
		t.Fatal(err)
	}
}

// mockT is a mock testing.TB for capturing test failures.
type mockT struct {
	testing.TB
	failed bool
	output string
}

func (m *mockT) Helper() {}

func (m *mockT) Fatalf(format string, args ...any) {
	m.failed = true
	m.output = strings.TrimSpace(strings.ReplaceAll(format, "%v", ""))

	for _, arg := range args {
		if s, ok := arg.(string); ok {
			m.output += " " + s
		}
	}
}

func (m *mockT) Errorf(format string, args ...any) {
	m.failed = true
	m.output = strings.TrimSpace(strings.ReplaceAll(format, "%v", ""))

	for _, arg := range args {
		if s, ok := arg.(string); ok {
			m.output += " " + s
		}
	}
}

func (m *mockT) Logf(format string, args ...any) {}
