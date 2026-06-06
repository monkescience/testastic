package testastic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertFile(t *testing.T) {
	t.Parallel()

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()

		// given: an expected file with exact content
		// when: asserting with matching content
		// then: no failure
		testastic.AssertFile(t, "testdata/file/exact_match.txt", "line 1\nline 2\nline 3")
	})

	t.Run("mismatch", func(t *testing.T) {
		t.Parallel()

		// given: expected file and mismatched actual
		mt := &mockT{}

		// when: asserting with mismatched content
		testastic.AssertFile(mt, "testdata/file/mismatch.txt", "actual line")

		// then: test fails
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("with anyString matcher", func(t *testing.T) {
		t.Parallel()

		// given: expected file with anyString matcher
		// when: asserting with any string value
		// then: passes (matcher accepts any string)
		testastic.AssertFile(t, "testdata/file/with_anystring.txt", "Name: John Doe\nStatus: active")
	})

	t.Run("with multiple matchers", func(t *testing.T) {
		t.Parallel()

		// given: expected file with multiple matchers
		// when: asserting with matching values
		// then: passes
		testastic.AssertFile(t, "testdata/file/with_multiple_matchers.txt", "User: Alice (30 years old)")
	})

	t.Run("with regex matcher", func(t *testing.T) {
		t.Parallel()

		// given: expected file with regex matcher
		// when: asserting with matching email
		// then: passes
		testastic.AssertFile(t, "testdata/file/with_regex.txt", "Email: alice@example.com")
	})

	t.Run("with bytes", func(t *testing.T) {
		t.Parallel()

		// given: expected file
		// when: asserting with []byte input
		// then: passes
		testastic.AssertFile(t, "testdata/file/bytes.txt", []byte("hello world"))
	})

	t.Run("with reader", func(t *testing.T) {
		t.Parallel()

		// given: an expected file
		// when: asserting with io.Reader input
		// then: the assertion passes
		testastic.AssertFile(t, "testdata/file/bytes.txt", strings.NewReader("hello world"))
	})

	t.Run("matcher fails", func(t *testing.T) {
		t.Parallel()

		// given: expected file with int matcher
		mt := &mockT{}

		// when: asserting with non-matching value
		testastic.AssertFile(mt, "testdata/file/matcher_fails.txt", "Count: not-a-number")

		// then: test fails
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		// given: expected file and mismatched actual
		mt := &mockT{}

		// when: asserting with custom message
		testastic.AssertFile(mt, "testdata/file/with_message.txt", "actual", testastic.Message("custom error message"))

		// then: failure includes custom message
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("empty files", func(t *testing.T) {
		t.Parallel()

		// given: empty expected file
		// when: asserting with empty actual
		// then: passes
		testastic.AssertFile(t, "testdata/file/empty.txt", "")
	})

	t.Run("non-empty expected non-empty actual", func(t *testing.T) {
		t.Parallel()

		// given: non-empty expected and mismatched actual
		mt := &mockT{}

		// when: asserting
		testastic.AssertFile(mt, "testdata/file/empty_non_empty_actual.txt", "some content")

		// then: test fails (content mismatch)
		if !mt.failed {
			t.Error("expected test to fail")
		}
	})

	t.Run("special chars", func(t *testing.T) {
		t.Parallel()

		// given: file with special regex characters
		// when: asserting with matching value
		// then: passes (special chars escaped properly)
		testastic.AssertFile(t, "testdata/file/special_chars.txt", "Price: $100.99 (USD)")
	})

	t.Run("diff output does not flag matcher lines that already matched", func(t *testing.T) {
		t.Parallel()

		// given: an expected file with a matcher line and a later mismatched line
		expectedFile := filepath.Join(t.TempDir(), "matcher_diff.txt")

		err := os.WriteFile(expectedFile, []byte("Version: {{regex `v[0-9]+`}}\nStatus: ready"), 0o644)
		if err != nil {
			t.Fatalf("write expected file: %v", err)
		}

		mt := &mockT{}

		// when: asserting with a matching version and mismatched status
		testastic.AssertFile(mt, expectedFile, "Version: v1\nStatus: failed")

		// then: the failure message reports only the real mismatched line
		if !mt.failed {
			t.Fatal("expected test to fail")
		}

		if strings.Contains(mt.message, "- Version: {{regex `v[0-9]+`}}") {
			t.Fatalf("expected matcher line to be omitted from diff, got: %s", mt.message)
		}

		if strings.Contains(mt.message, "+ Version: v1") {
			t.Fatalf("expected matched actual line to be omitted from diff, got: %s", mt.message)
		}

		if !strings.Contains(mt.message, "- Status: ready") {
			t.Fatalf("expected status removal in diff, got: %s", mt.message)
		}

		if !strings.Contains(mt.message, "+ Status: failed") {
			t.Fatalf("expected status addition in diff, got: %s", mt.message)
		}
	})
}

func TestAssertFile_CreateExpectedFile(t *testing.T) {
	t.Parallel()

	t.Run("creates expected file for plain text in update mode", func(t *testing.T) {
		t.Parallel()

		// given: a non-existent expected file and plain text actual content
		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "new-expected.txt")
		actual := "Hello, World!\nThis is plain text.\n"

		mt := &mockT{}

		// when: asserting with Update option (file does not exist yet)
		testastic.AssertFile(mt, expectedFile, actual, testastic.Update())

		// then: the file is created without errors
		if mt.failed {
			t.Errorf("expected no failure when creating file, got: %s", mt.message)
		}

		content, err := os.ReadFile(expectedFile)
		if err != nil {
			t.Fatalf("expected file was not created: %v", err)
		}

		if string(content) != actual {
			t.Errorf("expected file content %q, got %q", actual, string(content))
		}
	})
}

func TestAssertFile_UnsupportedOptions(t *testing.T) {
	t.Parallel()

	t.Run("structured data and html options rejected", func(t *testing.T) {
		t.Parallel()

		// given: JSON/YAML and HTML options passed to AssertFile
		mt := &mockT{}

		// when: asserting with unsupported options
		testastic.AssertFile(mt, "testdata/file/exact_match_unsupported_options.txt", "row 1\nrow 2\nrow 3",
			testastic.IgnoreArrayOrder(), testastic.IgnoreHTMLComments())

		// then: the test fatals with a message listing both unsupported options
		if !mt.fatal {
			t.Error("expected fatal error for unsupported options")
		}

		expectedMsg := "testastic: unsupported options for AssertFile: IgnoreArrayOrder, IgnoreHTMLComments"
		if mt.message != expectedMsg {
			t.Errorf("expected message %q, got: %q", expectedMsg, mt.message)
		}
	})

	t.Run("supported options accepted", func(t *testing.T) {
		t.Parallel()

		// given: supported options for AssertFile
		// when: asserting with Message option
		// then: the test passes without unsupported option error
		testastic.AssertFile(t, "testdata/file/exact_match_supported_options.txt", "line a\nline b\nline c\n",
			testastic.Message("custom message"))
	})
}

type auditOrderIDMatcher struct{}

func (auditOrderIDMatcher) Match(actual any) bool {
	s, ok := actual.(string)

	return ok && strings.HasPrefix(s, "ORD-")
}

func (auditOrderIDMatcher) String() string { return "{{auditOrderID}}" }

func writeFileExpected(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "expected.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write expected file: %v", err)
	}

	return path
}

