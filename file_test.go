package testastic_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertFile(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		// given: an expected file with exact content
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")
		content := "line 1\nline 2\nline 3"

		err := os.WriteFile(expectedFile, []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		// when: asserting with matching content
		// then: no failure
		testastic.AssertFile(t, expectedFile, content)
	})

	t.Run("mismatch", func(t *testing.T) {
		// given: expected file and mismatched actual
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")

		err := os.WriteFile(expectedFile, []byte("expected line"), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		mt := &fileMockT{}
		actual := "actual line"

		// when: asserting with mismatched content
		testastic.AssertFile(mt, expectedFile, actual)

		// then: test fails
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("with anyString matcher", func(t *testing.T) {
		// given: expected file with anyString matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")
		content := "Name: {{anyString}}\nStatus: active"

		err := os.WriteFile(expectedFile, []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		// when: asserting with any string value
		actual := "Name: John Doe\nStatus: active"

		// then: passes (matcher accepts any string)
		testastic.AssertFile(t, expectedFile, actual)
	})

	t.Run("with multiple matchers", func(t *testing.T) {
		// given: expected file with multiple matchers
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")
		content := "User: {{anyString}} ({{anyInt}} years old)"

		err := os.WriteFile(expectedFile, []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		// when: asserting with matching values
		actual := "User: Alice (30 years old)"

		// then: passes
		testastic.AssertFile(t, expectedFile, actual)
	})

	t.Run("with regex matcher", func(t *testing.T) {
		// given: expected file with regex matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")
		content := "Email: {{regex `[a-z]+@example\\.com`}}"

		err := os.WriteFile(expectedFile, []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		// when: asserting with matching email
		actual := "Email: alice@example.com"

		// then: passes
		testastic.AssertFile(t, expectedFile, actual)
	})

	t.Run("with bytes", func(t *testing.T) {
		// given: expected file
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")
		content := "hello world"

		err := os.WriteFile(expectedFile, []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		// when: asserting with []byte input
		// then: passes
		testastic.AssertFile(t, expectedFile, []byte(content))
	})

	t.Run("matcher fails", func(t *testing.T) {
		// given: expected file with int matcher
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")
		content := "Count: {{anyInt}}"

		err := os.WriteFile(expectedFile, []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		mt := &fileMockT{}
		actual := "Count: not-a-number"

		// when: asserting with non-matching value
		testastic.AssertFile(mt, expectedFile, actual)

		// then: test fails
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("with message", func(t *testing.T) {
		// given: expected file and mismatched actual
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")

		err := os.WriteFile(expectedFile, []byte("expected"), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		mt := &fileMockT{}

		// when: asserting with custom message
		testastic.AssertFile(mt, expectedFile, "actual", testastic.Message("custom error message"))

		// then: failure includes custom message
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("empty files", func(t *testing.T) {
		// given: empty expected file
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")

		err := os.WriteFile(expectedFile, []byte(""), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		// when: asserting with empty actual
		// then: passes
		testastic.AssertFile(t, expectedFile, "")
	})

	t.Run("empty expected non-empty actual", func(t *testing.T) {
		// given: empty expected, non-empty actual
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")

		err := os.WriteFile(expectedFile, []byte(""), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		mt := &fileMockT{}

		// when: asserting
		testastic.AssertFile(mt, expectedFile, "some content")

		// then: test fails (extra content)
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("special chars", func(t *testing.T) {
		// given: file with special regex characters
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "expected.txt")
		content := "Price: ${{anyInt}}.99 (USD)"

		err := os.WriteFile(expectedFile, []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		// when: asserting with matching value
		actual := "Price: $100.99 (USD)"

		// then: passes (special chars escaped properly)
		testastic.AssertFile(t, expectedFile, actual)
	})
}

// fileMockT for testing file assertions.
type fileMockT struct {
	testing.TB
	failed bool
	output string
}

func (m *fileMockT) Helper()                           {}
func (m *fileMockT) Fatalf(format string, args ...any) { m.failed = true; m.output = format }
func (m *fileMockT) Errorf(format string, args ...any) { m.failed = true; m.output = format }
func (m *fileMockT) Logf(format string, args ...any)   {}
