package testastic_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertFile_ExactMatch(t *testing.T) {
	// given: an expected file with exact content
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	content := "line 1\nline 2\nline 3"
	if err := os.WriteFile(expectedFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: asserting with matching content
	// then: no failure
	testastic.AssertFile(t, expectedFile, content)
}

func TestAssertFile_Mismatch(t *testing.T) {
	// given: expected file and mismatched actual
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	if err := os.WriteFile(expectedFile, []byte("expected line"), 0o644); err != nil {
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

func TestAssertFile_WithAnyStringMatcher(t *testing.T) {
	// given: expected file with anyString matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	content := "Name: {{anyString}}\nStatus: active"
	if err := os.WriteFile(expectedFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: asserting with any string value
	actual := "Name: John Doe\nStatus: active"

	// then: passes (matcher accepts any string)
	testastic.AssertFile(t, expectedFile, actual)
}

func TestAssertFile_WithMultipleMatchers(t *testing.T) {
	// given: expected file with multiple matchers
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	content := "User: {{anyString}} ({{anyInt}} years old)"
	if err := os.WriteFile(expectedFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: asserting with matching values
	actual := "User: Alice (30 years old)"

	// then: passes
	testastic.AssertFile(t, expectedFile, actual)
}

func TestAssertFile_WithRegexMatcher(t *testing.T) {
	// given: expected file with regex matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	content := "Email: {{regex `[a-z]+@example\\.com`}}"
	if err := os.WriteFile(expectedFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: asserting with matching email
	actual := "Email: alice@example.com"

	// then: passes
	testastic.AssertFile(t, expectedFile, actual)
}

func TestAssertFile_WithBytes(t *testing.T) {
	// given: expected file
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	content := "hello world"
	if err := os.WriteFile(expectedFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: asserting with []byte input
	// then: passes
	testastic.AssertFile(t, expectedFile, []byte(content))
}

func TestAssertFile_MatcherFails(t *testing.T) {
	// given: expected file with int matcher
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	content := "Count: {{anyInt}}"
	if err := os.WriteFile(expectedFile, []byte(content), 0o644); err != nil {
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
}

func TestAssertFile_WithMessage(t *testing.T) {
	// given: expected file and mismatched actual
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	if err := os.WriteFile(expectedFile, []byte("expected"), 0o644); err != nil {
		t.Fatal(err)
	}

	mt := &fileMockT{}

	// when: asserting with custom message
	testastic.AssertFile(mt, expectedFile, "actual", testastic.Message("custom error message"))

	// then: failure includes custom message
	if !mt.failed {
		t.Error("expected test to fail")
	}
}

func TestAssertFile_EmptyFiles(t *testing.T) {
	// given: empty expected file
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	if err := os.WriteFile(expectedFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: asserting with empty actual
	// then: passes
	testastic.AssertFile(t, expectedFile, "")
}

func TestAssertFile_EmptyExpectedNonEmptyActual(t *testing.T) {
	// given: empty expected, non-empty actual
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	if err := os.WriteFile(expectedFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	mt := &fileMockT{}

	// when: asserting
	testastic.AssertFile(mt, expectedFile, "some content")

	// then: test fails (extra content)
	if !mt.failed {
		t.Error("expected test to fail")
	}
}

func TestAssertFile_SpecialChars(t *testing.T) {
	// given: file with special regex characters
	dir := t.TempDir()
	expectedFile := filepath.Join(dir, "expected.txt")
	content := "Price: ${{anyInt}}.99 (USD)"
	if err := os.WriteFile(expectedFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// when: asserting with matching value
	actual := "Price: $100.99 (USD)"

	// then: passes (special chars escaped properly)
	testastic.AssertFile(t, expectedFile, actual)
}
