package testastic

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// errUnsupportedHTMLType is returned when an unsupported type is passed to AssertHTML.
var errUnsupportedHTMLType = errors.New("unsupported type for HTML comparison")

// AssertHTML compares actual HTML against an expected HTML file.
// T can be: []byte, string, io.Reader, or any type implementing fmt.Stringer.
//
// Example:
//
//	testastic.AssertHTML(t, "testdata/user.expected.html", resp.Body)
//	testastic.AssertHTML(t, "testdata/user.expected.html", htmlBytes)
//	testastic.AssertHTML(t, "testdata/user.expected.html", htmlString)
func AssertHTML[T any](tb testing.TB, expectedFile string, actual T, opts ...Option) {
	tb.Helper()

	actualBytes, err := toHTMLBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	cfg := newConfig(opts)

	if unsupported := cfg.validateOptions(assertHTML); len(unsupported) > 0 {
		tb.Fatalf("testastic: unsupported options for AssertHTML: %s", strings.Join(unsupported, ", "))

		return
	}

	if handleMissingExpectedFile(tb, expectedFile, actualBytes, cfg.Update, createExpectedHTMLFile) {
		return
	}

	expected, err := parseExpectedHTMLFile(expectedFile)
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

// handleHTMLDiffs handles update mode and error reporting for HTML.
// Returns true if the assertion should stop.
func handleHTMLDiffs(
	tb testing.TB, path string, actualBytes []byte, expectedRoot, actualNode *htmlNode,
	diffs []htmlDifference, cfg *config,
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
	tb.Errorf("testastic: assertion failed\n\n  %s\n%s", msg, formatHTMLDiffInline(expectedRoot, actualNode))

	return false
}

func toHTMLBytes[T any](v T) ([]byte, error) {
	if data, handled, err := bytesFromCommonInput(v); handled {
		return data, err
	}

	if stringer, ok := any(v).(fmt.Stringer); ok {
		return []byte(stringer.String()), nil
	}

	return nil, fmt.Errorf("unsupported HTML value of type %T: %w", v, errUnsupportedHTMLType)
}

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

func writeHTMLFile(path string, data []byte) error {
	return writeFileAtomic(path, data)
}
