package testastic

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// ErrUnsupportedFileType is returned when an unsupported type is passed to AssertFile.
var ErrUnsupportedFileType = errors.New("unsupported type, expected string, []byte, or io.Reader")

// expectedFilePerms is the file permission used when creating/updating expected files.
const expectedFilePerms = 0o644

// AssertFile compares actual content against an expected file with template matcher support.
// Supports {{anyString}}, {{regex `pattern`}}, and other matchers inline within text.
//
// Supported actual types: string, []byte, or io.Reader.
func AssertFile[T any](tb testing.TB, expectedFile string, actual T, opts ...Option) {
	tb.Helper()

	actualStr, err := fileToString(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to string: %v", err)

		return
	}

	cfg := buildConfig(opts)

	if unsupported := cfg.validateOptions(assertFile); len(unsupported) > 0 {
		tb.Fatalf("testastic: unsupported options for AssertFile: %s", strings.Join(unsupported, ", "))

		return
	}

	if handleMissingExpectedFile(tb, expectedFile, []byte(actualStr), cfg.Update, createExpectedFile) {
		return
	}

	expectedContent, err := os.ReadFile(expectedFile) //nolint:gosec // Path is controlled by test code.
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
		err := os.WriteFile(expectedFile, []byte(actualStr), expectedFilePerms)
		if err != nil {
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
		return "", fmt.Errorf("%w: %T", ErrUnsupportedFileType, v)
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
func formatFileDiff(expected, actual []string, diffs []difference) string {
	if len(diffs) == 0 {
		return ""
	}

	return FormatFileDiffInline(expected, actual)
}
