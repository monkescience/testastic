package testastic_test

import (
	"fmt"
	"testing"
)

// mockT implements testing.TB for capturing test failures without stopping the test runner.
type mockT struct {
	testing.TB
	failed  bool
	message string
}

func (m *mockT) Helper() {}

func (m *mockT) Fatalf(format string, args ...any) {
	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func (m *mockT) Errorf(format string, args ...any) {
	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func (m *mockT) Logf(format string, args ...any) {}
