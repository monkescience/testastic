package testastic

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// AssertFile compares actual content against an expected file with template matcher support.
// Supports {{anyString}}, {{regex `pattern`}}, and other matchers inline within text.
//
// Supported actual types: string, []byte, io.Reader
func AssertFile[T any](tb testing.TB, expectedFile string, actual T, opts ...AssertionOption) {
	tb.Helper()

	actualStr, err := fileToString(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to string: %v", err)
		return
	}

	cfg := buildFileConfig(opts)

	if handleMissingExpectedFile(tb, expectedFile, []byte(actualStr), cfg.Update, createExpectedFile) {
		return
	}

	expectedContent, err := os.ReadFile(expectedFile)
	if err != nil {
		tb.Fatalf("testastic: failed to read expected file: %v", err)
		return
	}

	expectedLines := splitLines(string(expectedContent))
	actualLines := splitLines(actualStr)

	diffs := compareFileLinesWithMatchers(expectedLines, actualLines)

	if len(diffs) == 0 {
		return
	}

	if cfg.Update {
		if err := os.WriteFile(expectedFile, []byte(actualStr), 0644); err != nil {
			tb.Fatalf("testastic: failed to update expected file: %v", err)
			return
		}
		tb.Logf("testastic: updated expected file %s", expectedFile)
		return
	}

	msg := formatAssertionMessage("AssertFile", expectedFile, cfg.Message)
	tb.Errorf("testastic: assertion failed\n\n  %s\n%s",
		msg, formatFileDiff(expectedLines, actualLines, diffs))
}

// buildFileConfig creates a file config from the provided options.
func buildFileConfig(opts []AssertionOption) *FileConfig {
	cfg := &FileConfig{BaseConfig: BaseConfig{Update: shouldUpdate()}}
	for _, opt := range opts {
		opt.applyToFileConfig(cfg)
	}
	return cfg
}

// fileToString converts various input types to string.
func fileToString[T any](v T) (string, error) {
	switch val := any(v).(type) {
	case string:
		return val, nil
	case []byte:
		return string(val), nil
	case io.Reader:
		data, err := io.ReadAll(val)
		if err != nil {
			return "", fmt.Errorf("failed to read from io.Reader: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported type %T, expected string, []byte, or io.Reader", v)
	}
}

// splitLines splits content into lines, handling different line endings.
func splitLines(content string) []string {
	if content == "" {
		return nil
	}
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.Split(content, "\n")
}

// formatFileDiff formats differences for display using inline diff style.
func formatFileDiff(expected, actual []string, diffs []Difference) string {
	if len(diffs) == 0 {
		return ""
	}
	return FormatFileDiffInline(expected, actual)
}
