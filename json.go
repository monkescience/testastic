// Package testastic provides JSON assertion utilities for Go tests.
// It compares API responses against expected JSON files with support for
// template-based matchers, semantic comparison, and automatic updates.
package testastic

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
func AssertJSON[T any](tb testing.TB, expectedFile string, actual T, opts ...any) {
	tb.Helper()

	actualBytes, err := toBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	cfg := buildJSONConfig(tb, opts)

	if handleMissingExpectedFile(tb, expectedFile, actualBytes, cfg.Update, createExpectedFile) {
		return
	}

	expected, err := ParseExpectedFile(expectedFile)
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

// buildJSONConfig creates a JSON config from the provided options.
func buildJSONConfig(tb testing.TB, opts []any) *JSONConfig {
	tb.Helper()

	cfg := &JSONConfig{BaseConfig: BaseConfig{Update: shouldUpdate()}}

	for _, opt := range opts {
		switch o := opt.(type) {
		case AssertionOption:
			o.applyToConfig(cfg)
		case JSONOption:
			o(cfg)
		default:
			tb.Fatalf("testastic: invalid option type: %T", opt)
		}
	}

	return cfg
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
	tb testing.TB, path string, actualBytes []byte, expected *ExpectedJSON,
	actualData any, diffs []Difference, cfg *JSONConfig,
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

	sortDiffs(diffs)

	msg := formatAssertionMessage("AssertJSON", path, cfg.Message)
	tb.Errorf("testastic: assertion failed\n\n  %s\n%s", msg, FormatDiffInline(expected.Data, actualData))

	return false
}

// formatAssertionMessage creates the assertion header with optional custom message.
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
