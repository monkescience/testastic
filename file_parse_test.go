package testastic

import (
	"testing"
)

func TestParseLine_NoMatchers(t *testing.T) {
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
}

func TestParseLine_SingleMatcher(t *testing.T) {
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
	if result.matchers[0].String() != "{{anyString}}" {
		t.Errorf("expected {{anyString}} matcher, got %s", result.matchers[0].String())
	}
	// Verify pattern matches expected format
	if !result.pattern.MatchString("Name: John Doe") {
		t.Error("pattern should match 'Name: John Doe'")
	}
	if result.pattern.MatchString("Email: test@example.com") {
		t.Error("pattern should not match 'Email: test@example.com'")
	}
}
