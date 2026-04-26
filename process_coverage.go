package testastic

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// sharedCoverDir is the package-level shared coverage directory, set by
// [CollectSubprocessCoverage]. When set, all subprocesses started without
// [WithCoverDir] write their coverage data here instead of a per-test
// temp directory.
//
// No synchronization is needed: CollectSubprocessCoverage is called from
// TestMain before m.Run, and all reads happen during m.Run. Go's test runner
// guarantees that TestMain runs single-threaded before and after m.Run.
var sharedCoverDir string

// errCovdataFailed is returned when `go tool covdata textfmt` fails.
var errCovdataFailed = errors.New("go tool covdata textfmt failed")

// CollectSubprocessCoverage runs all tests via m.Run and collects coverage data
// from subprocesses started with [Binary.Run] or [Binary.Start] into a single
// text profile at outputPath.
//
// The text profile uses Go's standard coverage format, compatible with
// `go tool cover`, codecov, coveralls, and other coverage tools.
//
// Call this from TestMain:
//
//	func TestMain(m *testing.M) {
//	    bin := testastic.BuildBinaryMain(m, "./cmd/api")
//	    code := testastic.CollectSubprocessCoverage(m, "coverage/process.out")
//	    bin.Cleanup()
//	    os.Exit(code)
//	}
//
// Package-level cleanup that must run before process exit should happen after
// CollectSubprocessCoverage returns and before calling [os.Exit].
//
// Without CollectSubprocessCoverage, coverage data is written to per-test temp
// directories and cleaned up automatically. Processes that use [WithCoverDir]
// are not affected — their coverage data goes to the specified directory.
func CollectSubprocessCoverage(m *testing.M, outputPath string) int {
	return collectSubprocessCoverage(m.Run, outputPath)
}

func collectSubprocessCoverage(run func() int, outputPath string) int {
	dir, err := os.MkdirTemp("", "testastic-coverage-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "testastic: failed to create shared coverage directory: %v\n", err)

		return 1
	}

	sharedCoverDir = dir

	code := run()

	err = convertProcessCoverage(dir, outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testastic: %v\n", err)

		if code == 0 {
			code = 1
		}
	}

	_ = os.RemoveAll(dir)

	sharedCoverDir = ""

	return code
}

// convertProcessCoverage converts binary coverage data in coverDir to a text
// profile at outputPath using `go tool covdata textfmt`.
func convertProcessCoverage(coverDir string, outputPath string) error {
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		return fmt.Errorf("failed to read coverage directory: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	const dirPerm = 0o750

	err = os.MkdirAll(filepath.Dir(outputPath), dirPerm)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	const covdataTimeout = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), covdataTimeout)
	defer cancel()

	//nolint:gosec // paths from test config
	cmd := exec.CommandContext(ctx,
		"go", "tool", "covdata", "textfmt",
		"-i="+coverDir, "-o="+outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", errCovdataFailed, output)
	}

	return nil
}
