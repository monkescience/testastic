package testastic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertYAML(t *testing.T) {
	t.Parallel()

	t.Run("matches multiple YAML documents", func(t *testing.T) {
		t.Parallel()

		// given: an expected stream containing two YAML documents
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("name: first\n---\nname: second\nitems:\n  - a\n  - b\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting an equivalent stream with different YAML formatting
		testastic.AssertYAML(mt, expectedFile, "name: first\n---\nname: second\nitems: [a, b]\n")

		// then: every document compares successfully
		if mt.failed {
			t.Errorf("expected multi-document YAML to match, got: %s", mt.message)
		}
	})

	t.Run("detects documents after an empty document", func(t *testing.T) {
		t.Parallel()

		// given: an expected stream with an empty document before a later document
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("a: 1\n---\n---\nb: 2\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting a stream that stops before the empty and later documents
		testastic.AssertYAML(mt, expectedFile, "a: 1\n")

		// then: the later document is reported as missing
		if !mt.failed || mt.fatal {
			t.Errorf("expected a nonfatal missing-document mismatch, got: %s", mt.message)
		}

		if !strings.Contains(mt.message, "b: 2") {
			t.Errorf("expected the document after the empty document in the diff, got: %s", mt.message)
		}
	})

	t.Run("distinguishes repeated empty documents from one null document", func(t *testing.T) {
		t.Parallel()

		// given: an expected stream containing two explicit empty documents
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("---\n---\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting a stream containing only one null document
		testastic.AssertYAML(mt, expectedFile, "null\n")

		// then: the document-count difference is reported
		if !mt.failed || mt.fatal {
			t.Errorf("expected a nonfatal document-count mismatch, got: %s", mt.message)
		}
	})

	t.Run("keeps document markers inside block scalars", func(t *testing.T) {
		t.Parallel()

		// given: a block scalar containing a line that resembles a document marker
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("value: |\n  before\n  ---\n  after\n---\nname: second\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting the same scalar through quoted YAML syntax
		testastic.AssertYAML(mt, expectedFile,
			"value: \"before\\n---\\nafter\\n\"\n---\nname: second\n")

		// then: only the root-level marker separates documents
		if mt.failed {
			t.Errorf("expected the block-scalar marker to remain content, got: %s", mt.message)
		}
	})

	t.Run("detects an extra actual YAML document", func(t *testing.T) {
		t.Parallel()

		// given: an expected stream containing one YAML document
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("name: first\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting an actual stream with an additional document
		testastic.AssertYAML(mt, expectedFile, "name: first\n---\nname: extra\n")

		// then: comparison fails without treating valid YAML as a parse error
		if !mt.failed || mt.fatal {
			t.Errorf("expected a nonfatal document-count mismatch, got: %s", mt.message)
		}

		if !strings.Contains(mt.message, "extra") || !strings.Contains(mt.message, "---") {
			t.Errorf("expected the extra document in the diff, got: %s", mt.message)
		}
	})

	t.Run("detects a missing actual YAML document", func(t *testing.T) {
		t.Parallel()

		// given: an expected stream containing two YAML documents
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("name: first\n---\nname: missing\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting an actual stream with only the first document
		testastic.AssertYAML(mt, expectedFile, "name: first\n")

		// then: comparison fails without treating valid YAML as a parse error
		if !mt.failed || mt.fatal {
			t.Errorf("expected a nonfatal document-count mismatch, got: %s", mt.message)
		}

		if !strings.Contains(mt.message, "missing") || !strings.Contains(mt.message, "---") {
			t.Errorf("expected the missing document in the diff, got: %s", mt.message)
		}
	})

	t.Run("detects a change in a later YAML document", func(t *testing.T) {
		t.Parallel()

		// given: an expected stream containing two YAML documents
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("name: first\n---\nstatus: expected\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: only the second document differs
		testastic.AssertYAML(mt, expectedFile, "name: first\n---\nstatus: actual\n")

		// then: the diff retains the document separator and changed values
		if !mt.failed {
			t.Error("expected a mismatch in the second YAML document")
		}

		if !strings.Contains(mt.message, "---") ||
			!strings.Contains(mt.message, "status: expected") ||
			!strings.Contains(mt.message, "status: actual") {
			t.Errorf("expected a multi-document diff for the later change, got: %s", mt.message)
		}
	})

	t.Run("matcher in a later YAML document", func(t *testing.T) {
		t.Parallel()

		// given: a matcher in the second document of an expected stream
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("name: first\n---\nname: {{anyString}}\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: the corresponding actual value satisfies the matcher
		testastic.AssertYAML(mt, expectedFile, "name: first\n---\nname: second\n")

		// then: the complete stream matches
		if mt.failed {
			t.Errorf("expected matcher in later document to match, got: %s", mt.message)
		}
	})

	t.Run("keeps document order when array order is ignored", func(t *testing.T) {
		t.Parallel()

		// given: two documents with arrays whose element order may vary
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("kind: first\nitems: [a, b]\n---\nkind: second\nitems: [c, d]\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: document order and the arrays inside each document are reversed
		testastic.AssertYAML(mt, expectedFile,
			"kind: second\nitems: [d, c]\n---\nkind: first\nitems: [b, a]\n",
			testastic.IgnoreArrayOrder(),
		)

		// then: document order still determines the comparison
		if !mt.failed || mt.fatal {
			t.Errorf("expected a nonfatal document-order mismatch, got: %s", mt.message)
		}
	})

	t.Run("applies scoped array order within every YAML document", func(t *testing.T) {
		t.Parallel()

		// given: two documents with the same root-level array path
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("items: [a, b]\n---\nitems: [c, d]\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: ignoring array order at the documented per-document root path
		testastic.AssertYAML(mt, expectedFile,
			"items: [b, a]\n---\nitems: [d, c]\n",
			testastic.IgnoreArrayOrderAt("$.items"),
		)

		// then: the scoped option applies independently within both documents
		if mt.failed {
			t.Errorf("expected scoped array order to apply to every document, got: %s", mt.message)
		}
	})

	t.Run("preserves root array paths within every YAML document", func(t *testing.T) {
		t.Parallel()

		// given: two root arrays containing nested arrays at the same path
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("- roles: [admin, user]\n---\n- roles: [viewer, editor]\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: ignoring order through an existing root-array path
		testastic.AssertYAML(mt, expectedFile,
			"- roles: [user, admin]\n---\n- roles: [editor, viewer]\n",
			testastic.IgnoreArrayOrderAt("$[0].roles"),
		)

		// then: the path remains relative to each YAML document
		if mt.failed {
			t.Errorf("expected root array path to apply to every document, got: %s", mt.message)
		}
	})

	t.Run("renders scoped unordered arrays consistently across YAML documents", func(t *testing.T) {
		t.Parallel()

		// given: reordered arrays and an unrelated mismatch in a two-document stream
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("items: [a, b]\nstatus: expected\n---\nitems: [c, d]\nstatus: same\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: comparing with document-relative unordered array paths
		testastic.AssertYAML(mt, expectedFile,
			"items: [b, a]\nstatus: actual\n---\nitems: [d, c]\nstatus: same\n",
			testastic.IgnoreArrayOrderAt("$.items"),
		)

		// then: only the status mismatch appears in the rendered diff
		if !mt.failed {
			t.Fatal("expected a status mismatch")
		}

		if strings.Contains(mt.message, "- - a") || strings.Contains(mt.message, "- - c") ||
			strings.Contains(mt.message, "+ - b") || strings.Contains(mt.message, "+ - d") {
			t.Errorf("expected reordered arrays to be omitted from the diff, got: %s", mt.message)
		}
	})

	t.Run("ignores a scoped field within every YAML document", func(t *testing.T) {
		t.Parallel()

		// given: two documents with a changing field at the same root-relative path
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("metadata:\n  revision: first\n---\nmetadata:\n  revision: second\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: ignoring that field at the documented per-document root path
		testastic.AssertYAML(mt, expectedFile,
			"metadata:\n  revision: changed\n---\nmetadata:\n  revision: changed\n",
			testastic.IgnoreFields("$.metadata.revision"),
		)

		// then: the scoped option applies independently within both documents
		if mt.failed {
			t.Errorf("expected scoped field to be ignored in every document, got: %s", mt.message)
		}
	})

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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

	t.Run("literal matcher prefix", func(t *testing.T) {
		t.Parallel()

		// given: expected YAML containing a literal value in the matcher namespace
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("value: __TESTASTIC_YAML_MATCHER_literal\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: comparing the identical literal value
		testastic.AssertYAML(mt, expectedFile, "value: __TESTASTIC_YAML_MATCHER_literal\n")

		// then: the literal is not interpreted as an internal placeholder
		if mt.failed {
			t.Errorf("expected literal matcher prefix to compare normally, got: %s", mt.message)
		}
	})

	t.Run("literal matcher placeholder does not resolve", func(t *testing.T) {
		t.Parallel()

		// given: a literal matching the first generated placeholder and a real matcher
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile,
			[]byte("literal: __TESTASTIC_YAML_MATCHER_0__\ndynamic: {{anyString}}\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: the literal differs while the real matcher succeeds
		testastic.AssertYAML(mt, expectedFile, "literal: different\ndynamic: value\n")

		// then: the literal remains a literal and causes a mismatch
		if !mt.failed {
			t.Error("expected generated-placeholder-like literal to remain literal")
		}
	})

	t.Run("whitespace-only matcher directive is fatal", func(t *testing.T) {
		t.Parallel()

		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("value: {{ }}\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		testastic.AssertYAML(mt, expectedFile, "value: anything\n")

		if !mt.fatal {
			t.Errorf("expected whitespace-only directive to be fatal, got: %s", mt.message)
		}
	})

	t.Run("with regex matcher", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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

	t.Run("large integer mismatch", func(t *testing.T) {
		t.Parallel()

		// given: adjacent negative integers above float64's exact integer range
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		err := os.WriteFile(expectedFile, []byte("id: -9007199254740992\n"), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting with a different integer that rounds to the same float64
		testastic.AssertYAML(mt, expectedFile, "id: -9007199254740993\n")

		// then: the assertion detects the numeric mismatch
		if !mt.failed {
			t.Error("expected large integer mismatch to fail")
		}
	})

	t.Run("missing field", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		// given: an expected YAML file and mismatched actual YAML
		mt := &mockT{}
		actual := "name: wrong-name\nversion: \"1.0\"\n"

		// when: asserting with a custom message
		testastic.AssertYAML(mt, "testdata/yaml/mismatch_value.yaml", actual, testastic.Message("custom yaml message"))

		// then: the failure includes the custom message
		if !mt.failed {
			t.Error("expected failure for mismatched YAML")
		}

		if !strings.Contains(mt.message, "custom yaml message") {
			t.Errorf("expected custom message in failure, got: %s", mt.message)
		}
	})

	t.Run("create expected file", func(t *testing.T) {
		t.Parallel()

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

	t.Run("create multi-document expected file", func(t *testing.T) {
		t.Parallel()

		// given: a missing expected file and an actual stream with two documents
		expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
		actual := "name: first\n---\nname: second\n"
		mt := &mockT{}

		// when: creating the expected file in update mode
		testastic.AssertYAML(mt, expectedFile, actual, testastic.Update())

		// then: every document and its separator are written
		if mt.failed {
			t.Errorf("expected multi-document file creation to succeed, got: %s", mt.message)
		}

		content, err := os.ReadFile(expectedFile)
		testastic.NoError(t, err)

		if !strings.Contains(string(content), "name: first") ||
			!strings.Contains(string(content), "name: second") ||
			strings.Count(string(content), "---") != 1 {
			t.Errorf("expected both YAML documents to be created, got:\n%s", content)
		}

		check := &mockT{}
		testastic.AssertYAML(check, expectedFile, actual)

		if check.failed {
			t.Errorf("expected created stream to compare successfully, got: %s", check.message)
		}
	})

	t.Run("create expected file in nested directory", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

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
	t.Parallel()

	t.Run("update roundtrip preserves unquoted template expressions", func(t *testing.T) {
		t.Parallel()

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

func TestAssertYAML_UpdatePreservesMatchersAcrossDocuments(t *testing.T) {
	t.Parallel()

	// given: matchers in both documents of an expected stream
	expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
	expected := "name: {{anyString}}\nversion: old\n---\nname: {{anyString}}\nreplicas: 1\n"
	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	testastic.NoError(t, err)

	actual := "name: config\nversion: new\n---\nname: deployment\nreplicas: 2\n"
	mt := &mockT{}

	// when: updating changed nonmatcher values across the stream
	testastic.AssertYAML(mt, expectedFile, actual, testastic.Update())

	// then: both documents update while their matcher positions remain intact
	if mt.failed {
		t.Errorf("expected multi-document update to succeed, got: %s", mt.message)
	}

	content, readErr := os.ReadFile(expectedFile)
	testastic.NoError(t, readErr)

	if strings.Count(string(content), "{{anyString}}") != 2 ||
		strings.Count(string(content), "---") != 1 ||
		!strings.Contains(string(content), "version: new") ||
		!strings.Contains(string(content), "replicas: 2") {
		t.Errorf("expected updated documents with preserved matchers, got:\n%s", content)
	}

	check := &mockT{}
	testastic.AssertYAML(check, expectedFile, actual)

	if check.failed {
		t.Errorf("expected updated stream to match, got: %s", check.message)
	}
}

func TestAssertYAML_UpdatePreservesMatchersInUnorderedArraysAcrossDocuments(t *testing.T) {
	t.Parallel()

	// given: a matcher in an unordered array in the second document
	expectedFile := filepath.Join(t.TempDir(), "expected.yaml")
	expected := "kind: first\n---\nitems: [a, {{anyInt}}]\nstatus: old\n"
	err := os.WriteFile(expectedFile, []byte(expected), 0o644)
	testastic.NoError(t, err)

	actual := "kind: first\n---\nitems: [1, a]\nstatus: new\n"
	mt := &mockT{}

	// when: updating after the array elements move and another field changes
	testastic.AssertYAML(mt, expectedFile, actual,
		testastic.IgnoreArrayOrderAt("$.items"), testastic.Update())

	// then: the matcher follows its matched value and the updated stream still matches
	if mt.failed {
		t.Errorf("expected multi-document update to succeed, got: %s", mt.message)
	}

	content, readErr := os.ReadFile(expectedFile)
	testastic.NoError(t, readErr)

	if !strings.Contains(string(content), "{{anyInt}}") || !strings.Contains(string(content), "- a") {
		t.Errorf("expected the matcher and literal array value to be preserved, got:\n%s", content)
	}

	check := &mockT{}
	testastic.AssertYAML(check, expectedFile, actual, testastic.IgnoreArrayOrderAt("$.items"))

	if check.failed {
		t.Errorf("expected updated unordered stream to match, got: %s", check.message)
	}
}

func TestAssertYAML_UnsupportedOptions(t *testing.T) {
	t.Parallel()

	t.Run("html options rejected", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		// given: supported options for AssertYAML
		// when: asserting with IgnoreArrayOrder
		// then: the test passes without unsupported option error
		testastic.AssertYAML(t, "testdata/yaml/exact_match_supported_options.yaml",
			"name: test-supported\nversion: \"1.1\"\n", testastic.IgnoreArrayOrder())
	})
}

func TestAssertYAML_IdenticalInfinityMatches(t *testing.T) {
	t.Parallel()

	// given: an expected file and actual that both carry the same infinity
	expectedFile := filepath.Join(t.TempDir(), "inf.yaml")

	err := os.WriteFile(expectedFile, []byte("value: .inf\n"), 0o644)
	if err != nil {
		t.Fatalf("write expected file: %v", err)
	}

	mt := &mockT{}

	// when: comparing identical infinities
	testastic.AssertYAML(mt, expectedFile, "value: .inf\n")

	// then: they compare equal rather than as a type mismatch
	if mt.failed {
		t.Errorf("expected identical +Inf to match, got: %s", mt.message)
	}
}

func TestAssertYAML_BinaryMatchesByteSlice(t *testing.T) {
	t.Parallel()

	// given: an expected file using a !!binary scalar
	expectedFile := filepath.Join(t.TempDir(), "binary.yaml")

	err := os.WriteFile(expectedFile, []byte("b: !!binary aGVsbG8=\n"), 0o644)
	if err != nil {
		t.Fatalf("write expected file: %v", err)
	}

	mt := &mockT{}
	actual := struct {
		B []byte `yaml:"b"`
	}{B: []byte("hello")}

	// when: the actual carries the equivalent Go []byte
	testastic.AssertYAML(mt, expectedFile, actual)

	// then: they compare equal
	if mt.failed {
		t.Errorf("expected !!binary to match an equivalent []byte, got: %s", mt.message)
	}
}

func TestAssertYAML_MatchedUnorderedArrayNotShownAsDiff(t *testing.T) {
	t.Parallel()

	// given: an unordered array that needs cross-index matcher correspondence
	expectedFile := filepath.Join(t.TempDir(), "matcher.yaml")

	err := os.WriteFile(expectedFile,
		[]byte("items:\n  - a\n  - {{anyString}}\nstatus: expected\n"), 0o644)
	if err != nil {
		t.Fatalf("write expected file: %v", err)
	}

	mt := &mockT{}

	// when: only a field outside the successfully matched array differs
	testastic.AssertYAML(mt, expectedFile, "items:\n  - z\n  - a\nstatus: actual\n",
		testastic.IgnoreArrayOrderAt("$.items"))

	// then: the failure does not render the unordered array as changed
	if !mt.failed {
		t.Fatal("expected a difference for the status field")
	}

	if strings.Contains(mt.message, "- - a") || strings.Contains(mt.message, "+ - z") {
		t.Errorf("matched unordered array should not appear changed in the diff, got: %s", mt.message)
	}
}
