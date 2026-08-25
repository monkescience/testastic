package testastic

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
)

// jsonNumberPattern matches the JSON number grammar (RFC 8259). It is stricter
// than big.Rat.SetString, which also accepts Go-only literals like "0x1p4" or
// "1_000" that no JSON or YAML decoder can produce.
var jsonNumberPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

// compare compares expected (from expected file) with actual JSON data.
// Returns a list of differences found.
//
//nolint:funlen // Complex type dispatch is clearer in one function.
func compare(expected, actual any, path string, cfg *config) []difference {
	if cfg.IsFieldIgnored(path) {
		return nil
	}

	if m, ok := expected.(Matcher); ok {
		if isIgnore(m) {
			return nil
		}

		if !m.Match(actual) {
			return []difference{{
				Path:     path,
				Expected: m.String(),
				Actual:   actual,
				Type:     diffMatcherFailed,
			}}
		}

		return nil
	}

	if expected == nil && actual == nil {
		return nil
	}

	if expected == nil {
		return []difference{{
			Path:     path,
			Expected: nil,
			Actual:   actual,
			Type:     diffAdded,
		}}
	}

	if actual == nil {
		return []difference{{
			Path:     path,
			Expected: expected,
			Actual:   nil,
			Type:     diffRemoved,
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
				return []difference{{
					Path:     path,
					Expected: exp,
					Actual:   act,
					Type:     diffChanged,
				}}
			}

			return nil
		}

		return []difference{{
			Path:     path,
			Expected: exp,
			Actual:   actual,
			Type:     diffTypeMismatch,
		}}

	case json.Number, float64, float32,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return compareNumbers(exp, actual, path)

	case bool:
		if act, ok := actual.(bool); ok {
			if exp != act {
				return []difference{{
					Path:     path,
					Expected: exp,
					Actual:   act,
					Type:     diffChanged,
				}}
			}

			return nil
		}

		return []difference{{
			Path:     path,
			Expected: exp,
			Actual:   actual,
			Type:     diffTypeMismatch,
		}}

	default:
		if !reflect.DeepEqual(expected, actual) {
			return []difference{{
				Path:     path,
				Expected: expected,
				Actual:   actual,
				Type:     diffChanged,
			}}
		}

		return nil
	}
}

func compareObjects(expected map[string]any, actual any, path string, cfg *config) []difference {
	actMap, ok := actual.(map[string]any)
	if !ok {
		return []difference{{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Type:     diffTypeMismatch,
		}}
	}

	var diffs []difference

	for key, expVal := range expected {
		childPath := path + "." + key
		if cfg.IsFieldIgnored(childPath) {
			continue
		}

		if m, ok := expVal.(Matcher); ok && isIgnore(m) {
			continue
		}

		actVal, exists := actMap[key]
		if !exists {
			diffs = append(diffs, difference{
				Path:     childPath,
				Expected: expVal,
				Actual:   nil,
				Type:     diffRemoved,
			})
		} else {
			diffs = append(diffs, compare(expVal, actVal, childPath, cfg)...)
		}
	}

	for key, actVal := range actMap {
		childPath := path + "." + key
		if cfg.IsFieldIgnored(childPath) {
			continue
		}

		if _, exists := expected[key]; !exists {
			diffs = append(diffs, difference{
				Path:     childPath,
				Expected: nil,
				Actual:   actVal,
				Type:     diffAdded,
			})
		}
	}

	return diffs
}

func compareArrays(expected []any, actual any, path string, cfg *config) []difference {
	actArr, ok := actual.([]any)
	if !ok {
		return []difference{{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Type:     diffTypeMismatch,
		}}
	}

	if cfg.ShouldIgnoreArrayOrder(path) {
		return compareArraysUnordered(expected, actArr, path, cfg)
	}

	return compareArraysOrdered(expected, actArr, path, cfg)
}

func compareArraysOrdered(expected, actual []any, path string, cfg *config) []difference {
	var diffs []difference

	for i := range max(len(expected), len(actual)) {
		childPath := fmt.Sprintf("%s[%d]", path, i)

		switch {
		case i >= len(expected):
			diffs = append(diffs, difference{
				Path:     childPath,
				Expected: nil,
				Actual:   actual[i],
				Type:     diffAdded,
			})
		case i >= len(actual):
			diffs = append(diffs, difference{
				Path:     childPath,
				Expected: expected[i],
				Actual:   nil,
				Type:     diffRemoved,
			})
		default:
			diffs = append(diffs, compare(expected[i], actual[i], childPath, cfg)...)
		}
	}

	return diffs
}

type unorderedMatchResult struct {
	expectedByActual  []int
	unmatchedExpected []int
	unusedActual      []int
}

