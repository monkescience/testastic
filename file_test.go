package testastic_test

import (
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertFile(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		// given: an expected file with exact content
		// when: asserting with matching content
		// then: no failure
		testastic.AssertFile(t, "testdata/file/exact_match.txt", "line 1\nline 2\nline 3")
	})

	t.Run("mismatch", func(t *testing.T) {
		// given: expected file and mismatched actual
		mt := &fileMockT{}

		// when: asserting with mismatched content
		testastic.AssertFile(mt, "testdata/file/mismatch.txt", "actual line")

		// then: test fails
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("with anyString matcher", func(t *testing.T) {
		// given: expected file with anyString matcher
		// when: asserting with any string value
		// then: passes (matcher accepts any string)
		testastic.AssertFile(t, "testdata/file/with_anystring.txt", "Name: John Doe\nStatus: active")
	})

	t.Run("with multiple matchers", func(t *testing.T) {
		// given: expected file with multiple matchers
		// when: asserting with matching values
		// then: passes
		testastic.AssertFile(t, "testdata/file/with_multiple_matchers.txt", "User: Alice (30 years old)")
	})

	t.Run("with regex matcher", func(t *testing.T) {
		// given: expected file with regex matcher
		// when: asserting with matching email
		// then: passes
		testastic.AssertFile(t, "testdata/file/with_regex.txt", "Email: alice@example.com")
	})

	t.Run("with bytes", func(t *testing.T) {
		// given: expected file
		// when: asserting with []byte input
		// then: passes
		testastic.AssertFile(t, "testdata/file/bytes.txt", []byte("hello world"))
	})

	t.Run("matcher fails", func(t *testing.T) {
		// given: expected file with int matcher
		mt := &fileMockT{}

		// when: asserting with non-matching value
		testastic.AssertFile(mt, "testdata/file/matcher_fails.txt", "Count: not-a-number")

		// then: test fails
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("with message", func(t *testing.T) {
		// given: expected file and mismatched actual
		mt := &fileMockT{}

		// when: asserting with custom message
		testastic.AssertFile(mt, "testdata/file/with_message.txt", "actual", testastic.Message("custom error message"))

		// then: failure includes custom message
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("empty files", func(t *testing.T) {
		// given: empty expected file
		// when: asserting with empty actual
		// then: passes
		testastic.AssertFile(t, "testdata/file/empty.txt", "")
	})

	t.Run("empty expected non-empty actual", func(t *testing.T) {
		// given: empty expected, non-empty actual
		mt := &fileMockT{}

		// when: asserting
		testastic.AssertFile(mt, "testdata/file/empty.txt", "some content")

		// then: test fails (extra content)
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("special chars", func(t *testing.T) {
		// given: file with special regex characters
		// when: asserting with matching value
		// then: passes (special chars escaped properly)
		testastic.AssertFile(t, "testdata/file/special_chars.txt", "Price: $100.99 (USD)")
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
