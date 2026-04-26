package testastic_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monkescience/testastic"
)

func TestAssertJSON(t *testing.T) {
	t.Parallel()

	t.Run("exact match", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with exact values
		// when: asserting with matching JSON
		// then: the test passes without failure
		testastic.AssertJSON(t, "testdata/json/exact_match.json", `{"name": "Alice", "age": 30, "active": true}`)
	})

	t.Run("mismatch", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file and non-matching actual JSON
		mt := &mockT{}
		actual := `{"name": "Bob", "age": 25}`

		// when: asserting with mismatched JSON
		testastic.AssertJSON(mt, "testdata/json/mismatch.json", actual)

		// then: the test fails and diff mentions the differing fields
		if !mt.failed {
			t.Error("expected test to fail")
		}

		if !strings.Contains(mt.message, `"name"`) {
			t.Errorf("expected diff to mention name field, got: %s", mt.message)
		}

		if !strings.Contains(mt.message, `"age"`) {
			t.Errorf("expected diff to mention age field, got: %s", mt.message)
		}
	})

	t.Run("with anyString matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with anyString matcher
		// when: asserting with any string value for id
		actual := `{"id": "abc-123-xyz", "name": "Alice"}`

		// then: the test passes (matcher accepts any string)
		testastic.AssertJSON(t, "testdata/json/with_anystring.json", actual)
	})

	t.Run("with anyInt matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with anyInt matcher
		// when: asserting with any integer value for count
		actual := `{"count": 42, "name": "test"}`

		// then: the test passes (matcher accepts any integer)
		testastic.AssertJSON(t, "testdata/json/with_anyint.json", actual)
	})

	t.Run("with ignore matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with ignore matchers
		// when: asserting with any values for ignored fields
		actual := `{"id": 12345, "timestamp": "2024-01-15T10:30:00Z", "name": "Alice"}`

		// then: the test passes (ignored fields are not compared)
		testastic.AssertJSON(t, "testdata/json/with_ignore.json", actual)
	})

	t.Run("with regex matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with regex matcher
		// when: asserting with a value matching the regex pattern
		actual := `{"email": "alice@example.com"}`

		// then: the test passes (value matches regex)
		testastic.AssertJSON(t, "testdata/json/with_regex.json", actual)
	})

	t.Run("with regex matcher containing braces", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with regex containing braces
		// when: asserting with a value matching the regex pattern
		actual := `{"date": "2024-01-15"}`

		// then: the test passes (value matches regex with quantifiers)
		testastic.AssertJSON(t, "testdata/json/with_regex_braces.json", actual)
	})

	t.Run("with oneOf matcher", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with oneOf matcher
		// when: asserting with a value from the allowed set
		actual := `{"status": "active"}`

		// then: the test passes (value is one of the allowed values)
		testastic.AssertJSON(t, "testdata/json/with_oneof.json", actual)
	})

	t.Run("nested objects", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with nested objects and matchers
		// when: asserting with matching nested structure
		actual := `{"user": {"id": "usr-123", "profile": {"name": "Alice", "age": 30}}}`

		// then: the test passes (nested structure matches)
		testastic.AssertJSON(t, "testdata/json/nested.json", actual)
	})

	t.Run("arrays", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with arrays
		// when: asserting with matching array content and order
		actual := `{"items": [{"id": 1, "name": "first"}, {"id": 2, "name": "second"}]}`

		// then: the test passes (array matches exactly)
		testastic.AssertJSON(t, "testdata/json/arrays.json", actual)
	})

	t.Run("ignore array order", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with an array
		// when: asserting with same elements in different order using IgnoreArrayOrder
		actual := `{"tags": ["c", "a", "b"]}`

		// then: the test passes (order is ignored)
		testastic.AssertJSON(t, "testdata/json/array_order.json", actual, testastic.IgnoreArrayOrder())
	})

	t.Run("ignore array order at", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with ordered and unordered arrays
		// when: asserting with different order only in the unordered array
		actual := `{"ordered": [1, 2, 3], "unordered": ["c", "a", "b"]}`

		// then: the test passes (order ignored only at specified path)
		testastic.AssertJSON(t, "testdata/json/array_order_at.json", actual, testastic.IgnoreArrayOrderAt("$.unordered"))
	})

	t.Run("ignore array order backtracks broad matchers", func(t *testing.T) {
		t.Parallel()

		// given: a broad matcher before a more specific expected object
		expectedFile := filepath.Join(t.TempDir(), "expected.json")
		err := os.WriteFile(expectedFile, []byte(`{"items":["{{anyValue}}",{"id":1}]}`), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}
		actual := `{"items":[{"id":1},"anything"]}`

		// when: comparing the array without considering order
		testastic.AssertJSON(mt, expectedFile, actual, testastic.IgnoreArrayOrderAt("$.items"))

		// then: the broad matcher does not consume the only value needed by the specific object
		if mt.failed {
			t.Errorf("expected no failure with backtracking unordered match, got: %s", mt.message)
		}
	})

	t.Run("ignore fields", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with fields to ignore
		// when: asserting with different values for ignored fields
		actual := `{"id": "different", "name": "Alice", "timestamp": "2024-12-15"}`

		// then: the test passes (specified fields are ignored)
		testastic.AssertJSON(t, "testdata/json/ignore_fields.json", actual, testastic.IgnoreFields("id", "timestamp"))
	})

	t.Run("from struct", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file and a Go struct with matching data
		type User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		actual := User{Name: "StructUser", Age: 35}

		// when: asserting with the struct as actual value
		// then: the test passes (struct is serialized and matches)
		testastic.AssertJSON(t, "testdata/json/struct.json", actual)
	})

	t.Run("from reader", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file and an io.Reader with matching content
		actual := bytes.NewReader([]byte(`{"name": "Reader User"}`))

		// when: asserting with the io.Reader as actual value
		// then: the test passes (reader content matches)
		testastic.AssertJSON(t, "testdata/json/reader.json", actual)
	})

	t.Run("from bytes", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file and matching JSON bytes
		actual := []byte(`{"name": "Alice", "age": 30, "active": true}`)

		// when: asserting with []byte as actual input
		// then: the test passes without failure
		testastic.AssertJSON(t, "testdata/json/exact_match.json", actual)
	})

	t.Run("extra field", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file without an extra field
		mt := &mockT{}
		actual := `{"name": "Alice", "extra": "field"}`

		// when: asserting with JSON containing an extra field
		testastic.AssertJSON(mt, "testdata/json/extra_field.json", actual)

		// then: the test fails and diff mentions the extra field
		if !mt.failed {
			t.Error("expected test to fail due to extra field")
		}

		if !strings.Contains(mt.message, `"extra"`) {
			t.Errorf("expected diff to mention extra field, got: %s", mt.message)
		}
	})

	t.Run("large integer mismatch", func(t *testing.T) {
		// given: adjacent integers above float64's exact integer range
		expectedFile := filepath.Join(t.TempDir(), "expected.json")
		err := os.WriteFile(expectedFile, []byte(`{"id":9007199254740992}`), 0o600)
		testastic.NoError(t, err)

		mt := &mockT{}

		// when: asserting with a different integer that rounds to the same float64
		testastic.AssertJSON(mt, expectedFile, `{"id":9007199254740993}`)

		// then: the assertion detects the numeric mismatch
		if !mt.failed {
			t.Error("expected large integer mismatch to fail")
		}
	})

	t.Run("missing field", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file with a field that actual lacks
		mt := &mockT{}

		// when: asserting with JSON missing the age field
		testastic.AssertJSON(mt, "testdata/json/missing_field.json", `{"name": "Alice"}`)

		// then: the test fails and diff mentions the missing field
		if !mt.failed {
			t.Error("expected test to fail due to missing field")
		}

		if !strings.Contains(mt.message, `"age"`) {
			t.Errorf("expected diff to mention age field, got: %s", mt.message)
		}
	})

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		// given: an expected JSON file and mismatched actual JSON
		mt := &mockT{}
		actual := `{"name": "Bob", "age": 25}`

		// when: asserting with a custom message
		testastic.AssertJSON(mt, "testdata/json/mismatch.json", actual, testastic.Message("custom json message"))

		// then: the failure includes the custom message
		if !mt.failed {
			t.Error("expected test to fail")
		}

		if !strings.Contains(mt.message, "custom json message") {
			t.Errorf("expected custom message in failure, got: %s", mt.message)
		}
	})
}

