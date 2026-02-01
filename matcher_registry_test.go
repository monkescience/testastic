package testastic_test

import (
	"path/filepath"
	"testing"

	"github.com/monkescience/testastic"
)

func TestNewMatchers(t *testing.T) {
	t.Run("AnyUUID", func(t *testing.T) {
		m := testastic.AnyUUID()

		if !m.Match("550e8400-e29b-41d4-a716-446655440000") {
			t.Error("Should match valid UUID")
		}

		if !m.Match("6ba7b810-9dad-11d1-80b4-00c04fd430c8") {
			t.Error("Should match valid UUID")
		}

		if m.Match("not-a-uuid") {
			t.Error("Should not match invalid UUID")
		}

		if m.Match(123) {
			t.Error("Should not match non-string")
		}

		if m.String() != "{{anyUUID}}" {
			t.Errorf("String() = %v, want {{anyUUID}}", m.String())
		}
	})

	t.Run("AnyDateTime", func(t *testing.T) {
		m := testastic.AnyDateTime()

		if !m.Match("2024-01-15") {
			t.Error("Should match date")
		}

		if !m.Match("2024-01-15T10:30:00Z") {
			t.Error("Should match ISO 8601 datetime")
		}

		if !m.Match("2024-01-15 10:30:00") {
			t.Error("Should match datetime with space")
		}

		if m.Match("not-a-date") {
			t.Error("Should not match invalid date")
		}

		if m.Match(123) {
			t.Error("Should not match non-string")
		}

		if m.String() != "{{anyDateTime}}" {
			t.Errorf("String() = %v, want {{anyDateTime}}", m.String())
		}
	})

	t.Run("AnyURL", func(t *testing.T) {
		m := testastic.AnyURL()

		if !m.Match("https://example.com") {
			t.Error("Should match HTTPS URL")
		}

		if !m.Match("http://example.com/path") {
			t.Error("Should match HTTP URL with path")
		}

		if m.Match("not-a-url") {
			t.Error("Should not match invalid URL")
		}

		if m.Match(123) {
			t.Error("Should not match non-string")
		}

		if m.String() != "{{anyURL}}" {
			t.Errorf("String() = %v, want {{anyURL}}", m.String())
		}
	})
}

func TestCustomMatcherRegistry(t *testing.T) {
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
}
