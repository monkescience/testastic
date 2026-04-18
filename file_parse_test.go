//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"testing"
)

func TestParseLine(t *testing.T) {
	t.Parallel()

	t.Run("no matchers", func(t *testing.T) {
		t.Parallel()

		// given: a plain line without matchers
		line := "Hello, World!"

		// when: parsing the line
		result, err := parseLine(line)
		// then: no error, pattern is nil (exact match), original preserved
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.pattern != nil {
			t.Error("expected nil pattern for line without matchers")
		}

		if result.original != line {
			t.Errorf("expected original %q, got %q", line, result.original)
		}

		if len(result.matchers) != 0 {
			t.Errorf("expected 0 matchers, got %d", len(result.matchers))
		}
	})

	t.Run("single matcher", func(t *testing.T) {
		t.Parallel()

		// given: a line with a single anyString matcher
		line := "Name: {{anyString}}"

		// when: parsing the line
		result, err := parseLine(line)
		// then: pattern is set, one matcher extracted
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.pattern == nil {
			t.Fatal("expected non-nil pattern for line with matcher")
		}

		if result.original != line {
			t.Errorf("expected original %q, got %q", line, result.original)
		}

		if len(result.matchers) != 1 {
			t.Fatalf("expected 1 matcher, got %d", len(result.matchers))
		}

		if result.matchers[0].String() != anyStringMatcherValue {
			t.Errorf("expected {{anyString}} matcher, got %s", result.matchers[0].String())
		}

		// Verify pattern matches expected format
		if !result.pattern.MatchString("Name: John Doe") {
			t.Error("pattern should match 'Name: John Doe'")
		}

		if result.pattern.MatchString("Email: test@example.com") {
			t.Error("pattern should not match 'Email: test@example.com'")
		}
	})

	t.Run("multiple matchers", func(t *testing.T) {
		t.Parallel()

		// given: a line with multiple matchers
		line := "User {{anyString}} is {{anyInt}} years old"

		// when: parsing the line
		result, err := parseLine(line)
		// then: pattern matches, both matchers extracted
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.pattern == nil {
			t.Fatal("expected non-nil pattern")
		}

		if len(result.matchers) != 2 {
			t.Fatalf("expected 2 matchers, got %d", len(result.matchers))
		}

		if result.matchers[0].String() != anyStringMatcherValue {
			t.Errorf("expected first matcher {{anyString}}, got %s", result.matchers[0].String())
		}

		if result.matchers[1].String() != "{{anyInt}}" {
			t.Errorf("expected second matcher {{anyInt}}, got %s", result.matchers[1].String())
		}

		// Verify pattern matches
		if !result.pattern.MatchString("User Alice is 30 years old") {
			t.Error("pattern should match 'User Alice is 30 years old'")
		}

		if !result.pattern.MatchString("User Bob Smith is 25 years old") {
			t.Error("pattern should match 'User Bob Smith is 25 years old'")
		}
	})

	t.Run("regex matcher", func(t *testing.T) {
		t.Parallel()

		// given: a line with a regex matcher (no anchors - they're added by parseLine)
		line := "Email: {{regex `[a-z]+@example\\.com`}}"

		// when: parsing the line
		result, err := parseLine(line)
		// then: pattern uses the regex
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.pattern == nil {
			t.Fatal("expected non-nil pattern")
		}

		if !result.pattern.MatchString("Email: alice@example.com") {
			t.Error("pattern should match 'Email: alice@example.com'")
		}

		if result.pattern.MatchString("Email: ALICE@example.com") {
			t.Error("pattern should not match 'Email: ALICE@example.com'")
		}
	})

	t.Run("special regex chars", func(t *testing.T) {
		t.Parallel()

		// given: a line with regex special characters in literal text
		line := "Price: ${{anyInt}}.00 (USD)"

		// when: parsing the line
		result, err := parseLine(line)
		// then: special chars are escaped in pattern
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.pattern == nil {
			t.Fatal("expected non-nil pattern")
		}

		if !result.pattern.MatchString("Price: $99.00 (USD)") {
			t.Error("pattern should match 'Price: $99.00 (USD)'")
		}

		if result.pattern.MatchString("Price: 99.00 USD") {
			t.Error("pattern should not match 'Price: 99.00 USD'")
		}
	})
}
