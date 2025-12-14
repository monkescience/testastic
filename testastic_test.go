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
	// Create temp expected file
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "exact.expected.json")

	expected := `{
  "name": "Alice",
  "age": 30,
  "active": true
}`
	writeTestFile(t, expectedFile, expected)

	// Test exact match
	testastic.AssertJSON(t, expectedFile, testJSONAliceAge30Full)
}

func TestAssertJSON_Mismatch(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "mismatch.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceAge30)

	// Use a mock testing.T to capture the error
	mt := &mockT{}
	actual := `{"name": "Bob", "age": 25}`
	testastic.AssertJSON(mt, expectedFile, actual)

	if !mt.failed {
		t.Error("expected test to fail")
	}
	// New diff format shows actual JSON lines with - and + prefixes
	if !strings.Contains(mt.output, `"name"`) {
		t.Errorf("expected diff to mention name field, got: %s", mt.output)
	}

	if !strings.Contains(mt.output, `"age"`) {
		t.Errorf("expected diff to mention age field, got: %s", mt.output)
	}
}

func TestAssertJSON_WithAnyStringMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "any_string.expected.json")

	expected := `{
  "id": "{{anyString}}",
  "name": "Alice"
}`
	writeTestFile(t, expectedFile, expected)

	// Should match any string for id
	actual := `{"id": "abc-123-xyz", "name": "Alice"}`
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithAnyIntMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "any_int.expected.json")

	expected := `{
  "count": "{{anyInt}}",
  "name": "test"
}`
	writeTestFile(t, expectedFile, expected)

	actual := `{"count": 42, "name": "test"}`
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithIgnoreMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "ignore.expected.json")

	expected := `{
  "id": "{{ignore}}",
  "timestamp": "{{ignore}}",
  "name": "Alice"
}`
	writeTestFile(t, expectedFile, expected)

	actual := `{"id": 12345, "timestamp": "2024-01-15T10:30:00Z", "name": "Alice"}`
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithRegexMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "regex.expected.json")

	expected := "{\"email\": \"{{regex `^[a-z]+@example\\.com$`}}\"}"
	writeTestFile(t, expectedFile, expected)

	actual := `{"email": "alice@example.com"}`
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithOneOfMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "one_of.expected.json")

	// Use regular string with escaped quotes to get actual quote characters in file
	expected := "{\"status\": \"{{oneOf \\\"pending\\\" \\\"active\\\" \\\"completed\\\"}}\"}"
	writeTestFile(t, expectedFile, expected)

	actual := `{"status": "active"}`
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_NestedObjects(t *testing.T) {
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

	actual := `{"user": {"id": "usr-123", "profile": {"name": "Alice", "age": 30}}}`
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_Arrays(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "arrays.expected.json")

	expected := `{
  "items": [
    {"id": 1, "name": "first"},
    {"id": 2, "name": "second"}
  ]
}`
	writeTestFile(t, expectedFile, expected)

	actual := `{"items": [{"id": 1, "name": "first"}, {"id": 2, "name": "second"}]}`
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_IgnoreArrayOrder(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "array_order.expected.json")

	expected := `{"tags": ["a", "b", "c"]}`
	writeTestFile(t, expectedFile, expected)

	// Different order
	actual := `{"tags": ["c", "a", "b"]}`
	testastic.AssertJSON(t, expectedFile, actual, testastic.IgnoreArrayOrder())
}

func TestAssertJSON_IgnoreArrayOrderAt(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "array_order_at.expected.json")

	expected := `{"ordered": [1, 2, 3], "unordered": ["a", "b", "c"]}`
	writeTestFile(t, expectedFile, expected)

	actual := `{"ordered": [1, 2, 3], "unordered": ["c", "a", "b"]}`
	testastic.AssertJSON(t, expectedFile, actual, testastic.IgnoreArrayOrderAt("$.unordered"))
}

