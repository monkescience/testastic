//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateExpectedFile(t *testing.T) {
	t.Parallel()

	t.Run("creates file with pretty-printed JSON", func(t *testing.T) {
		t.Parallel()

		// given: valid JSON data
		dir := t.TempDir()
		path := filepath.Join(dir, "output.json")
		data := []byte(`{"name":"Alice","age":30}`)

		// when: creating expected file
		err := createExpectedFile(path, data)
		// then: file is created with formatted JSON
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("failed to read file: %v", readErr)
		}

		got := string(content)

		if !strings.Contains(got, "  ") {
			t.Error("expected indented JSON")
		}

		if !strings.HasSuffix(got, "\n") {
			t.Error("expected trailing newline")
		}

		if !strings.Contains(got, `"name"`) {
			t.Error("expected name field in output")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		t.Parallel()

		// given: path with non-existent parent directories
		dir := t.TempDir()
		path := filepath.Join(dir, "sub", "dir", "output.json")
		data := []byte(`{"key":"value"}`)

		// when: creating expected file
		err := createExpectedFile(path, data)
		// then: file is created successfully
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, statErr := os.Stat(path)
		if os.IsNotExist(statErr) {
			t.Error("expected file to exist")
		}
	})

	t.Run("preserves large integers", func(t *testing.T) {
		t.Parallel()

		// given: JSON data containing an integer that cannot round-trip through float64
		path := filepath.Join(t.TempDir(), "output.json")
		data := []byte(`{"id":9007199254740993}`)

		// when: creating expected file
		err := createExpectedFile(path, data)
		// then: the integer is written exactly as supplied
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("failed to read file: %v", readErr)
		}

		if !strings.Contains(string(content), `9007199254740993`) {
			t.Errorf("expected large integer to be preserved, got:\n%s", content)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		t.Parallel()

		// given: invalid JSON data
		dir := t.TempDir()
		path := filepath.Join(dir, "output.json")
		data := []byte(`{invalid json}`)

		// when: creating expected file
		err := createExpectedFile(path, data)

		// then: returns error
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestUpdateExpectedFile(t *testing.T) {
	t.Parallel()

	t.Run("preserves large integers", func(t *testing.T) {
		t.Parallel()

		// given: actual JSON data containing an integer that cannot round-trip through float64
		path := filepath.Join(t.TempDir(), "output.json")
		data := []byte(`{"id":9007199254740993}`)
		expected := &expectedJSON{}

		// when: updating expected file
		err := updateExpectedFile(path, data, expected)
		// then: the integer is written exactly as supplied
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("failed to read file: %v", readErr)
		}

		if !strings.Contains(string(content), `9007199254740993`) {
			t.Errorf("expected large integer to be preserved, got:\n%s", content)
		}
	})
}

func TestGenerateUpdatedJSON(t *testing.T) {
	t.Parallel()

	t.Run("no matchers", func(t *testing.T) {
		t.Parallel()

		// given: data with no matcher positions
		data := map[string]any{"name": "Alice"}

		// when: generating updated JSON
		result, err := generateUpdatedJSON(data, nil)
		// then: produces pretty JSON with trailing newline
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.HasSuffix(result, "\n") {
			t.Error("expected trailing newline")
		}

		if !strings.Contains(result, `"name"`) {
			t.Error("expected name field in output")
		}
	})

	t.Run("with matcher positions", func(t *testing.T) {
		t.Parallel()

		// given: data with a matcher position to preserve
		data := map[string]any{"id": "abc-123", "name": "Alice"}
		matchers := map[string]string{
			"$.id": "{{anyString}}",
		}

		// when: generating updated JSON
		result, err := generateUpdatedJSON(data, matchers)
		// then: matcher is preserved in output
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "{{anyString}}") {
			t.Error("expected matcher to be preserved in output")
		}
	})

	t.Run("preserves only targeted path with duplicate keys", func(t *testing.T) {
		t.Parallel()

		// given: data with duplicate key names at different nesting levels
		data := map[string]any{
			"user":    map[string]any{"id": "user-1", "name": "Alice"},
			"company": map[string]any{"id": "comp-1", "name": "Acme"},
		}
		matchers := map[string]string{
			"$.user.id": "{{anyString}}",
		}

		// when: generating updated JSON
		result, err := generateUpdatedJSON(data, matchers)
		// then: only user.id has the matcher, company.id is unchanged
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, `"{{anyString}}"`) {
			t.Error("expected matcher for user.id in output")
		}

		if !strings.Contains(result, `"comp-1"`) {
			t.Errorf("expected company.id to be preserved, got:\n%s", result)
		}
	})

	t.Run("empty matcher positions", func(t *testing.T) {
		t.Parallel()

		// given: empty matcher map
		data := map[string]any{"name": "Alice"}

		// when: generating updated JSON
		result, err := generateUpdatedJSON(data, map[string]string{})
		// then: produces normal JSON
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, `"Alice"`) {
			t.Error("expected Alice in output")
		}
	})
}