func TestAssertJSON_UnsupportedOptions(t *testing.T) {
	t.Parallel()

	t.Run("html options rejected", func(t *testing.T) {
		t.Parallel()

		// given: HTML-only options passed to AssertJSON
		mt := &mockT{}
		actual := `{"name": "Alice Unsupported", "age": 32, "active": false}`

		// when: asserting with unsupported options
		testastic.AssertJSON(mt, "testdata/json/exact_match_unsupported_options.json", actual,
			testastic.IgnoreHTMLComments(), testastic.PreserveWhitespace())

		// then: the test fatals with a message listing both unsupported options
		if !mt.fatal {
			t.Error("expected fatal error for unsupported options")
		}

		expectedMsg := "testastic: unsupported options for AssertJSON: IgnoreHTMLComments, PreserveWhitespace"
		if mt.message != expectedMsg {
			t.Errorf("expected message %q, got: %q", expectedMsg, mt.message)
		}
	})

	t.Run("supported options accepted", func(t *testing.T) {
		t.Parallel()

		// given: supported options for AssertJSON
		actual := `{"name": "Alice Supported", "age": 31, "active": true}`

		// when: asserting with IgnoreFields
		// then: the test passes without unsupported option error
		testastic.AssertJSON(t, "testdata/json/exact_match_supported_options.json", actual,
			testastic.IgnoreFields("nonexistent"))
	})
}
