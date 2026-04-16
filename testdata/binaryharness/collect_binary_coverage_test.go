package binaryharness_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/monkescience/testastic"
)

var harnessBinary *testastic.Binary

func TestMain(m *testing.M) {
	harnessBinary = testastic.BuildBinaryMain(m, "../testcli")

	outputPath := os.Getenv("TESTASTIC_PROCESS_COVERAGE_OUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), "testastic-binary-coverage.out")
	}

	code := testastic.CollectSubprocessCoverage(m, outputPath)
	harnessBinary.Cleanup()
	os.Exit(code)
}

func TestHarnessCollectsBinaryCoverage(t *testing.T) {
	// given: a binary built in TestMain and a shared coverage collector

	// when: exercising the CLI fixture during the test run
	result := harnessBinary.Run(t, "stdout", "coverage")

	// then: the command succeeds
	testastic.Equal(t, 0, result.ExitCode)
	testastic.Equal(t, "coverage", result.Stdout)
}