func TestAssertJSON_IgnoreFields(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "ignore_fields.expected.json")

	expected := `{"id": "fixed", "name": "Alice", "timestamp": "2024-01-01"}`
	writeTestFile(t, expectedFile, expected)

	// id and timestamp are different but ignored
	actual := `{"id": "different", "name": "Alice", "timestamp": "2024-12-15"}`
	testastic.AssertJSON(t, expectedFile, actual, testastic.IgnoreFields("id", "timestamp"))
}

func TestAssertJSON_FromStruct(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "struct.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceAge30)

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	actual := User{Name: "Alice", Age: 30}
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_FromReader(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "reader.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceOnly)

	actual := bytes.NewReader([]byte(testJSONAliceOnly))
	testastic.AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_ExtraField(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "extra.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceOnly)

	mt := &mockT{}
	actual := `{"name": "Alice", "extra": "field"}`
	testastic.AssertJSON(mt, expectedFile, actual)

	if !mt.failed {
		t.Error("expected test to fail due to extra field")
	}
	// New diff format shows actual JSON with + prefix for added lines
	if !strings.Contains(mt.output, `"extra"`) {
		t.Errorf("expected diff to mention extra field, got: %s", mt.output)
	}
}

func TestAssertJSON_MissingField(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "missing.expected.json")

	writeTestFile(t, expectedFile, testJSONAliceAge30)

	mt := &mockT{}
	testastic.AssertJSON(mt, expectedFile, testJSONAliceOnly)

	if !mt.failed {
		t.Error("expected test to fail due to missing field")
	}
	// New diff format shows actual JSON with - prefix for removed lines
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
			_, err := testastic.ParseMatcher(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMatcher(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestMatchers(t *testing.T) {
	t.Run("AnyString", func(t *testing.T) {
		m := testastic.AnyString()

		if !m.Match("hello") {
			t.Error("expected to match string")
		}

		if m.Match(123) {
			t.Error("expected not to match int")
		}
	})

	t.Run("AnyInt", func(t *testing.T) {
		m := testastic.AnyInt()

		if !m.Match(float64(42)) {
			t.Error("expected to match integer float64")
		}

		if m.Match(42.5) {
			t.Error("expected not to match non-integer float")
		}

		if m.Match("42") {
			t.Error("expected not to match string")
		}
	})

	t.Run("AnyFloat", func(t *testing.T) {
		m := testastic.AnyFloat()

		if !m.Match(float64(42.5)) {
			t.Error("expected to match float")
		}

		if !m.Match(float64(42)) {
			t.Error("expected to match integer")
		}
	})

	t.Run("AnyBool", func(t *testing.T) {
		m := testastic.AnyBool()

		if !m.Match(true) {
			t.Error("expected to match bool")
		}

		if m.Match("true") {
			t.Error("expected not to match string")
		}
	})

	t.Run("AnyValue", func(t *testing.T) {
		m := testastic.AnyValue()

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
		m, err := testastic.Regex(`^\d{4}-\d{2}-\d{2}$`)
		if err != nil {
			t.Fatal(err)
		}

		if !m.Match("2024-01-15") {
			t.Error("expected to match date format")
		}

		if m.Match("invalid") {
			t.Error("expected not to match invalid format")
		}
	})

	t.Run("OneOf", func(t *testing.T) {
		m := testastic.OneOf("a", "b", "c")

		if !m.Match("a") {
			t.Error("expected to match 'a'")
		}

		if m.Match("d") {
			t.Error("expected not to match 'd'")
		}
	})
}

func TestFormatDiff(t *testing.T) {
	diffs := []testastic.Difference{
		{Path: "$.name", Expected: "Alice", Actual: "Bob", Type: testastic.DiffChanged},
		{Path: "$.age", Expected: float64(30), Actual: nil, Type: testastic.DiffRemoved},
		{Path: "$.extra", Expected: nil, Actual: "value", Type: testastic.DiffAdded},
	}

	output := testastic.FormatDiff(diffs)

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
