package testastic_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/monkescience/testastic"
)

func TestBuildBinary(t *testing.T) {
	t.Run("builds a reusable binary during a regular test", func(t *testing.T) {
		// given: a CLI fixture import path
		binary := testastic.BuildBinary(t, "./testdata/testcli")

		// when: running the built binary
		result := binary.Run(t, "stdout", "built in test")

		// then: the reusable binary executes successfully
		testastic.Equal(t, 0, result.ExitCode)
		testastic.Equal(t, "built in test", result.Stdout)
	})
}

func TestBinaryRun(t *testing.T) {
	t.Run("captures stdout on success", func(t *testing.T) {
		// given: a coverage-instrumented CLI fixture built in TestMain

		// when: running the command successfully
		result := testCLI.Run(t, "stdout", "binary success output")

		// then: stdout is captured and the exit code is zero
		testastic.Equal(t, 0, result.ExitCode)
		testastic.Equal(t, "binary success output", result.Stdout)
		testastic.Equal(t, "", result.Stderr)
	})

	t.Run("captures non-zero exit without failing the test", func(t *testing.T) {
		// given: a CLI command that exits with a non-zero code

		// when: running the command
		result := testCLI.Run(t, "fail", "7", "binary failure output")

		// then: stderr and the exit code are available to the caller
		testastic.Equal(t, 7, result.ExitCode)
		testastic.Equal(t, "", result.Stdout)
		testastic.Equal(t, "binary failure output", result.Stderr)
	})

	t.Run("passes stdin to the process", func(t *testing.T) {
		// given: a CLI command that reads from stdin
		input := strings.NewReader("stdin payload")

		// when: running with stdin configured
		result := testCLI.RunWithOptions(t, []string{"stdin"}, testastic.WithStdin(input))

		// then: the process output reflects the provided stdin
		testastic.Equal(t, 0, result.ExitCode)
		testastic.Equal(t, "stdin payload", result.Stdout)
		testastic.Equal(t, "", result.Stderr)
	})

	t.Run("sets per-run environment variables", func(t *testing.T) {
		// given: a CLI command that reads an environment variable

		// when: running with a per-invocation environment override
		result := testCLI.RunWithOptions(t, []string{"env", "BINARY_TEST_VALUE"},
			testastic.WithRunEnv("BINARY_TEST_VALUE=env success value"),
		)

		// then: the process sees the configured environment variable
		testastic.Equal(t, 0, result.ExitCode)
		testastic.Equal(t, "env success value", result.Stdout)
	})

	t.Run("uses the build work dir by default", func(t *testing.T) {
		// given: a CLI command that prints its working directory
		repoRoot, err := os.Getwd()
		testastic.NoError(t, err)

		// when: running without an explicit work dir override
		result := testCLI.Run(t, "cwd")

		// then: the command runs from the same default directory used at build time
		testastic.Equal(t, 0, result.ExitCode)
		testastic.Equal(t, repoRoot, result.Stdout)
	})

	t.Run("overrides the work dir per run", func(t *testing.T) {
		// given: a CLI command and a custom working directory
		customDir := t.TempDir()
		resolvedCustomDir, err := filepath.EvalSymlinks(customDir)
		testastic.NoError(t, err)

		// when: running with an explicit work dir override
		result := testCLI.RunWithOptions(t, []string{"cwd"}, testastic.WithRunWorkDir(customDir))

		// then: the command runs from the requested directory
		testastic.Equal(t, 0, result.ExitCode)
		testastic.Equal(t, resolvedCustomDir, result.Stdout)
	})

	t.Run("times out on hung commands", func(t *testing.T) {
		// given: a CLI command that sleeps longer than the configured timeout
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: running with a short timeout
		runExpectingFatal(func() {
			testCLI.RunWithOptions(mt, []string{"sleep", "2s"}, testastic.WithRunTimeout(50*time.Millisecond))
		})

		// then: the test fails with a timeout message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "timed out")
	})

	t.Run("rejects GOCOVERDIR in run env", func(t *testing.T) {
		// given: an env override that tries to replace the coverage directory
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: running with a forbidden env key
		runExpectingFatal(func() {
			testCLI.RunWithOptions(mt, []string{"stdout", "ignored"}, testastic.WithRunEnv("GOCOVERDIR=/tmp/cover"))
		})

		// then: the test fails with a validation error
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "must not include GOCOVERDIR")
	})
}

func TestCollectBinaryCoverage(t *testing.T) {
	t.Run("exported helper collects CLI coverage through TestMain", func(t *testing.T) {
		// given: a harness package that uses BuildBinaryMain and CollectSubprocessCoverage
		outputPath := filepath.Join(t.TempDir(), "binary.out")
		cmd := exec.CommandContext(t.Context(),
			"go", "test", "-count=1", "-run", "^TestHarnessCollectsBinaryCoverage$", "./testdata/binaryharness",
		)

		cmd.Env = append(os.Environ(), "TESTASTIC_PROCESS_COVERAGE_OUT="+outputPath)

		// when: running the harness package test suite
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go test harness failed: %s", output)
		}

		// then: the exported helper produces a valid text coverage profile
		content, readErr := os.ReadFile(outputPath)
		testastic.NoError(t, readErr)
		testastic.True(t, len(content) > 0)
		testastic.HasPrefix(t, string(content), "mode:")
	})
}

func TestRunWithOptions_usesTestContext(t *testing.T) {
	t.Run("inherits cancellation from the test context", func(t *testing.T) {
		// given: a command launched with a cancelled context-backed test double
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mt := newRunMockT(ctx)
		defer mt.cleanup()

		// when: running the binary after the context is already cancelled
		runExpectingFatal(func() {
			testCLI.Run(mt, "sleep", "10ms")
		})

		// then: the run fails as an infrastructure error instead of returning an exit code
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "context canceled")
	})
}

type runMockT struct {
	*processMockT
	contextFunc func() context.Context
}

func newRunMockT(ctx context.Context) *runMockT {
	return &runMockT{
		processMockT: newProcessMockT(),
		contextFunc:  func() context.Context { return ctx },
	}
}

func (m *runMockT) Context() context.Context {
	return m.contextFunc()
}
