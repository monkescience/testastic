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

func TestMain(m *testing.M) {
	outputPath := os.Getenv("TESTASTIC_PROCESS_COVERAGE_OUT")
	if outputPath == "" {
		outputPath = filepath.Join(os.TempDir(), "testastic-process-coverage.out")
	}

	os.Exit(testastic.CollectProcessCoverage(m, outputPath))
}

func TestHarnessCollectsProcessCoverage(t *testing.T) {
	// given: a process started under a package TestMain that collects coverage
	port := 19100
	proc := testastic.StartProcess(t.Context(), t,
		"../testservice",
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
