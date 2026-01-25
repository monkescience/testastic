package testastic

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

// ErrUnsupportedHTMLType is returned when an unsupported type is passed to AssertHTML.
var ErrUnsupportedHTMLType = errors.New("unsupported type for HTML comparison")

// AssertHTML compares actual HTML against an expected HTML file.
// T can be: []byte, string, io.Reader, or any type implementing fmt.Stringer.
//
// Example:
//
//	testastic.AssertHTML(t, "testdata/user.expected.html", resp.Body)
//	testastic.AssertHTML(t, "testdata/user.expected.html", htmlBytes)
//	testastic.AssertHTML(t, "testdata/user.expected.html", htmlString)
//
//nolint:funlen // Main assertion function needs sequential validation steps.
func AssertHTML[T any](tb testing.TB, expectedFile string, actual T, opts ...interface{}) {
	tb.Helper()

	// Convert actual to []byte
	actualBytes, err := toHTMLBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	// Build config
	cfg := &HTMLConfig{
		BaseConfig: BaseConfig{
			Update: shouldUpdate(),
		},
	}
	for _, opt := range opts {
		switch o := opt.(type) {
		case AssertionOption:
			o.applyToHTMLConfig(cfg)
		case HTMLOption:
			o(cfg)
		default:
			tb.Fatalf("testastic: invalid option type: %T", opt)
		}
	}

	// Check if expected file exists
	_, statErr := os.Stat(expectedFile)
	if os.IsNotExist(statErr) {
		if cfg.Update {
			createErr := createExpectedHTMLFile(expectedFile, actualBytes)
			if createErr != nil {
				tb.Fatalf("testastic: failed to create expected HTML file: %v", createErr)
			}

			tb.Logf("testastic: created expected HTML file %s", expectedFile)

			return
		}

		tb.Fatalf(
			"testastic: expected HTML file does not exist: %s (run with -update to create)",
			expectedFile,
		)

		return
	}

	// Parse expected file
	expected, err := ParseExpectedHTMLFile(expectedFile)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	// Parse actual HTML
	actualNode, err := parseActualHTMLBytes(actualBytes)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	// Compare
	diffs := compareHTML(expected.Root, actualNode, cfg)

	// If update mode and there are differences, update the file
	if cfg.Update && len(diffs) > 0 {
		updateErr := updateExpectedHTMLFile(expectedFile, actualBytes)
		if updateErr != nil {
			tb.Fatalf("testastic: failed to update expected HTML file: %v", updateErr)
		}

		tb.Logf("testastic: updated expected HTML file %s", expectedFile)

		return
	}

	// Report differences
	if len(diffs) > 0 {
		sortHTMLDiffs(diffs)
		msg := formatAssertionMessage("AssertHTML", expectedFile, cfg.Message)
		tb.Errorf(
			"testastic: assertion failed\n\n  %s\n%s",
			msg, FormatHTMLDiffInline(expected.Root, actualNode),
		)
	}
}

// toHTMLBytes converts various input types to []byte.
func toHTMLBytes[T any](v T) ([]byte, error) {
	if data, handled, err := bytesFromCommonInput(v); handled {
		return data, err
	}

	if stringer, ok := any(v).(fmt.Stringer); ok {
		return []byte(stringer.String()), nil
	}

	return nil, fmt.Errorf("%w: %T (expected []byte, string, io.Reader, or fmt.Stringer)", ErrUnsupportedHTMLType, v)
}

// createExpectedHTMLFile creates a new expected HTML file with formatted content.
func createExpectedHTMLFile(path string, actual []byte) error {
	// Parse and re-render for consistent formatting
	node, err := parseActualHTMLBytes(actual)
	if err != nil {
		// If parsing fails, just write the raw content
		return writeHTMLFile(path, actual)
	}

	formatted := renderPrettyHTML(node, 0)

	return writeHTMLFile(path, []byte(formatted))
}

// updateExpectedHTMLFile updates an existing expected HTML file.
func updateExpectedHTMLFile(path string, actual []byte) error {
	// Parse and re-render for consistent formatting
	node, err := parseActualHTMLBytes(actual)
	if err != nil {
		// If parsing fails, just write the raw content
		return writeHTMLFile(path, actual)
	}

	formatted := renderPrettyHTML(node, 0)

	return writeHTMLFile(path, []byte(formatted))
}

// writeHTMLFile writes data to a file with proper error wrapping.
func writeHTMLFile(path string, data []byte) error {
	err := os.WriteFile(path, data, filePerm)
	if err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return nil
}
