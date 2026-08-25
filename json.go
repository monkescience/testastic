package testastic

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// AssertJSON compares actual JSON against an expected JSON file.
// T can be: []byte, string, io.Reader, or any struct (auto-marshaled).
//
// Example:
//
//	testastic.AssertJSON(t, "testdata/user.expected.json", resp.Body)
//	testastic.AssertJSON(t, "testdata/user.expected.json", myUser)
//	testastic.AssertJSON(t, "testdata/user.expected.json", jsonBytes)
//
//nolint:dupl // Parallel structure with AssertYAML keeps each format's flow readable.
func AssertJSON[T any](tb testing.TB, expectedFile string, actual T, opts ...Option) {
	tb.Helper()

	actualBytes, err := toBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	cfg := newConfig(opts)

	if unsupported := cfg.validateOptions(assertJSON); len(unsupported) > 0 {
		tb.Fatalf("testastic: unsupported options for AssertJSON: %s", strings.Join(unsupported, ", "))

		return
	}

	if handleMissingExpectedFile(tb, expectedFile, actualBytes, cfg.Update, createExpectedFile) {
		return
	}

	expected, err := parseExpectedJSONFile(expectedFile)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	actualData, err := parseActualJSON(actualBytes)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	diffs := compare(expected.Data, actualData, "$", cfg)

	if handleJSONDiffs(tb, expectedFile, actualBytes, expected, actualData, diffs, cfg) {
		return
	}
}

// handleMissingExpectedFile checks if file exists and creates it in update mode.
// Returns true if the assertion should stop (file was created or fatal error).
func handleMissingExpectedFile(
	tb testing.TB, path string, actualBytes []byte, update bool, createFn func(string, []byte) error,
) bool {
	tb.Helper()

	_, statErr := os.Stat(path)
	if !os.IsNotExist(statErr) {
		return false
	}

	if update {
		createErr := createFn(path, actualBytes)
		if createErr != nil {
			tb.Fatalf("testastic: failed to create expected file: %v", createErr)
		}

		tb.Logf("testastic: created expected file %s", path)

		return true
	}

	tb.Fatalf("testastic: expected file does not exist: %s (run with -update to create)", path)

	return true
}

// handleJSONDiffs handles update mode and error reporting for JSON.
// Returns true if the assertion should stop.
func handleJSONDiffs(
	tb testing.TB, path string, actualBytes []byte, expected *expectedJSON,
	actualData any, diffs []difference, cfg *config,
) bool {
	tb.Helper()

	if len(diffs) == 0 {
		return false
	}

	if cfg.Update {
		updateErr := updateExpectedFile(path, actualBytes, expected)
		if updateErr != nil {
			tb.Fatalf("testastic: failed to update expected file: %v", updateErr)
		}

		tb.Logf("testastic: updated expected file %s", path)

		return true
	}

	msg := formatAssertionMessage("AssertJSON", path, cfg.Message)
	tb.Errorf("testastic: assertion failed\n\n  %s\n%s", msg, formatJSONDiffInline(expected.Data, actualData, cfg))

	return false
}

func formatAssertionMessage(assertType, file, customMsg string) string {
	if customMsg != "" {
		return assertType + " (" + file + "): " + customMsg
	}

	return assertType + " (" + file + ")"
}

// bytesFromCommonInput handles common input types ([]byte, string, io.Reader).
// Returns (bytes, handled, error). If handled is false, caller should handle the type.
func bytesFromCommonInput[T any](v T) ([]byte, bool, error) {
	switch val := any(v).(type) {
	case []byte:
		return val, true, nil

	case string:
		return []byte(val), true, nil

	case io.Reader:
		data, err := io.ReadAll(val)
		if err != nil {
			return nil, true, fmt.Errorf("failed to read from io.Reader: %w", err)
		}

		return data, true, nil

	default:
		return nil, false, nil
	}
}

// toBytes converts various input types to []byte of JSON.
func toBytes[T any](v T) ([]byte, error) {
	if data, handled, err := bytesFromCommonInput(v); handled {
		return data, err
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return data, nil
}
