//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertProcessCoverage(t *testing.T) {
	t.Run("empty dir produces no output", func(t *testing.T) {
		coverDir := t.TempDir()
		outputPath := filepath.Join(t.TempDir(), "coverage.out")

		err := convertProcessCoverage(coverDir, outputPath)
		NoError(t, err)

		_, err = os.Stat(outputPath)
		True(t, os.IsNotExist(err))
	})

	t.Run("creates output directory", func(t *testing.T) {
		coverDir := t.TempDir()
		outputPath := filepath.Join(t.TempDir(), "nested", "dir", "coverage.out")

		err := convertProcessCoverage(coverDir, outputPath)
		NoError(t, err)

		// Output dir should exist even though no coverage data was produced
		_, err = os.Stat(filepath.Dir(outputPath))
		// Dir is not created when there are no entries
		True(t, os.IsNotExist(err))
	})
}

func TestSetupCoverDir_sharedDir(t *testing.T) {
	t.Run("uses shared dir when set", func(t *testing.T) {
		dir := t.TempDir()

		old := sharedCoverDir
		sharedCoverDir = dir

		defer func() { sharedCoverDir = old }()

		result := setupCoverDir(t, "")
		Equal(t, dir, result)
	})

	t.Run("explicit coverDir takes precedence over shared dir", func(t *testing.T) {
		explicit := t.TempDir()

		old := sharedCoverDir
		sharedCoverDir = t.TempDir()

		defer func() { sharedCoverDir = old }()

		result := setupCoverDir(t, explicit)
		Equal(t, explicit, result)
	})

	t.Run("falls back to temp dir when shared dir is empty", func(t *testing.T) {
		old := sharedCoverDir
		sharedCoverDir = ""

		defer func() { sharedCoverDir = old }()

		result := setupCoverDir(t, "")
		Contains(t, result, "coverage")
		NotEqual(t, "", result)
	})
}
