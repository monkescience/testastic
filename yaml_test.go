package testastic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertYAML(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
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
	})

	t.Run("exact match nested structure", func(t *testing.T) {
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
	})

	t.Run("with anyString matcher", func(t *testing.T) {
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
	})

	t.Run("with regex matcher", func(t *testing.T) {
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
	})

	t.Run("with regex matcher fails", func(t *testing.T) {
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
	})

	t.Run("with ignore matcher", func(t *testing.T) {
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
	})

	t.Run("with anyInt matcher", func(t *testing.T) {
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
	})

	t.Run("with oneOf matcher", func(t *testing.T) {
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
	})

	t.Run("with oneOf matcher fails", func(t *testing.T) {
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
	})

	t.Run("mismatch value", func(t *testing.T) {
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
	})

	t.Run("missing field", func(t *testing.T) {
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
	})

	t.Run("extra field", func(t *testing.T) {
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
	})

	t.Run("array exact match", func(t *testing.T) {
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
	})

	t.Run("array order matters", func(t *testing.T) {
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
	})

	t.Run("array ignore order", func(t *testing.T) {
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
		testastic.AssertYAML(mt, expectedFile, actual, testastic.IgnoreArrayOrder())

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with IgnoreArrayOrder, got: %s", mt.message)
		}
	})

	t.Run("array ignore order at", func(t *testing.T) {
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
		testastic.AssertYAML(mt, expectedFile, actual, testastic.IgnoreArrayOrderAt("$.items"))

		// then: the test fails because tags order still matters
		if !mt.failed {
			t.Error("expected failure because tags order should still matter")
		}
	})

	t.Run("ignore fields", func(t *testing.T) {
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
		testastic.AssertYAML(mt, expectedFile, actual, testastic.IgnoreFields("timestamp"))

		// then: the test passes (timestamp is ignored)
		if mt.failed {
			t.Errorf("expected no failure with IgnoreFields, got: %s", mt.message)
		}
	})

	t.Run("create expected file", func(t *testing.T) {
		// given: a non-existent expected file path
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "new-expected.yaml")

		mt := newMockT()
		actual := `name: my-app
version: "1.0"
`

		// when: asserting with YAMLUpdate option
		testastic.AssertYAML(mt, expectedFile, actual, testastic.Update())

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
	})

	t.Run("byte slice input", func(t *testing.T) {
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
	})

	t.Run("struct input", func(t *testing.T) {
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
	})

	t.Run("reader input", func(t *testing.T) {
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
	})
}
