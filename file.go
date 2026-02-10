package testastic

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errUnsupportedFileType is returned when an unsupported type is passed to AssertFile.
var errUnsupportedFileType = errors.New("unsupported type, expected string, []byte, or io.Reader")

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

	if handleMissingExpectedFile(tb, expectedFile, []byte(actualStr), cfg.Update, createExpectedTextFile) {
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
		return "", fmt.Errorf("%w: %T", errUnsupportedFileType, v)
	}
}

func createExpectedTextFile(path string, actual []byte) error {
	dir := filepath.Dir(path)

	mkdirErr := os.MkdirAll(dir, dirPerm)
	if mkdirErr != nil {
		return fmt.Errorf("failed to create directory: %w", mkdirErr)
	}

	writeErr := os.WriteFile(path, actual, filePerm)
	if writeErr != nil {
		return fmt.Errorf("failed to write expected file: %w", writeErr)
	}

	return nil
}

// splitLines splits content into lines, handling different line endings.
func splitLines(content string) []string {
	if content == "" {
		return nil
	}

	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	return strings.Split(content, "\n")
}

func formatFileDiff(expected, actual []string, diffs []difference) string {
	if len(diffs) == 0 {
		return ""
	}

	return formatFileDiffInline(expected, actual)
}
