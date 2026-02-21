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
		mt := &mockT{}
		actual := "name: test\nversion: \"1.0\"\n"

		// when: asserting with matching YAML
		testastic.AssertYAML(mt, "testdata/yaml/exact_match.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})

	t.Run("exact match nested structure", func(t *testing.T) {
		// given: an expected YAML file with nested content
		mt := &mockT{}
		actual := `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  key1: value1
  key2: value2
`

		// when: asserting with matching nested YAML
		testastic.AssertYAML(mt, "testdata/yaml/nested_structure.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})

	t.Run("with anyString matcher", func(t *testing.T) {
		// given: an expected YAML file with anyString matcher
		mt := &mockT{}
		actual := "name: my-app\nversion: \"1.0\"\n"

		// when: asserting with any string in the name field
		testastic.AssertYAML(mt, "testdata/yaml/with_anystring.yaml", actual)

		// then: the test passes (matcher accepts any string)
		if mt.failed {
			t.Errorf("expected no failure with anyString matcher, got: %s", mt.message)
		}
	})

	t.Run("with regex matcher", func(t *testing.T) {
		// given: an expected YAML file with regex matcher
		mt := &mockT{}
		actual := "name: app-test\nversion: \"1.0\"\n"

		// when: asserting with a value matching the regex
		testastic.AssertYAML(mt, "testdata/yaml/with_regex.yaml", actual)

		// then: the test passes (regex matches)
		if mt.failed {
			t.Errorf("expected no failure with regex matcher, got: %s", mt.message)
		}
	})

	t.Run("with regex matcher fails", func(t *testing.T) {
		// given: an expected YAML file with regex matcher
		mt := &mockT{}
		actual := "name: invalid-123\nversion: \"1.0\"\n"

		// when: asserting with a value not matching the regex
		testastic.AssertYAML(mt, "testdata/yaml/with_regex_fails.yaml", actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching regex")
		}
	})

	t.Run("with ignore matcher", func(t *testing.T) {
		// given: an expected YAML file with ignore matcher
		mt := &mockT{}
		actual := "name: my-app\ntimestamp: \"2024-01-15T10:30:00Z\"\nversion: \"1.0\"\n"

		// when: asserting with any value in the ignored field
		testastic.AssertYAML(mt, "testdata/yaml/with_ignore.yaml", actual)

		// then: the test passes (ignored content is not compared)
		if mt.failed {
			t.Errorf("expected no failure with ignore matcher, got: %s", mt.message)
		}
	})

	t.Run("with anyInt matcher", func(t *testing.T) {
		// given: an expected YAML file with anyInt matcher
		mt := &mockT{}
		actual := "name: my-app\nreplicas: 3\n"

		// when: asserting with any integer in replicas
		testastic.AssertYAML(mt, "testdata/yaml/with_anyint.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with anyInt matcher, got: %s", mt.message)
		}
	})

	t.Run("with oneOf matcher", func(t *testing.T) {
		// given: an expected YAML file with oneOf matcher
		mt := &mockT{}
		actual := "name: my-app\nenvironment: staging\n"

		// when: asserting with a value from the oneOf list
		testastic.AssertYAML(mt, "testdata/yaml/with_oneof.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with oneOf matcher, got: %s", mt.message)
		}
	})

	t.Run("with oneOf matcher fails", func(t *testing.T) {
		// given: an expected YAML file with oneOf matcher
		mt := &mockT{}
		actual := "name: my-app-fail\nenvironment: test\n"

		// when: asserting with a value not in the oneOf list
		testastic.AssertYAML(mt, "testdata/yaml/with_oneof_fails.yaml", actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure with non-matching oneOf value")
		}
	})

	t.Run("mismatch value", func(t *testing.T) {
		// given: an expected YAML file with a specific value
		mt := &mockT{}
		actual := "name: actual-name\nversion: \"1.0\"\n"

		// when: asserting with a different value
		testastic.AssertYAML(mt, "testdata/yaml/mismatch_value.yaml", actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for mismatched value")
		}
	})

	t.Run("missing field", func(t *testing.T) {
		// given: an expected YAML file with multiple fields
		mt := &mockT{}
		actual := "name: my-app\nversion: \"1.0\"\n"

		// when: asserting with YAML missing a field
		testastic.AssertYAML(mt, "testdata/yaml/missing_field.yaml", actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for missing field")
		}
	})

	t.Run("extra field", func(t *testing.T) {
		// given: an expected YAML file with specific fields
		mt := &mockT{}
		actual := "name: expected-extra\nversion: \"1.0\"\nextra: unexpected-field\n"

		// when: asserting with YAML containing an extra field
		testastic.AssertYAML(mt, "testdata/yaml/extra_field.yaml", actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for extra field")
		}
	})

	t.Run("array exact match", func(t *testing.T) {
		// given: an expected YAML file with an array
		mt := &mockT{}
		actual := "items:\n  - name: item1\n  - name: item2\n  - name: item3\n"

		// when: asserting with matching array
		testastic.AssertYAML(mt, "testdata/yaml/array_exact.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure, got: %s", mt.message)
		}
	})

	t.Run("array order matters", func(t *testing.T) {
		// given: an expected YAML file with an array in specific order
		mt := &mockT{}
		actual := "items:\n  - name: second\n  - name: first\n"

		// when: asserting with reordered array (order matters by default)
		testastic.AssertYAML(mt, "testdata/yaml/array_order.yaml", actual)

		// then: the test fails
		if !mt.failed {
			t.Error("expected failure for reordered array")
		}
	})

	t.Run("array ignore order", func(t *testing.T) {
		// given: an expected YAML file with an array
		mt := &mockT{}
		actual := "items:\n  - name: beta\n  - name: alpha\n"

		// when: asserting with IgnoreArrayOrder option
		testastic.AssertYAML(mt, "testdata/yaml/array_order_ignore_order.yaml", actual, testastic.IgnoreArrayOrder())

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with IgnoreArrayOrder, got: %s", mt.message)
		}
	})

	t.Run("array ignore order at", func(t *testing.T) {
		// given: an expected YAML file with multiple arrays
		mt := &mockT{}
		actual := "items:\n  - name: second\n  - name: first\ntags:\n  - b\n  - a\n"

		// when: asserting with IgnoreArrayOrderAt for items only
		testastic.AssertYAML(mt, "testdata/yaml/array_multiple.yaml", actual, testastic.IgnoreArrayOrderAt("$.items"))

		// then: the test fails because tags order still matters
		if !mt.failed {
			t.Error("expected failure because tags order should still matter")
		}
	})

	t.Run("ignore fields", func(t *testing.T) {
		// given: an expected YAML file
		mt := &mockT{}
		actual := "name: my-app\ntimestamp: \"2024-12-31T23:59:59Z\"\nversion: \"1.0\"\n"

		// when: asserting with IgnoreFields option
		testastic.AssertYAML(mt, "testdata/yaml/ignore_fields.yaml", actual, testastic.IgnoreFields("timestamp"))

		// then: the test passes (timestamp is ignored)
		if mt.failed {
			t.Errorf("expected no failure with IgnoreFields, got: %s", mt.message)
		}
	})

	t.Run("create expected file", func(t *testing.T) {
		// given: a non-existent expected file path
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "new-expected.yaml")

		mt := &mockT{}
		actual := "name: my-app\nversion: \"1.0\"\n"

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

	t.Run("create expected file in nested directory", func(t *testing.T) {
		// given: a non-existent expected file path in nested directories
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "nested", "path", "new-expected.yaml")

		mt := &mockT{}
		actual := "name: my-app\nversion: \"1.0\"\n"

		// when: asserting with update option
		testastic.AssertYAML(mt, expectedFile, actual, testastic.Update())

		// then: the test passes and the nested file is created
		if mt.failed {
			t.Errorf("expected no failure when creating nested file, got: %s", mt.message)
		}

		content, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Fatalf("expected nested file was not created: %v", err)
		}

		if !strings.Contains(string(content), "my-app") {
			t.Errorf("expected file content incorrect: %s", content)
		}
	})

	t.Run("byte slice input", func(t *testing.T) {
		// given: an expected YAML file and actual as []byte
		mt := &mockT{}
		actual := []byte("name: byte-test\nversion: \"2.0\"\n")

		// when: asserting with []byte input
		testastic.AssertYAML(mt, "testdata/yaml/byte_slice.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with []byte input, got: %s", mt.message)
		}
	})

	t.Run("struct input", func(t *testing.T) {
		// given: an expected YAML file and actual as a struct
		type Config struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		}

		mt := &mockT{}
		actual := Config{Name: "struct-app", Version: "2.0"}

		// when: asserting with struct input (auto-marshaled)
		testastic.AssertYAML(mt, "testdata/yaml/struct_input.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with struct input, got: %s", mt.message)
		}
	})

	t.Run("reader input", func(t *testing.T) {
		// given: an expected YAML file and actual as io.Reader
		mt := &mockT{}
		actual := strings.NewReader("name: reader-test\nversion: \"3.0\"\n")

		// when: asserting with io.Reader input
		testastic.AssertYAML(mt, "testdata/yaml/reader_input.yaml", actual)

		// then: the test passes
		if mt.failed {
			t.Errorf("expected no failure with io.Reader input, got: %s", mt.message)
		}
	})
}

