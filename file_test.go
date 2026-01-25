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
	if err := os.WriteFile(expectedFile, []byte(content), 0644); err != nil {
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
	if err := os.WriteFile(expectedFile, []byte("expected line"), 0644); err != nil {
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