func TestReplaceValueAtPath(t *testing.T) {
	t.Parallel()

	t.Run("replaces string value", func(t *testing.T) {
		t.Parallel()

		// given: JSON with a string value
		jsonStr := `{"id": "abc-123"}`

		// when: replacing value at path
		result := replaceValueAtPath(jsonStr, "$.id", "{{anyString}}")

		// then: value is replaced with matcher
		if !strings.Contains(result, "{{anyString}}") {
			t.Errorf("expected matcher in result, got %q", result)
		}
	})

	t.Run("replaces numeric value", func(t *testing.T) {
		t.Parallel()

		// given: JSON with a numeric value
		jsonStr := `{"count": 42}`

		// when: replacing value at path
		result := replaceValueAtPath(jsonStr, "$.count", "{{anyInt}}")

		// then: value is replaced with matcher
		if !strings.Contains(result, "{{anyInt}}") {
			t.Errorf("expected matcher in result, got %q", result)
		}
	})

	t.Run("replaces boolean value", func(t *testing.T) {
		t.Parallel()

		// given: JSON with a boolean value
		jsonStr := `{"active": true}`

		// when: replacing value at path
		result := replaceValueAtPath(jsonStr, "$.active", "{{anyBool}}")

		// then: value is replaced with matcher
		if !strings.Contains(result, "{{anyBool}}") {
			t.Errorf("expected matcher in result, got %q", result)
		}
	})

	t.Run("empty path returns unchanged", func(t *testing.T) {
		t.Parallel()

		// given: empty path
		jsonStr := `{"id": "abc"}`

		// when: replacing with empty path
		result := replaceValueAtPath(jsonStr, "", "{{anyString}}")

		// then: unchanged
		if result != jsonStr {
			t.Errorf("expected unchanged JSON, got %q", result)
		}
	})

	t.Run("handles array index in path", func(t *testing.T) {
		t.Parallel()

		// given: path with array index
		jsonStr := `{"items": [{"id": "abc"}]}`

		// when: replacing value at array path
		result := replaceValueAtPath(jsonStr, "$.items[0].id", "{{anyString}}")

		// then: value is replaced
		if !strings.Contains(result, "{{anyString}}") {
			t.Errorf("expected matcher in result, got %q", result)
		}
	})

	t.Run("only replaces value at target path not duplicates", func(t *testing.T) {
		t.Parallel()

		// given: JSON with duplicate key names at different paths
		jsonStr := `{
  "user": {
    "name": "Alice"
  },
  "company": {
    "name": "Acme"
  }
}`

		// when: replacing only the user name
		result := replaceValueAtPath(jsonStr, "$.user.name", "{{anyString}}")

		// then: only user.name is replaced, company.name is preserved
		if !strings.Contains(result, `"{{anyString}}"`) {
			t.Errorf("expected matcher in result, got %q", result)
		}

		if !strings.Contains(result, `"Acme"`) {
			t.Errorf("expected company name to be preserved, got %q", result)
		}
	})
}

func TestShouldUpdate(t *testing.T) {
	t.Run("TESTASTIC_UPDATE=true", func(t *testing.T) {
		// given: env var set to true
		t.Setenv("TESTASTIC_UPDATE", "true")

		// when: checking whether update mode is enabled
		// then: update mode is enabled
		if !shouldUpdate() {
			t.Error("expected shouldUpdate to return true")
		}
	})

	t.Run("TESTASTIC_UPDATE=1", func(t *testing.T) {
		// given: env var set to 1
		t.Setenv("TESTASTIC_UPDATE", "1")

		// when: checking whether update mode is enabled
		// then: update mode is enabled
		if !shouldUpdate() {
			t.Error("expected shouldUpdate to return true")
		}
	})

	t.Run("TESTASTIC_UPDATE=TRUE", func(t *testing.T) {
		// given: env var set to TRUE (uppercase)
		t.Setenv("TESTASTIC_UPDATE", "TRUE")

		// when: checking whether update mode is enabled
		// then: update mode is enabled case-insensitively
		if !shouldUpdate() {
			t.Error("expected shouldUpdate to return true")
		}
	})

	t.Run("TESTASTIC_UPDATE=false", func(t *testing.T) {
		// given: env var set to false
		t.Setenv("TESTASTIC_UPDATE", "false")

		// when: checking whether update mode is enabled
		// then: update mode remains disabled
		if shouldUpdate() {
			t.Error("expected shouldUpdate to return false")
		}
	})

	t.Run("TESTASTIC_UPDATE unset", func(t *testing.T) {
		// given: env var is not set
		t.Setenv("TESTASTIC_UPDATE", "")
		os.Unsetenv("TESTASTIC_UPDATE") //nolint:errcheck // best-effort cleanup in test

		// when: checking whether update mode is enabled
		// then: update mode stays disabled without env or flags
		// Note: this test runs in go test context which does not have -update flag
		result := shouldUpdate()

		if result {
			t.Error("expected shouldUpdate to return false when env unset and no flag")
		}
	})
}
