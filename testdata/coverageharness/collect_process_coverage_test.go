package coverageharness_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/monkescience/testastic"
)

var serviceBinary *testastic.Binary

func TestMain(m *testing.M) {
	serviceBinary = testastic.BuildBinaryMain(m, "../testservice")

	outputPath := os.Getenv("TESTASTIC_PROCESS_COVERAGE_OUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), "testastic-process-coverage.out")
	}

	exitCode := testastic.CollectSubprocessCoverage(m, outputPath)

	cleanupPath := os.Getenv("TESTASTIC_PROCESS_CLEANUP_MARK")
	if cleanupPath != "" {
		err := os.WriteFile(cleanupPath, []byte("cleaned\n"), 0o600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "testastic: failed to write cleanup marker: %v\n", err)

			exitCode = 1
		}
	}

	serviceBinary.Cleanup()
	os.Exit(exitCode)
}

func TestHarnessCollectsProcessCoverage(t *testing.T) {
	// given: a service binary built in TestMain with coverage collection
	port := 19100
	proc := serviceBinary.Start(t.Context(), t,
		testastic.HTTPCheck(port, "/health"),
		testastic.WithPort(port),
		testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
		testastic.WithReadyTimeout(10*time.Second),
	)

	// when: exercising the subprocess before test shutdown
	resp, err := http.Get(proc.URL() + "/data")
	testastic.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	// then: the subprocess responds successfully
	testastic.Equal(t, http.StatusOK, resp.StatusCode)
}
