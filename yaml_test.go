package testastic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

// --- AssertYAML Exact Match Tests ---

func TestAssertYAML_ExactMatch(t *testing.T) {
	// given: an expected YAML file with exact content
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	yamlContent := `name: test
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(yamlContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()

	// when: asserting with matching YAML
	testastic.AssertYAML(mt, expectedFile, yamlContent)

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure, got: %s", mt.message)
	}
}

func TestAssertYAML_ExactMatch_NestedStructure(t *testing.T) {
	// given: an expected YAML file with nested content
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	yamlContent := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  key1: value1
  key2: value2
`

	err := os.WriteFile(expectedFile, []byte(yamlContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()

	// when: asserting with matching nested YAML
	testastic.AssertYAML(mt, expectedFile, yamlContent)

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure, got: %s", mt.message)
	}
}

// --- Matcher Tests ---

func TestAssertYAML_WithAnyStringMatcher(t *testing.T) {
	// given: an expected YAML file with anyString matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: {{anyString}}
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
version: "1.0"
`

	// when: asserting with any string in the name field
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test passes (matcher accepts any string)
	if mt.failed {
		t.Errorf("expected no failure with anyString matcher, got: %s", mt.message)
	}
}

func TestAssertYAML_WithRegexMatcher(t *testing.T) {
	// given: an expected YAML file with regex matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := "name: {{regex `^app-[a-z]+$`}}\nversion: \"1.0\"\n"

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: app-test
version: "1.0"
`

	// when: asserting with a value matching the regex
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test passes (regex matches)
	if mt.failed {
		t.Errorf("expected no failure with regex matcher, got: %s", mt.message)
	}
}

func TestAssertYAML_WithRegexMatcher_Fails(t *testing.T) {
	// given: an expected YAML file with regex matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := "name: {{regex `^app-[a-z]+$`}}\nversion: \"1.0\"\n"

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: invalid-123
version: "1.0"
`

	// when: asserting with a value not matching the regex
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test fails
	if !mt.failed {
		t.Error("expected failure with non-matching regex")
	}
}

func TestAssertYAML_WithIgnoreMatcher(t *testing.T) {
	// given: an expected YAML file with ignore matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: my-app
timestamp: {{ignore}}
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
timestamp: "2024-01-15T10:30:00Z"
version: "1.0"
`

	// when: asserting with any value in the ignored field
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test passes (ignored content is not compared)
	if mt.failed {
		t.Errorf("expected no failure with ignore matcher, got: %s", mt.message)
	}
}

func TestAssertYAML_WithAnyIntMatcher(t *testing.T) {
	// given: an expected YAML file with anyInt matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: my-app
replicas: {{anyInt}}
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
replicas: 3
`

	// when: asserting with any integer in replicas
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure with anyInt matcher, got: %s", mt.message)
	}
}

func TestAssertYAML_WithOneOfMatcher(t *testing.T) {
	// given: an expected YAML file with oneOf matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: my-app
environment: {{oneOf "dev" "staging" "prod"}}
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
environment: staging
`

	// when: asserting with a value from the oneOf list
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure with oneOf matcher, got: %s", mt.message)
	}
}

func TestAssertYAML_WithOneOfMatcher_Fails(t *testing.T) {
	// given: an expected YAML file with oneOf matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: my-app
environment: {{oneOf "dev" "staging" "prod"}}
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
environment: test
`

	// when: asserting with a value not in the oneOf list
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test fails
	if !mt.failed {
		t.Error("expected failure with non-matching oneOf value")
	}
}

// --- Difference Detection Tests ---

func TestAssertYAML_MismatchValue(t *testing.T) {
	// given: an expected YAML file with a specific value
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: expected-name
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: actual-name
version: "1.0"
`

	// when: asserting with a different value
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test fails
	if !mt.failed {
		t.Error("expected failure for mismatched value")
	}
}

func TestAssertYAML_MissingField(t *testing.T) {
	// given: an expected YAML file with multiple fields
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: my-app
version: "1.0"
description: My app description
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
version: "1.0"
`

	// when: asserting with YAML missing a field
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test fails
	if !mt.failed {
		t.Error("expected failure for missing field")
	}
}

func TestAssertYAML_ExtraField(t *testing.T) {
	// given: an expected YAML file with specific fields
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: my-app
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
version: "1.0"
extra: unexpected-field
`

	// when: asserting with YAML containing an extra field
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test fails
	if !mt.failed {
		t.Error("expected failure for extra field")
	}
}

// --- Array Tests ---

func TestAssertYAML_ArrayExactMatch(t *testing.T) {
	// given: an expected YAML file with an array
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `items:
  - name: item1
  - name: item2
  - name: item3
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()

	// when: asserting with matching array
	testastic.AssertYAML(mt, expectedFile, expected)

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure, got: %s", mt.message)
	}
}

func TestAssertYAML_ArrayOrderMatters(t *testing.T) {
	// given: an expected YAML file with an array in specific order
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `items:
  - name: first
  - name: second
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `items:
  - name: second
  - name: first
`

	// when: asserting with reordered array (order matters by default)
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test fails
	if !mt.failed {
		t.Error("expected failure for reordered array")
	}
}

