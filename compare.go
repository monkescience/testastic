package testastic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// compare compares expected (from expected file) with actual JSON data.
// Returns a list of differences found.
//
//nolint:funlen // Complex type dispatch is clearer in one function.
func compare(expected, actual any, path string, cfg *Config) []Difference {
	if cfg.isFieldIgnored(path) {
		return nil
	}

	if m, ok := expected.(Matcher); ok {
		if IsIgnore(m) {
			return nil
		}

		if !m.Match(actual) {
			return []Difference{{
				Path:     path,
				Expected: m.String(),
				Actual:   actual,
				Type:     DiffMatcherFailed,
			}}
		}

		return nil
	}

	if expected == nil && actual == nil {
		return nil
	}

	if expected == nil {
		return []Difference{{
			Path:     path,
			Expected: nil,
			Actual:   actual,
			Type:     DiffAdded,
		}}
	}

	if actual == nil {
		return []Difference{{
			Path:     path,
			Expected: expected,
			Actual:   nil,
			Type:     DiffRemoved,
		}}
	}

	switch exp := expected.(type) {
	case map[string]any:
		return compareObjects(exp, actual, path, cfg)

	case []any:
		return compareArrays(exp, actual, path, cfg)

	case string:
		if act, ok := actual.(string); ok {
			if exp != act {
				return []Difference{{
					Path:     path,
					Expected: exp,
					Actual:   act,
					Type:     DiffChanged,
				}}
			}

			return nil
		}

		return []Difference{{
			Path:     path,
			Expected: exp,
			Actual:   actual,
			Type:     DiffTypeMismatch,
		}}

	case float64:
		return compareNumbers(exp, actual, path)

	case bool:
		if act, ok := actual.(bool); ok {
			if exp != act {
				return []Difference{{
					Path:     path,
					Expected: exp,
					Actual:   act,
					Type:     DiffChanged,
				}}
			}

			return nil
		}

		return []Difference{{
			Path:     path,
			Expected: exp,
			Actual:   actual,
			Type:     DiffTypeMismatch,
		}}

	default:
		// For other types, use deep equality
		if !reflect.DeepEqual(expected, actual) {
			return []Difference{{
				Path:     path,
				Expected: expected,
				Actual:   actual,
				Type:     DiffChanged,
			}}
		}

		return nil
	}
}

// compareObjects compares two JSON objects (maps).
func compareObjects(expected map[string]any, actual any, path string, cfg *Config) []Difference {
	actMap, ok := actual.(map[string]any)
	if !ok {
		return []Difference{{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Type:     DiffTypeMismatch,
		}}
	}

	var diffs []Difference

	// First pass: check for missing and changed keys in expected.
	for key, expVal := range expected {
		childPath := path + "." + key
		if cfg.isFieldIgnored(childPath) {
			continue
		}

		if m, ok := expVal.(Matcher); ok && IsIgnore(m) {
			continue
		}

		actVal, exists := actMap[key]
		if !exists {
			diffs = append(diffs, Difference{
				Path:     childPath,
				Expected: expVal,
				Actual:   nil,
				Type:     DiffRemoved,
			})
		} else {
			diffs = append(diffs, compare(expVal, actVal, childPath, cfg)...)
		}
	}

	// Second pass: check for extra keys in actual.
	for key, actVal := range actMap {
		childPath := path + "." + key
		if cfg.isFieldIgnored(childPath) {
			continue
		}

		if _, exists := expected[key]; !exists {
			diffs = append(diffs, Difference{
				Path:     childPath,
				Expected: nil,
				Actual:   actVal,
				Type:     DiffAdded,
			})
		}
	}

	return diffs
}

// compareArrays compares two JSON arrays.
func compareArrays(expected []any, actual any, path string, cfg *Config) []Difference {
	actArr, ok := actual.([]any)
	if !ok {
		return []Difference{{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Type:     DiffTypeMismatch,
		}}
	}

	if cfg.shouldIgnoreArrayOrder(path) {
		return compareArraysUnordered(expected, actArr, path, cfg)
	}

	return compareArraysOrdered(expected, actArr, path, cfg)
}

// compareArraysOrdered compares arrays where order matters.
func compareArraysOrdered(expected, actual []any, path string, cfg *Config) []Difference {
	var diffs []Difference

	for i := range max(len(expected), len(actual)) {
		childPath := fmt.Sprintf("%s[%d]", path, i)

		switch {
		case i >= len(expected):
			diffs = append(diffs, Difference{
				Path:     childPath,
				Expected: nil,
				Actual:   actual[i],
				Type:     DiffAdded,
			})
		case i >= len(actual):
			diffs = append(diffs, Difference{
				Path:     childPath,
				Expected: expected[i],
				Actual:   nil,
				Type:     DiffRemoved,
			})
		default:
			diffs = append(diffs, compare(expected[i], actual[i], childPath, cfg)...)
		}
	}

	return diffs
}

// compareArraysUnordered compares arrays where order doesn't matter.
//
//nolint:funlen // Unordered comparison requires explicit matching logic.
func compareArraysUnordered(expected, actual []any, path string, cfg *Config) []Difference {
	if len(expected) != len(actual) {
		return []Difference{{
			Path:     path,
			Expected: fmt.Sprintf("array of length %d", len(expected)),
			Actual:   fmt.Sprintf("array of length %d", len(actual)),
			Type:     DiffChanged,
		}}
	}

	used := make([]bool, len(actual))

	var unmatched []int

	for i, exp := range expected {
		found := false

		for j, act := range actual {
			if used[j] {
				continue
			}

			if len(compare(exp, act, path, cfg)) == 0 {
				used[j] = true
				found = true

				break
			}
		}

		if !found {
			unmatched = append(unmatched, i)
		}
	}

	if len(unmatched) > 0 {
		var unusedActual []int

		for i, u := range used {
			if !u {
				unusedActual = append(unusedActual, i)
			}
		}

		var diffs []Difference

		for i, idx := range unmatched {
			childPath := fmt.Sprintf("%s[%d]", path, idx)

			var actualVal any
			if i < len(unusedActual) {
				actualVal = actual[unusedActual[i]]
			}

			diffs = append(diffs, Difference{
				Path:     childPath,
				Expected: expected[idx],
				Actual:   actualVal,
				Type:     DiffChanged,
			})
		}

		return diffs
	}

	return nil
}

// compareNumbers compares numeric values, handling JSON number quirks.
func compareNumbers(expected float64, actual any, path string) []Difference {
	var actNum float64

	switch v := actual.(type) {
	case float64:
		actNum = v
	case float32:
		actNum = float64(v)
	case int:
		actNum = float64(v)
	case int64:
		actNum = float64(v)
	case int32:
		actNum = float64(v)
	default:
		return []Difference{{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Type:     DiffTypeMismatch,
		}}
	}

	if expected != actNum {
		return []Difference{{
			Path:     path,
			Expected: expected,
			Actual:   actNum,
			Type:     DiffChanged,
		}}
	}

	return nil
}

// parseActualJSON converts the actual value to a comparable JSON structure.
func parseActualJSON(data []byte) (any, error) {
	var result any

	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actual JSON: %w", err)
	}

	return result, nil
}

// sortDiffs sorts differences by path for consistent output.
func sortDiffs(diffs []Difference) {
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})
}