func TestAssertYAML_UpdatePreservesUnquotedMatchers(t *testing.T) {
	t.Run("update roundtrip preserves unquoted template expressions", func(t *testing.T) {
		// given: an expected YAML file with unquoted matchers and a matching actual value
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.yaml")

		originalContent := "name: {{anyString}}\nversion: \"1.0\"\n"

		err := os.WriteFile(expectedFile, []byte(originalContent), 0o644)
		if err != nil {
			t.Fatalf("failed to write expected file: %v", err)
		}

		mt := &mockT{}
		actual := "name: updated-name\nversion: \"2.0\"\n"

		// when: asserting with update mode (actual differs from expected)
		testastic.AssertYAML(mt, expectedFile, actual, testastic.Update())

		// then: the updated file preserves unquoted {{anyString}} syntax
		content, readErr := os.ReadFile(expectedFile)
		if readErr != nil {
			t.Fatalf("failed to read updated file: %v", readErr)
		}

		fileContent := string(content)

		if !strings.Contains(fileContent, "{{anyString}}") {
			t.Errorf("expected matcher to be preserved, got:\n%s", fileContent)
		}

		// Matchers must not be double-wrapped: {{{{anyString}}}} is wrong
		if strings.Contains(fileContent, "{{{{") {
			t.Errorf("matcher expressions are double-wrapped with braces:\n%s", fileContent)
		}

		// The matcher line should use unquoted form: name: {{anyString}}
		// not the YAML-quoted form: name: "{{anyString}}"
		if strings.Contains(fileContent, `name: "{{anyString}}"`) {
			t.Errorf("expected unquoted matcher (name: {{anyString}}), got YAML-quoted form:\n%s", fileContent)
		}
	})
}

func TestAssertYAML_UnsupportedOptions(t *testing.T) {
	t.Run("html options rejected", func(t *testing.T) {
		// given: HTML-only options passed to AssertYAML
		mt := &mockT{}

		// when: asserting with unsupported options
		testastic.AssertYAML(mt, "testdata/yaml/exact_match_unsupported_options.yaml",
			"name: test-unsupported\nversion: \"1.2\"\n", testastic.IgnoreElements("script"))

		// then: the test fatals with a message listing the unsupported option
		if !mt.fatal {
			t.Error("expected fatal error for unsupported options")
		}

		expectedMsg := "testastic: unsupported options for AssertYAML: IgnoreElements"
		if mt.message != expectedMsg {
			t.Errorf("expected message %q, got: %q", expectedMsg, mt.message)
		}
	})

	t.Run("supported options accepted", func(t *testing.T) {
		// given: supported options for AssertYAML
		// when: asserting with IgnoreArrayOrder
		// then: the test passes without unsupported option error
		testastic.AssertYAML(t, "testdata/yaml/exact_match_supported_options.yaml",
			"name: test-supported\nversion: \"1.1\"\n", testastic.IgnoreArrayOrder())
	})
}