func TestAssertYAML_ArrayIgnoreOrder(t *testing.T) {
	// given: an expected YAML file with an array
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `items:
  - name: first
  - name: second
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `items:
  - name: second
  - name: first
`

	// when: asserting with IgnoreArrayOrder option
	testastic.AssertYAML(mt, expectedFile, actual, testastic.YAMLIgnoreArrayOrder())

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure with IgnoreArrayOrder, got: %s", mt.message)
	}
}

func TestAssertYAML_ArrayIgnoreOrderAt(t *testing.T) {
	// given: an expected YAML file with multiple arrays
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `items:
  - name: first
  - name: second
tags:
  - a
  - b
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `items:
  - name: second
  - name: first
tags:
  - b
  - a
`

	// when: asserting with IgnoreArrayOrderAt for items only
	testastic.AssertYAML(mt, expectedFile, actual, testastic.YAMLIgnoreArrayOrderAt("$.items"))

	// then: the test fails because tags order still matters
	if !mt.failed {
		t.Error("expected failure because tags order should still matter")
	}
}

// --- Option Tests ---

func TestAssertYAML_IgnoreFields(t *testing.T) {
	// given: an expected YAML file
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	expected := `name: my-app
timestamp: "2024-01-01T00:00:00Z"
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := `name: my-app
timestamp: "2024-12-31T23:59:59Z"
version: "1.0"
`

	// when: asserting with IgnoreFields option
	testastic.AssertYAML(mt, expectedFile, actual, testastic.YAMLIgnoreFields("timestamp"))

	// then: the test passes (timestamp is ignored)
	if mt.failed {
		t.Errorf("expected no failure with IgnoreFields, got: %s", mt.message)
	}
}

// --- Update Mode Tests ---

func TestAssertYAML_CreateExpectedFile(t *testing.T) {
	// given: a non-existent expected file path
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "new-expected.yaml")

	mt := newMockT()
	actual := `name: my-app
version: "1.0"
`

	// when: asserting with YAMLUpdate option
	testastic.AssertYAML(mt, expectedFile, actual, testastic.YAMLUpdate())

	// then: the test passes and the file is created
	if mt.failed {
		t.Errorf("expected no failure when creating file, got: %s", mt.message)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("expected file was not created: %v", err)
	}

	if !strings.Contains(string(content), "my-app") {
		t.Errorf("expected file content incorrect: %s", content)
	}
}

// --- Input Type Tests ---

func TestAssertYAML_ByteSliceInput(t *testing.T) {
	// given: an expected YAML file and actual as []byte
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	yamlContent := `name: test
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(yamlContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()

	// when: asserting with []byte input
	testastic.AssertYAML(mt, expectedFile, []byte(yamlContent))

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure with []byte input, got: %s", mt.message)
	}
}

func TestAssertYAML_StructInput(t *testing.T) {
	// given: an expected YAML file and actual as a struct
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	type Config struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	expected := `name: my-app
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()
	actual := Config{Name: "my-app", Version: "1.0"}

	// when: asserting with struct input (auto-marshaled)
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure with struct input, got: %s", mt.message)
	}
}

func TestAssertYAML_ReaderInput(t *testing.T) {
	// given: an expected YAML file and actual as io.Reader
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.yaml")

	yamlContent := `name: test
version: "1.0"
`

	err := os.WriteFile(expectedFile, []byte(yamlContent), 0o644)
	if err != nil {
		t.Fatalf("failed to create expected file: %v", err)
	}

	mt := newMockT()

	// when: asserting with io.Reader input
	testastic.AssertYAML(mt, expectedFile, strings.NewReader(yamlContent))

	// then: the test passes
	if mt.failed {
		t.Errorf("expected no failure with io.Reader input, got: %s", mt.message)
	}
}

// --- Parse Tests ---

func TestParseExpectedYAMLString_WithMatchers(t *testing.T) {
	// given: a YAML string with matchers
	input := `name: {{anyString}}
version: "1.0"
`

	// when: parsing the expected YAML string
	result, err := testastic.ParseExpectedYAMLString(input)
	// then: the result contains the matcher
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Data == nil {
		t.Fatal("expected data to be non-nil")
	}

	data, ok := result.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be map[string]any, got %T", result.Data)
	}

	if _, ok := data["name"].(testastic.Matcher); !ok {
		t.Errorf("expected 'name' field to be a Matcher, got %T", data["name"])
	}
}

// --- Diff Output Tests ---

func TestFormatYAMLDiffInline(t *testing.T) {
	// given: expected and actual YAML data with different values
	expected := map[string]any{
		"name":    "Alice",
		"version": "1.0",
	}

	actual := map[string]any{
		"name":    "Bob",
		"version": "1.0",
	}

	// when: formatting the diff
	result := testastic.FormatYAMLDiffInline(expected, actual)

	// then: the diff contains both expected and actual values
	if !strings.Contains(result, "Alice") {
		t.Error("expected diff to contain 'Alice'")
	}

	if !strings.Contains(result, "Bob") {
		t.Error("expected diff to contain 'Bob'")
	}
}