func TestAssertFile_CustomMatcherActuallyValidates(t *testing.T) {
	t.Parallel()

	testastic.RegisterMatcher("auditOrderID", func(string) (testastic.Matcher, error) {
		return auditOrderIDMatcher{}, nil
	})

	expectedFile := writeFileExpected(t, "id: {{auditOrderID}}")

	t.Run("rejects a value the custom matcher rejects", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}

		testastic.AssertFile(mt, expectedFile, "id: total-garbage")

		if !mt.failed {
			t.Error("expected custom matcher to reject 'total-garbage'")
		}
	})

	t.Run("accepts a value the custom matcher accepts", func(t *testing.T) {
		t.Parallel()

		mt := &mockT{}

		testastic.AssertFile(mt, expectedFile, "id: ORD-12345")

		if mt.failed {
			t.Errorf("expected custom matcher to accept 'ORD-12345', got: %s", mt.message)
		}
	})
}

func TestAssertFile_AnchoredRegexMatches(t *testing.T) {
	t.Parallel()

	// given: a regex matcher whose pattern is already anchored
	expectedFile := writeFileExpected(t, "Val: {{regex `^abc$`}}")

	mt := &mockT{}

	// when: comparing against the matching value
	testastic.AssertFile(mt, expectedFile, "Val: abc")

	// then: the user anchors do not make the line pattern unsatisfiable
	if mt.failed {
		t.Errorf("expected anchored regex to match, got: %s", mt.message)
	}
}

func TestAssertFile_QuotedRegexWithBraceQuantifier(t *testing.T) {
	t.Parallel()

	// given: a double-quoted regex arg containing a {n} quantifier
	expectedFile := writeFileExpected(t, "Code: {{regex \"\\d{3}\"}}")

	mt := &mockT{}

	// when: comparing against three digits
	testastic.AssertFile(mt, expectedFile, "Code: 123")

	// then: the directive parses and matches instead of being treated as literal
	if mt.failed {
		t.Errorf("expected quoted regex with brace quantifier to match, got: %s", mt.message)
	}
}

func TestAssertFile_WhitespaceOnlyDirectiveIsLiteral(t *testing.T) {
	t.Parallel()

	// given: a line that literally contains empty braces
	line := "Template uses {{ }} for interpolation"
	expectedFile := writeFileExpected(t, line)

	mt := &mockT{}

	// when: comparing against the identical line
	testastic.AssertFile(mt, expectedFile, line)

	// then: the empty braces are treated as literal text, not a failed matcher
	if mt.failed {
		t.Errorf("expected empty braces to compare literally, got: %s", mt.message)
	}
}

func TestAssertFile_InvalidMatcherIsFatalWithMessage(t *testing.T) {
	t.Parallel()

	// given: an expected file with an invalid regex matcher
	expectedFile := writeFileExpected(t, "X: {{regex `[unclosed`}}")

	mt := &mockT{}

	// when: asserting against it
	testastic.AssertFile(mt, expectedFile, "X: anything")

	// then: it is a fatal setup error and the parse message is surfaced
	if !mt.fatal {
		t.Error("expected an invalid matcher in the expected file to be fatal")
	}

	if !strings.Contains(mt.message, "regex") {
		t.Errorf("expected the parse error message to be surfaced, got: %s", mt.message)
	}
}

func TestAssertFile_ToleratesSingleTrailingNewline(t *testing.T) {
	t.Parallel()

	// given: an expected file with a trailing newline (as editors add)
	expectedFile := writeFileExpected(t, "line1\n")

	mt := &mockT{}

	// when: the actual has no trailing newline
	testastic.AssertFile(mt, expectedFile, "line1")

	// then: the single trailing-newline difference is tolerated
	if mt.failed {
		t.Errorf("expected a single trailing newline to be tolerated, got: %s", mt.message)
	}
}
