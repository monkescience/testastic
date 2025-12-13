package testastic

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	// Test exact match
	actual := `{"name": "Alice", "age": 30, "active": true}`
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_Mismatch(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "mismatch.expected.json")

	expected := `{"name": "Alice", "age": 30}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	// Use a mock testing.T to capture the error
	mt := &mockT{}
	actual := `{"name": "Bob", "age": 25}`
	AssertJSON(mt, expectedFile, actual)

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
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	// Should match any string for id
	actual := `{"id": "abc-123-xyz", "name": "Alice"}`
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithAnyIntMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "any_int.expected.json")

	expected := `{
  "count": "{{anyInt}}",
  "name": "test"
}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := `{"count": 42, "name": "test"}`
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithIgnoreMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "ignore.expected.json")

	expected := `{
  "id": "{{ignore}}",
  "timestamp": "{{ignore}}",
  "name": "Alice"
}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := `{"id": 12345, "timestamp": "2024-01-15T10:30:00Z", "name": "Alice"}`
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithRegexMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "regex.expected.json")

	expected := "{\"email\": \"{{regex `^[a-z]+@example\\.com$`}}\"}"
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := `{"email": "alice@example.com"}`
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_WithOneOfMatcher(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "one_of.expected.json")

	// Use regular string with escaped quotes to get actual quote characters in file
	expected := "{\"status\": \"{{oneOf \\\"pending\\\" \\\"active\\\" \\\"completed\\\"}}\"}"
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := `{"status": "active"}`
	AssertJSON(t, expectedFile, actual)
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
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := `{"user": {"id": "usr-123", "profile": {"name": "Alice", "age": 30}}}`
	AssertJSON(t, expectedFile, actual)
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
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := `{"items": [{"id": 1, "name": "first"}, {"id": 2, "name": "second"}]}`
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_IgnoreArrayOrder(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "array_order.expected.json")

	expected := `{"tags": ["a", "b", "c"]}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	// Different order
	actual := `{"tags": ["c", "a", "b"]}`
	AssertJSON(t, expectedFile, actual, IgnoreArrayOrder())
}

func TestAssertJSON_IgnoreArrayOrderAt(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "array_order_at.expected.json")

	expected := `{"ordered": [1, 2, 3], "unordered": ["a", "b", "c"]}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := `{"ordered": [1, 2, 3], "unordered": ["c", "a", "b"]}`
	AssertJSON(t, expectedFile, actual, IgnoreArrayOrderAt("$.unordered"))
}

func TestAssertJSON_IgnoreFields(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "ignore_fields.expected.json")

	expected := `{"id": "fixed", "name": "Alice", "timestamp": "2024-01-01"}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	// id and timestamp are different but ignored
	actual := `{"id": "different", "name": "Alice", "timestamp": "2024-12-15"}`
	AssertJSON(t, expectedFile, actual, IgnoreFields("id", "timestamp"))
}

func TestAssertJSON_FromStruct(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "struct.expected.json")

	expected := `{"name": "Alice", "age": 30}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	actual := User{Name: "Alice", Age: 30}
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_FromReader(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "reader.expected.json")

	expected := `{"name": "Alice"}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	actual := bytes.NewReader([]byte(`{"name": "Alice"}`))
	AssertJSON(t, expectedFile, actual)
}

func TestAssertJSON_ExtraField(t *testing.T) {
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "extra.expected.json")

	expected := `{"name": "Alice"}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	mt := &mockT{}
	actual := `{"name": "Alice", "extra": "field"}`
	AssertJSON(mt, expectedFile, actual)

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

	expected := `{"name": "Alice", "age": 30}`
	if err := os.WriteFile(expectedFile, []byte(expected), 0644); err != nil {
		t.Fatal(err)
	}

	mt := &mockT{}
	actual := `{"name": "Alice"}`
	AssertJSON(mt, expectedFile, actual)

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
			_, err := parseMatcher(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMatcher(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestMatchers(t *testing.T) {
	t.Run("AnyString", func(t *testing.T) {
		m := AnyString()
		if !m.Match("hello") {
			t.Error("expected to match string")
		}
		if m.Match(123) {
			t.Error("expected not to match int")
		}
	})

	t.Run("AnyInt", func(t *testing.T) {
		m := AnyInt()
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
		m := AnyFloat()
		if !m.Match(float64(42.5)) {
			t.Error("expected to match float")
		}
		if !m.Match(float64(42)) {
			t.Error("expected to match integer")
		}
	})

	t.Run("AnyBool", func(t *testing.T) {
		m := AnyBool()
		if !m.Match(true) {
			t.Error("expected to match bool")
		}
		if m.Match("true") {
			t.Error("expected not to match string")
		}
	})

	t.Run("AnyValue", func(t *testing.T) {
		m := AnyValue()
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
		m, err := Regex(`^\d{4}-\d{2}-\d{2}$`)
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
		m := OneOf("a", "b", "c")
		if !m.Match("a") {
			t.Error("expected to match 'a'")
		}
		if m.Match("d") {
			t.Error("expected not to match 'd'")
		}
	})
}

func TestFormatDiff(t *testing.T) {
	diffs := []Difference{
		{Path: "$.name", Expected: "Alice", Actual: "Bob", Type: DiffChanged},
		{Path: "$.age", Expected: float64(30), Actual: nil, Type: DiffRemoved},
		{Path: "$.extra", Expected: nil, Actual: "value", Type: DiffAdded},
	}

	output := FormatDiff(diffs)

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
