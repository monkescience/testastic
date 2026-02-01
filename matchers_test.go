package testastic_test

import (
	"testing"

	"github.com/monkescience/testastic"
)

func TestMatchers(t *testing.T) {
	t.Run("AnyString", func(t *testing.T) {
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

	t.Run("Regex", func(t *testing.T) {
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
