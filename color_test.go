//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"os"
	"testing"
)

func TestDetectColors(t *testing.T) {
	// Save and restore all relevant env vars.
	envVars := []string{"NO_COLOR", "FORCE_COLOR", "CI", "TERM"}
	saved := make(map[string]string, len(envVars))

	for _, key := range envVars {
		saved[key] = os.Getenv(key)
	}

	t.Cleanup(func() {
		for _, key := range envVars {
			if saved[key] == "" {
				t.Setenv(key, "")
				os.Unsetenv(key) //nolint:errcheck // best-effort cleanup
			} else {
				t.Setenv(key, saved[key])
			}
		}
	})

	clearEnv := func() {
		for _, key := range envVars {
			os.Unsetenv(key) //nolint:errcheck // best-effort cleanup in test helper
		}
	}

	t.Run("NO_COLOR disables colors", func(t *testing.T) {
		// given: NO_COLOR is set
		clearEnv()
		t.Setenv("NO_COLOR", "1")

		// when: detecting colors
		result := detectColors()

		// then: colors are disabled
		if result {
			t.Error("expected colors to be disabled with NO_COLOR set")
		}
	})

	t.Run("FORCE_COLOR enables colors", func(t *testing.T) {
		// given: FORCE_COLOR is set (and NO_COLOR is not)
		clearEnv()
		t.Setenv("FORCE_COLOR", "1")

		// when: detecting colors
		result := detectColors()

		// then: colors are enabled
		if !result {
			t.Error("expected colors to be enabled with FORCE_COLOR set")
		}
	})

	t.Run("NO_COLOR takes precedence over FORCE_COLOR", func(t *testing.T) {
		// given: both NO_COLOR and FORCE_COLOR are set
		clearEnv()
		t.Setenv("NO_COLOR", "1")
		t.Setenv("FORCE_COLOR", "1")

		// when: detecting colors
		result := detectColors()

		// then: colors are disabled (NO_COLOR wins)
		if result {
			t.Error("expected NO_COLOR to take precedence over FORCE_COLOR")
		}
	})

	t.Run("CI disables colors", func(t *testing.T) {
		// given: CI is set
		clearEnv()
		t.Setenv("CI", "true")

		// when: detecting colors
		result := detectColors()

		// then: colors are disabled
		if result {
			t.Error("expected colors to be disabled in CI")
		}
	})

	t.Run("TERM=dumb disables colors", func(t *testing.T) {
		// given: TERM=dumb
		clearEnv()
		t.Setenv("TERM", "dumb")

		// when: detecting colors
		result := detectColors()

		// then: colors are disabled
		if result {
			t.Error("expected colors to be disabled with TERM=dumb")
		}
	})
}

func TestColorize(t *testing.T) {
	// Save and restore the useColors function so each subtest can stub it.
	// TestColorize is serial (no t.Parallel), so swapping the variable is
	// safe, parallel tests in the package only run after this one finishes.
	savedUseColors := useColors

	t.Cleanup(func() {
		useColors = savedUseColors
	})

	t.Run("with colors disabled", func(t *testing.T) {
		// given: colors are disabled
		useColors = func() bool { return false }

		// when: colorizing text
		result := colorize("hello", colorRed)

		// then: returns plain text
		if result != "hello" {
			t.Errorf("expected plain text, got %q", result)
		}
	})

	t.Run("with colors enabled", func(t *testing.T) {
		// given: colors are enabled
		useColors = func() bool { return true }

		// when: colorizing text
		result := colorize("hello", colorRed)

		// then: returns text wrapped in ANSI codes
		expected := colorRed + "hello" + colorReset

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("red helper", func(t *testing.T) {
		// given: colors are enabled
		useColors = func() bool { return true }

		// when: using red helper
		result := red("error")

		// then: wrapped in red ANSI codes
		expected := colorRed + "error" + colorReset

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("green helper", func(t *testing.T) {
		// given: colors are enabled
		useColors = func() bool { return true }

		// when: using green helper
		result := green("success")

		// then: wrapped in green ANSI codes
		expected := colorGreen + "success" + colorReset

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}
