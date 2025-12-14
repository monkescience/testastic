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
//
//nolint:funlen // Main assertion function needs sequential validation steps.
func AssertJSON[T any](tb testing.TB, expectedFile string, actual T, opts ...Option) {
	tb.Helper()

	// Convert actual to []byte
	actualBytes, err := toBytes(actual)
	if err != nil {
		tb.Fatalf("testastic: failed to convert actual to bytes: %v", err)

		return
	}

	// Build config
	cfg := newConfig(opts...)

	// Check if expected file exists
	_, statErr := os.Stat(expectedFile)
	if os.IsNotExist(statErr) {
		if cfg.Update {
			createErr := createExpectedFile(expectedFile, actualBytes)
			if createErr != nil {
				tb.Fatalf("testastic: failed to create expected file: %v", createErr)
			}

			tb.Logf("testastic: created expected file %s", expectedFile)

			return
		}

		tb.Fatalf(
			"testastic: expected file does not exist: %s (run with -update to create)",
			expectedFile,
		)

		return
	}

	// Parse expected file
	expected, err := ParseExpectedFile(expectedFile)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	// Parse actual JSON
	actualData, err := parseActualJSON(actualBytes)
	if err != nil {
		tb.Fatalf("testastic: %v", err)

		return
	}

	// Compare
	diffs := compare(expected.Data, actualData, "$", cfg)

	// If update mode and there are differences, update the file
	if cfg.Update && len(diffs) > 0 {
		updateErr := updateExpectedFile(expectedFile, actualBytes, expected)
		if updateErr != nil {
			tb.Fatalf("testastic: failed to update expected file: %v", updateErr)
		}

		tb.Logf("testastic: updated expected file %s", expectedFile)

		return
	}

	// Report differences
	if len(diffs) > 0 {
		sortDiffs(diffs)
		tb.Errorf(
			"testastic: assertion failed\n\n  AssertJSON (%s)\n%s",
			expectedFile, FormatDiffInline(expected.Data, actualData),
		)
	}
}

// toBytes converts various input types to []byte of JSON.
func toBytes[T any](v T) ([]byte, error) {
	switch val := any(v).(type) {
	case []byte:
		return val, nil

	case string:
		return []byte(val), nil

	case io.Reader:
		data, err := io.ReadAll(val)
		if err != nil {
			return nil, fmt.Errorf("failed to read from io.Reader: %w", err)
		}

		return data, nil

	default:
		// Marshal struct or other types to JSON
		data, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
		}

		return data, nil
	}
}
