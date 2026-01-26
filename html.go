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
func AssertHTML[T any](tb testing.TB, expectedFile string, actual T, opts ...AssertionOption) {
	tb.Helper()

	actualBytes, err := toHTMLBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	cfg := buildHTMLConfig(tb, opts)

	if handleMissingExpectedFile(tb, expectedFile, actualBytes, cfg.Update, createExpectedHTMLFile) {
		return
	}

	expected, err := ParseExpectedHTMLFile(expectedFile)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	actualNode, err := parseActualHTMLBytes(actualBytes)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	diffs := compareHTML(expected.Root, actualNode, cfg)

	if handleHTMLDiffs(tb, expectedFile, actualBytes, expected.Root, actualNode, diffs, cfg) {
		return
	}
}

// buildHTMLConfig creates an HTML config from the provided options.
func buildHTMLConfig(tb testing.TB, opts []AssertionOption) *HTMLConfig {
	tb.Helper()

	cfg := &HTMLConfig{BaseConfig: BaseConfig{Update: shouldUpdate()}}

	for _, opt := range opts {
		opt.applyToHTMLConfig(cfg)
	}

	return cfg
}

// handleHTMLDiffs handles update mode and error reporting for HTML.
// Returns true if the assertion should stop.
func handleHTMLDiffs(
	tb testing.TB, path string, actualBytes []byte, expectedRoot, actualNode *HTMLNode,
	diffs []HTMLDifference, cfg *HTMLConfig,
) bool {
	tb.Helper()

	if len(diffs) == 0 {
		return false
	}

	if cfg.Update {
		updateErr := updateExpectedHTMLFile(path, actualBytes)
		if updateErr != nil {
			tb.Fatalf("testastic: failed to update expected HTML file: %v", updateErr)
		}

		tb.Logf("testastic: updated expected HTML file %s", path)

		return true
	}

	sortHTMLDiffs(diffs)

	msg := formatAssertionMessage("AssertHTML", path, cfg.Message)
	tb.Errorf("testastic: assertion failed\n\n  %s\n%s", msg, FormatHTMLDiffInline(expectedRoot, actualNode))

	return false
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
