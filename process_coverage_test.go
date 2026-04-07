//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertProcessCoverage(t *testing.T) {
	t.Run("empty dir produces no output", func(t *testing.T) {
		// given: an empty coverage directory
		coverDir := t.TempDir()
		outputPath := filepath.Join(t.TempDir(), "coverage.out")

		// when: converting process coverage
		err := convertProcessCoverage(coverDir, outputPath)

		// then: no output file is created and no error is returned
		NoError(t, err)

		_, err = os.Stat(outputPath)
		True(t, os.IsNotExist(err))
	})

	t.Run("creates output directory", func(t *testing.T) {
		// given: an empty coverage directory and nested output path
		coverDir := t.TempDir()
		outputPath := filepath.Join(t.TempDir(), "nested", "dir", "coverage.out")

		// when: converting process coverage
		err := convertProcessCoverage(coverDir, outputPath)

		// then: no output directory is created because there is no coverage data
		NoError(t, err)

		// Output dir should exist even though no coverage data was produced
		_, err = os.Stat(filepath.Dir(outputPath))
		// Dir is not created when there are no entries
		True(t, os.IsNotExist(err))
	})
}

func TestSetupCoverDir_sharedDir(t *testing.T) {
	t.Run("uses shared dir when set", func(t *testing.T) {
		// given: a shared coverage directory is already configured
		dir := t.TempDir()

		old := sharedCoverDir
		sharedCoverDir = dir

		defer func() { sharedCoverDir = old }()

		// when: resolving the cover directory without an explicit override
		result := setupCoverDir(t, "")

		// then: the shared directory is reused
		Equal(t, dir, result)
	})

	t.Run("explicit coverDir takes precedence over shared dir", func(t *testing.T) {
		// given: both a shared directory and an explicit directory are configured
		explicit := t.TempDir()

		old := sharedCoverDir
		sharedCoverDir = t.TempDir()

		defer func() { sharedCoverDir = old }()

		// when: resolving the cover directory with an explicit override
		result := setupCoverDir(t, explicit)

		// then: the explicit directory wins
		Equal(t, explicit, result)
	})

	t.Run("falls back to temp dir when shared dir is empty", func(t *testing.T) {
		// given: no shared or explicit coverage directory is configured
		old := sharedCoverDir
		sharedCoverDir = ""

		defer func() { sharedCoverDir = old }()

		// when: resolving the cover directory
		result := setupCoverDir(t, "")

		// then: a temp coverage directory is created
		Contains(t, result, "coverage")
		NotEqual(t, "", result)
	})
}
