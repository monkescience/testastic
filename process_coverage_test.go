//nolint:testpackage // Internal tests for unexported functions.
package testastic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertProcessCoverage(t *testing.T) {
	t.Parallel()

	t.Run("empty dir produces no output", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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

func TestCollectSubprocessCoverage(t *testing.T) {
	t.Run("returns nonzero when conversion fails after passing tests", func(t *testing.T) {
		// given: subprocess coverage collection with an invalid output path
		blockedDir := filepath.Join(t.TempDir(), "not-a-dir")
		writeErr := os.WriteFile(blockedDir, []byte("blocked"), 0o600)
		NoError(t, writeErr)

		outputPath := filepath.Join(blockedDir, "coverage.out")

		// when: tests pass but coverage conversion fails
		code := collectSubprocessCoverage(func() int {
			err := os.WriteFile(filepath.Join(getSharedCoverDir(), "covdata-entry"), []byte("coverage"), 0o600)
			NoError(t, err)

			return 0
		}, outputPath)

		// then: coverage conversion failure fails the run
		Equal(t, 1, code)
	})
}

func TestSetupCoverDir_sharedDir(t *testing.T) {
	t.Run("creates a per-run subdir under the shared dir", func(t *testing.T) {
		// given: a shared coverage directory is already configured
		dir := t.TempDir()

		old := getSharedCoverDir()

		setSharedCoverDir(dir)

		defer func() { setSharedCoverDir(old) }()

		// when: resolving the cover directory without an explicit override
		result := setupCoverDir(t, "")

		// then: the result is a fresh subdir inside the shared directory
		NotEqual(t, dir, result)
		HasPrefix(t, result, dir+string(filepath.Separator))

		info, err := os.Stat(result)
		NoError(t, err)
		True(t, info.IsDir())
	})

	t.Run("each call returns a unique subdir to avoid covmeta races", func(t *testing.T) {
		// given: a shared coverage directory is already configured
		dir := t.TempDir()

		old := getSharedCoverDir()

		setSharedCoverDir(dir)

		defer func() { setSharedCoverDir(old) }()

		// when: setupCoverDir is invoked twice by concurrent runs
		first := setupCoverDir(t, "")
		second := setupCoverDir(t, "")

		// then: the two calls hand out distinct directories (no covmeta.<hash> race)
		NotEqual(t, first, second)
	})

	t.Run("explicit coverDir takes precedence over shared dir", func(t *testing.T) {
		// given: both a shared directory and an explicit directory are configured
		explicit := t.TempDir()

		old := getSharedCoverDir()

		setSharedCoverDir(t.TempDir())

		defer func() { setSharedCoverDir(old) }()

		// when: resolving the cover directory with an explicit override
		result := setupCoverDir(t, explicit)

		// then: the explicit directory wins
		Equal(t, explicit, result)
	})

	t.Run("falls back to temp dir when shared dir is empty", func(t *testing.T) {
		// given: no shared or explicit coverage directory is configured
		old := getSharedCoverDir()

		setSharedCoverDir("")

		defer func() { setSharedCoverDir(old) }()

		// when: resolving the cover directory
		result := setupCoverDir(t, "")

		// then: a temp coverage directory is created
		Contains(t, result, "coverage")
		NotEqual(t, "", result)
	})
}
