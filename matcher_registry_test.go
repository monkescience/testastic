package testastic_test

import (
	"path/filepath"
	"testing"

	"github.com/monkescience/testastic"
)

func TestNewMatchers(t *testing.T) {
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

func TestCustomMatcherRegistry(t *testing.T) {
	t.Run("custom matcher works through public API", func(t *testing.T) {
		// given: a custom matcher registered in the registry
		testastic.RegisterMatcher("customTest", func(args string) (testastic.Matcher, error) {
			return testastic.AnyString(), nil
		})

		dir := t.TempDir()
		expectedFile := filepath.Join(dir, "custom.expected.json")

		expected := `{"value": "{{customTest}}"}`
		writeTestFile(t, expectedFile, expected)

		// when: asserting JSON with the custom matcher
		// then: the custom matcher is used and matches any string
		testastic.AssertJSON(t, expectedFile, `{"value": "test"}`)
	})
}