// findUnorderedMatches finds a maximum matching between expected and actual slices.
func findUnorderedMatches[T any](expected, actual []T, matches func(expIdx int, act T) bool) unorderedMatchResult {
	result := unorderedMatchResult{
		expectedByActual: make([]int, len(actual)),
	}

	for i := range result.expectedByActual {
		result.expectedByActual[i] = -1
	}

	for i := range expected {
		seen := make([]bool, len(actual))
		_ = assignUnorderedMatch(i, actual, matches, result.expectedByActual, seen)
	}

	expectedMatched := make([]bool, len(expected))

	for _, expIdx := range result.expectedByActual {
		if expIdx >= 0 {
			expectedMatched[expIdx] = true
		}
	}

	for i, matched := range expectedMatched {
		if !matched {
			result.unmatchedExpected = append(result.unmatchedExpected, i)
		}
	}

	for i, expIdx := range result.expectedByActual {
		if expIdx < 0 {
			result.unusedActual = append(result.unusedActual, i)
		}
	}

	return result
}

func assignUnorderedMatch[T any](
	expIdx int,
	actual []T,
	matches func(expIdx int, act T) bool,
	expectedByActual []int,
	seen []bool,
) bool {
	for j, act := range actual {
		if seen[j] || !matches(expIdx, act) {
			continue
		}

		seen[j] = true

		if expectedByActual[j] < 0 ||
			assignUnorderedMatch(expectedByActual[j], actual, matches, expectedByActual, seen) {
			expectedByActual[j] = expIdx

			return true
		}
	}

	return false
}

func compareArraysUnordered(expected, actual []any, path string, cfg *config) []difference {
	if len(expected) != len(actual) {
		return []difference{{
			Path:     path,
			Expected: fmt.Sprintf("array of length %d", len(expected)),
			Actual:   fmt.Sprintf("array of length %d", len(actual)),
			Type:     diffChanged,
		}}
	}

	matches := findUnorderedMatches(expected, actual, func(expIdx int, act any) bool {
		childPath := fmt.Sprintf("%s[%d]", path, expIdx)

		return len(compare(expected[expIdx], act, childPath, cfg)) == 0
	})

	if len(matches.unmatchedExpected) == 0 {
		return nil
	}

	var diffs []difference

	for i, idx := range matches.unmatchedExpected {
		childPath := fmt.Sprintf("%s[%d]", path, idx)

		var actualVal any
		if i < len(matches.unusedActual) {
			actualVal = actual[matches.unusedActual[i]]
		}

		diffs = append(diffs, difference{
			Path:     childPath,
			Expected: expected[idx],
			Actual:   actualVal,
			Type:     diffChanged,
		})
	}

	return diffs
}

func compareNumbers(expected, actual any, path string) []difference {
	expNum, expOK := numberToRat(expected)

	actNum, actOK := numberToRat(actual)
	if !expOK || !actOK {
		// big.Rat cannot represent infinities, but identical infinities are
		// still equal. NaN intentionally remains unequal to everything.
		if exp, ok := floatInfSign(expected); ok {
			if act, ok := floatInfSign(actual); ok && exp == act {
				return nil
			}
		}

		return []difference{{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Type:     diffTypeMismatch,
		}}
	}

	if expNum.Cmp(actNum) != 0 {
		return []difference{{
			Path:     path,
			Expected: expected,
			Actual:   actual,
			Type:     diffChanged,
		}}
	}

	return nil
}

// floatInfSign reports the sign of an infinite float value (+1 or -1) and
// whether v is in fact an infinity.
func floatInfSign(v any) (int, bool) {
	var f float64

	switch n := v.(type) {
	case float64:
		f = n
	case float32:
		f = float64(n)
	default:
		return 0, false
	}

	switch {
	case math.IsInf(f, 1):
		return 1, true
	case math.IsInf(f, -1):
		return -1, true
	default:
		return 0, false
	}
}

func numberToRat(value any) (*big.Rat, bool) {
	switch v := value.(type) {
	case json.Number:
		return parseJSONNumberRat(v.String())
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, false
		}

		return parseNumberRat(strconv.FormatFloat(v, 'g', -1, 64))
	case float32:
		f := float64(v)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, false
		}

		return parseNumberRat(strconv.FormatFloat(f, 'g', -1, 32))
	case int:
		return intRat(int64(v)), true
	case int8:
		return intRat(int64(v)), true
	case int16:
		return intRat(int64(v)), true
	case int32:
		return intRat(int64(v)), true
	case int64:
		return intRat(v), true
	case uint:
		return uintRat(uint64(v)), true
	case uint8:
		return uintRat(uint64(v)), true
	case uint16:
		return uintRat(uint64(v)), true
	case uint32:
		return uintRat(uint64(v)), true
	case uint64:
		return uintRat(v), true
	default:
		return nil, false
	}
}

func parseNumberRat(s string) (*big.Rat, bool) {
	rat := new(big.Rat)

	return rat.SetString(s)
}

// parseJSONNumberRat parses s as a JSON number, rejecting Go-only literal forms
// that big.Rat.SetString would otherwise accept.
func parseJSONNumberRat(s string) (*big.Rat, bool) {
	if !jsonNumberPattern.MatchString(s) {
		return nil, false
	}

	return new(big.Rat).SetString(s)
}

func intRat(v int64) *big.Rat {
	return new(big.Rat).SetInt64(v)
}

func uintRat(v uint64) *big.Rat {
	integer := new(big.Int).SetUint64(v)

	return new(big.Rat).SetInt(integer)
}

func parseActualJSON(data []byte) (any, error) {
	var result any

	err := decodeJSON(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actual JSON: %w", err)
	}

	return result, nil
}
