package testastic_test

import (
	"encoding/json"
	"testing"

	"github.com/monkescience/testastic"
)

func TestMatchers(t *testing.T) {
	t.Parallel()

	t.Run("AnyString", func(t *testing.T) {
		t.Parallel()

		// given: an AnyString matcher
		m := testastic.AnyString()

		// when: matching against a string
		// then: it matches
		if !m.Match("hello") {
			t.Error("expected to match string")
		}

		// when: matching against an int
		// then: it does not match
		if m.Match(123) {
			t.Error("expected not to match int")
		}
	})

	t.Run("AnyInt", func(t *testing.T) {
		t.Parallel()

		// given: an AnyInt matcher
		m := testastic.AnyInt()

		// when: matching against an integer float64
		// then: it matches
		if !m.Match(float64(42)) {
			t.Error("expected to match integer float64")
		}

		// when: matching against a non-integer float
		// then: it does not match
		if m.Match(42.5) {
			t.Error("expected not to match non-integer float")
		}

		// when: matching against a string
		// then: it does not match
		if m.Match("42") {
			t.Error("expected not to match string")
		}
	})

	t.Run("AnyFloat", func(t *testing.T) {
		t.Parallel()

		// given: an AnyFloat matcher
		m := testastic.AnyFloat()

		// when: matching against a float
		// then: it matches
		if !m.Match(float64(42.5)) {
			t.Error("expected to match float")
		}

		// when: matching against an integer (as float64)
		// then: it also matches
		if !m.Match(float64(42)) {
			t.Error("expected to match integer")
		}
	})

	t.Run("AnyBool", func(t *testing.T) {
		t.Parallel()

		// given: an AnyBool matcher
		m := testastic.AnyBool()

		// when: matching against a bool
		// then: it matches
		if !m.Match(true) {
			t.Error("expected to match bool")
		}

		// when: matching against a string "true"
		// then: it does not match
		if m.Match("true") {
			t.Error("expected not to match string")
		}
	})

	t.Run("AnyValue", func(t *testing.T) {
		t.Parallel()

		// given: an AnyValue matcher
		m := testastic.AnyValue()

		// when: matching against any type
		// then: it always matches
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

	t.Run("Ignore", func(t *testing.T) {
		t.Parallel()

		// given: an Ignore matcher
		m := testastic.Ignore()

		// when: matching against arbitrary values
		// then: it always matches and reports the ignore token
		if !m.Match("anything") {
			t.Error("expected Ignore matcher to match strings")
		}

		if !m.Match(123) {
			t.Error("expected Ignore matcher to match integers")
		}

		if m.String() != "{{ignore}}" {
			t.Errorf("String() = %v, want {{ignore}}", m.String())
		}
	})

	t.Run("Regex", func(t *testing.T) {
		t.Parallel()

		// given: a Regex matcher for date format
		m, err := testastic.Regex(`^\d{4}-\d{2}-\d{2}$`)
		if err != nil {
			t.Fatal(err)
		}

		// when: matching against a valid date string
		// then: it matches
		if !m.Match("2024-01-15") {
			t.Error("expected to match date format")
		}

		// when: matching against an invalid format
		// then: it does not match
		if m.Match("invalid") {
			t.Error("expected not to match invalid format")
		}
	})

	t.Run("OneOf", func(t *testing.T) {
		t.Parallel()

		// given: a OneOf matcher with allowed values
		m := testastic.OneOf("a", "b", "c")

		// when: matching against an allowed value
		// then: it matches
		if !m.Match("a") {
			t.Error("expected to match 'a'")
		}

		// when: matching against a non-allowed value
		// then: it does not match
		if m.Match("d") {
			t.Error("expected not to match 'd'")
		}
	})

	t.Run("AnyUUID", func(t *testing.T) {
		t.Parallel()

		// given: an AnyUUID matcher
		m := testastic.AnyUUID()

		// when: matching against valid UUIDs
		// then: they match
		if !m.Match("550e8400-e29b-41d4-a716-446655440000") {
			t.Error("should match valid UUID")
		}

		if !m.Match("6ba7b810-9dad-11d1-80b4-00c04fd430c8") {
			t.Error("should match valid UUID")
		}

		// when: matching against invalid UUID
		// then: it does not match
		if m.Match("not-a-uuid") {
			t.Error("should not match invalid UUID")
		}

		// when: matching against non-string
		// then: it does not match
		if m.Match(123) {
			t.Error("should not match non-string")
		}

		// then: String() returns expected format
		if m.String() != "{{anyUUID}}" {
			t.Errorf("String() = %v, want {{anyUUID}}", m.String())
		}
	})

	t.Run("AnyDateTime", func(t *testing.T) {
		t.Parallel()

		// given: an AnyDateTime matcher
		m := testastic.AnyDateTime()

		// when: matching against valid date/time formats
		// then: they match
		if !m.Match("2024-01-15") {
			t.Error("should match date")
		}

		if !m.Match("2024-01-15T10:30:00Z") {
			t.Error("should match ISO 8601 datetime")
		}

		if !m.Match("2024-01-15 10:30:00") {
			t.Error("should match datetime with space")
		}

		// when: matching against invalid date
		// then: it does not match
		if m.Match("not-a-date") {
			t.Error("should not match invalid date")
		}

		// when: matching against non-string
		// then: it does not match
		if m.Match(123) {
			t.Error("should not match non-string")
		}

		// then: String() returns expected format
		if m.String() != "{{anyDateTime}}" {
			t.Errorf("String() = %v, want {{anyDateTime}}", m.String())
		}
	})

	t.Run("AnyURL", func(t *testing.T) {
		t.Parallel()

		// given: an AnyURL matcher
		m := testastic.AnyURL()

		// when: matching against valid URLs
		// then: they match
		if !m.Match("https://example.com") {
			t.Error("should match HTTPS URL")
		}

		if !m.Match("http://example.com/path") {
			t.Error("should match HTTP URL with path")
		}

		// when: matching against invalid URL
		// then: it does not match
		if m.Match("not-a-url") {
			t.Error("should not match invalid URL")
		}

		// when: matching against non-string
		// then: it does not match
		if m.Match(123) {
			t.Error("should not match non-string")
		}

		// then: String() returns expected format
		if m.String() != "{{anyURL}}" {
			t.Errorf("String() = %v, want {{anyURL}}", m.String())
		}
	})
}

func TestRegisterMatcher(t *testing.T) {
	t.Run("custom matcher works through public API", func(t *testing.T) {
		// given: a custom matcher registered in the registry
		testastic.RegisterMatcher("customTest", func(args string) (testastic.Matcher, error) {
			return testastic.AnyString(), nil
		})

		// when: asserting JSON with the custom matcher
		// then: the custom matcher is used and matches any string
		testastic.AssertJSON(t, "testdata/json/custom_matcher.json", `{"value": "test"}`)
	})
}

func TestOneOfMatcher_StringRoundTrips(t *testing.T) {
	t.Parallel()

	// given: a oneOf matcher with string values
	m := testastic.OneOf("pending", "active")

	// when: rendering it back to template syntax
	got := m.String()

	// then: each value is individually quoted so it re-parses as valid syntax
	want := `{{oneOf "pending" "active"}}`
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestOneOfMatcher_MatchesNumericJSONValues(t *testing.T) {
	t.Parallel()

	// given: JSON numbers decode as json.Number
	actual := json.Number("200")

	// when/then: string candidates match a numeric actual
	if !testastic.OneOf("200", "404").Match(actual) {
		t.Error("expected oneOf string candidates to match json.Number(200)")
	}

	// when/then: numeric candidates also match
	if !testastic.OneOf(200, 404).Match(actual) {
		t.Error("expected oneOf numeric candidates to match json.Number(200)")
	}

	// when/then: a non-member numeric does not match
	if testastic.OneOf("200", "404").Match(json.Number("500")) {
		t.Error("expected json.Number(500) not to match")
	}
}

func TestOneOfMatcher_DoesNotPanicOnUncomparableValue(t *testing.T) {
	t.Parallel()

	// given: a map actual (uncomparable with ==)
	actual := map[string]any{"a": 1}

	// when: matching an equal map candidate
	// then: it matches via deep equality without panicking
	if !testastic.OneOf(map[string]any{"a": 1}).Match(actual) {
		t.Error("expected deep-equal map to match")
	}

	// when: matching a string candidate against a map actual
	// then: it returns false without panicking
	if testastic.OneOf("a").Match(actual) {
		t.Error("expected string candidate not to match map actual")
	}
}

func TestRegexMatcher_StringRoundTripsWithBacktick(t *testing.T) {
	t.Parallel()

	// given: a regex whose pattern contains a backtick
	m, err := testastic.Regex("a`b")
	if err != nil {
		t.Fatal(err)
	}

	// when: rendering it back to template syntax
	got := m.String()

	// then: it falls back to the double-quoted form so the backtick is representable
	want := "{{regex \"a`b\"}}"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
