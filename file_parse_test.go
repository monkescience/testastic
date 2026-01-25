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
