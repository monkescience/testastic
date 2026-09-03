package testastic_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/monkescience/testastic"
)

// processMockT implements testing.TB for process tests.
// Unlike mockT, it calls runtime.Goexit in Fatalf because Binary.Start
// performs real work (starting processes) after validation calls
// to tb.Fatalf. Without Goexit, execution would continue past the fatal
// into exec, overwriting the error message.
type processMockT struct {
	testing.TB
	mu       sync.Mutex
	failed   bool
	fatal    bool
	message  string
	cleanups []func()
	tempDirs []string
}

func newProcessMockT() *processMockT {
	return &processMockT{}
}

func (m *processMockT) Helper() {}

func (m *processMockT) Context() context.Context { return context.Background() }

func (m *processMockT) Fatalf(format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failed = true
	m.fatal = true
	m.message = fmt.Sprintf(format, args...)

	runtime.Goexit()
}

func (m *processMockT) Errorf(format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func (m *processMockT) Logf(string, ...any) {}

func (m *processMockT) Log(...any) {}

func (m *processMockT) Name() string { return "processMockT" }

func (m *processMockT) Failed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.failed
}

func (m *processMockT) Cleanup(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanups = append(m.cleanups, f)
}

func (m *processMockT) TempDir() string {
	dir, err := os.MkdirTemp("", "testastic-process-test-*")
	if err != nil {
		panic(fmt.Sprintf("TempDir: %v", err))
	}

	m.mu.Lock()
	m.tempDirs = append(m.tempDirs, dir)
	m.mu.Unlock()

	return dir
}

func (m *processMockT) cleanup() {
	m.mu.Lock()
	cleanups := m.cleanups
	dirs := m.tempDirs
	m.mu.Unlock()

	for _, cleanup := range slices.Backward(cleanups) {
		cleanup()
	}

	for _, d := range dirs {
		os.RemoveAll(d) //nolint:errcheck // best effort cleanup
	}
}

func (m *processMockT) result() (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.fatal, m.message
}

// runExpectingFatal runs fn in a separate goroutine so that runtime.Goexit
// from processMockT.Fatalf terminates only that goroutine.
func runExpectingFatal(fn func()) {
	done := make(chan struct{})

	go func() {
		defer close(done)

		fn()
	}()

	<-done
}

// testProcessPort tracks the next available port for test processes.
var testProcessPort = struct {
	mu   sync.Mutex
	next int
}{next: 18900}

func nextPort() int {
	testProcessPort.mu.Lock()
	defer testProcessPort.mu.Unlock()

	p := testProcessPort.next
	testProcessPort.next++

	return p
}

func doGet(t *testing.T, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	testastic.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	testastic.NoError(t, err)

	return resp
}

func TestBinaryStart(t *testing.T) {
	t.Run("serves requests after startup", func(t *testing.T) {
		// given: a service binary built for this test
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		// when: starting the service and making a request
		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		resp := doGet(t, proc.URL()+"/data")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the response matches the expected JSON
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.AssertJSON(t, "testdata/service_data.expected.json", resp.Body)
	})

	t.Run("collects coverage data", func(t *testing.T) {
		// given: a running service with coverage instrumentation
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// when: making a request and stopping the process to flush coverage
		resp := doGet(t, proc.URL()+"/data")
		resp.Body.Close() //nolint:errcheck // test cleanup
		proc.Stop()

		// then: coverage data is written and contains actual counter data
		cmd := exec.CommandContext(t.Context(), "go", "tool", "covdata", "percent", "-i="+proc.CoverDir())
		output, err := cmd.CombinedOutput()
		testastic.NoError(t, err)
		testastic.True(t, len(output) > 0)
	})

	t.Run("pre-built binary", func(t *testing.T) {
		// given: a manually built coverage-instrumented binary
		outputName := "testservice"

		if runtime.GOOS == "windows" {
			outputName = "testservice.exe"
		}

		binaryPath := filepath.Join(t.TempDir(), outputName)
		cmd := exec.CommandContext(t.Context(), "go", "build", "-cover", "-o", binaryPath, "./testdata/testservice")

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("manual build failed: %s", output)
		}

		port := nextPort()

		// when: starting from a pre-built binary
		proc := testastic.NewBinary(binaryPath).Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		resp := doGet(t, proc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the process responds successfully
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("custom ready check func", func(t *testing.T) {
		// given: a config with a custom ReadyCheckFunc
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		// when: starting with a ReadyCheckFunc
		proc := binary.Start(t.Context(), t,
			testastic.ReadyCheckFunc(func(ctx context.Context) bool {
				client := &http.Client{Timeout: 500 * time.Millisecond}

				req, err := http.NewRequestWithContext(
					ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil,
				)
				if err != nil {
					return false
				}

				resp, err := client.Do(req)
				if err != nil {
					return false
				}

				_ = resp.Body.Close()

				return resp.StatusCode == http.StatusOK
			}),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		resp := doGet(t, proc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the process is reachable
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("port-less process with ready func", func(t *testing.T) {
		// given: a config without a port, using a custom ReadyCheckFunc
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		// when: starting without Port
		proc := binary.Start(t.Context(), t,
			testastic.ReadyCheckFunc(func(ctx context.Context) bool {
				req, err := http.NewRequestWithContext(
					ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil,
				)
				if err != nil {
					return false
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return false
				}

				_ = resp.Body.Close()

				return resp.StatusCode == http.StatusOK
			}),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// then: URL returns empty and the process is running
		testastic.Equal(t, "", proc.URL())
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		// given: a running process
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// when: calling Stop twice
		proc.Stop()
		proc.Stop()

		// then: no panic occurs (test passes)
	})

	t.Run("detects early exit", func(t *testing.T) {
		// given: a process that exits immediately on startup
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		mt := newProcessMockT()
		defer mt.cleanup()

		port := nextPort()

		// when: starting the process
		runExpectingFatal(func() {
			binary.Start(context.Background(), mt,
				testastic.HTTPCheck(port, "/health"),
				testastic.WithPort(port),
				testastic.WithEnv(fmt.Sprintf("PORT=%d", port), "EXIT_EARLY=true"),
				testastic.WithReadyTimeout(5*time.Second),
			)
		})

		// then: the test fails with an early exit message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "process exited before becoming ready")
	})

	t.Run("reports build failure", func(t *testing.T) {
		// given: a config pointing to a nonexistent package
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: building the binary
		runExpectingFatal(func() {
			testastic.BuildBinary(mt, "./nonexistent/package")
		})

		// then: the test fails with a build error
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "go build failed")
	})

	t.Run("custom cover dir", func(t *testing.T) {
		// given: a config with an explicit CoverDir
		port := nextPort()
		coverDir := filepath.Join(t.TempDir(), "my-coverage")
		testastic.NoError(t, os.MkdirAll(coverDir, 0o750))
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithCoverDir(coverDir),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// when: making a request and stopping to flush coverage
		resp := doGet(t, proc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup
		proc.Stop()

		// then: the returned CoverDir matches and contains coverage data
		testastic.Equal(t, coverDir, proc.CoverDir())

		entries, err := os.ReadDir(coverDir)
		testastic.NoError(t, err)
		testastic.True(t, len(entries) > 0)
	})

	t.Run("only exposes the fixture test environment variable", func(t *testing.T) {
		// given: a process with a fixed fixture value and an inherited secret
		t.Setenv("TESTASTIC_PARENT_SECRET", "parent-only-secret")

		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port), "TESTASTIC_TEST_VALUE=hello-from-test"),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// when: trying to select the inherited secret through the fixture endpoint
		resp := doGet(t, proc.URL()+"/env?key=TESTASTIC_PARENT_SECRET")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		body, err := io.ReadAll(resp.Body)
		testastic.NoError(t, err)

		// then: only the fixed fixture value is exposed
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.Equal(t, "hello-from-test", string(body))
	})

	t.Run("args are passed to the process", func(t *testing.T) {
		// given: a config with explicit process arguments
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithArgs("--mode=test", "--feature=alpha"),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// when: querying the process for its arguments
		resp := doGet(t, proc.URL()+"/args")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		body, err := io.ReadAll(resp.Body)
		testastic.NoError(t, err)

		// then: the process reports the passed arguments
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.AssertJSON(t, "testdata/json/process_args.json", body)
	})

	t.Run("build args are passed to go build", func(t *testing.T) {
		// given: a config with build-time ldflags
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice",
			testastic.WithBuildArgs("-ldflags", "-X main.buildStamp=custom-build-stamp"),
		)

		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// when: querying the process for its build stamp
		resp := doGet(t, proc.URL()+"/build-info")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		body, err := io.ReadAll(resp.Body)
		testastic.NoError(t, err)

		// then: the build-time value is embedded in the binary
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.Equal(t, "custom-build-stamp", string(body))
	})

	t.Run("work dir is used for build and process", func(t *testing.T) {
		// given: a config with a custom working directory inside the repo
		port := nextPort()
		workDir := filepath.Join(".", "testdata")
		binary := testastic.BuildBinary(t, "./testservice",
			testastic.WithWorkDir(workDir),
		)

		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithReadyTimeout(10*time.Second),
		)

		// when: querying the process for its working directory
		resp := doGet(t, proc.URL()+"/cwd")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		body, err := io.ReadAll(resp.Body)
		testastic.NoError(t, err)

		// then: the process runs in the configured working directory
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.HasSuffix(t, string(body), string(filepath.Separator)+"testdata")
	})

	t.Run("pre-cancelled context", func(t *testing.T) {
		// given: a context that is already cancelled
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		mt := newProcessMockT()
		defer mt.cleanup()

		port := nextPort()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// when: starting the process with the cancelled context
		runExpectingFatal(func() {
			binary.Start(ctx, mt,
				testastic.HTTPCheck(port, "/health"),
				testastic.WithPort(port),
				testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
				testastic.WithReadyTimeout(2*time.Second),
			)
		})

		// then: the test fails (process never becomes ready)
		fatal, _ := mt.result()
		testastic.True(t, fatal)
	})

	t.Run("context timeout during readiness", func(t *testing.T) {
		// given: a context that expires before the process can become ready
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		mt := newProcessMockT()
		defer mt.cleanup()

		port := nextPort()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// when: starting a slow process with a short context timeout
		runExpectingFatal(func() {
			binary.Start(ctx, mt,
				testastic.HTTPCheck(port, "/health"),
				testastic.WithPort(port),
				testastic.WithEnv(fmt.Sprintf("PORT=%d", port), "SLOW_START=10s"),
				testastic.WithReadyTimeout(10*time.Second),
			)
		})

		// then: the test fails because the context expired during polling
		fatal, _ := mt.result()
		testastic.True(t, fatal)
	})

	t.Run("cleanup stops process after readiness timeout", func(t *testing.T) {
		// given: a process that starts but never satisfies readiness
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		mt := newProcessMockT()
		defer mt.cleanup()

		port := nextPort()

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		// when: startup fails during readiness polling
		runExpectingFatal(func() {
			binary.Start(ctx, mt,
				testastic.ReadyCheckFunc(func(context.Context) bool { return false }),
				testastic.WithPort(port),
				testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
				testastic.WithReadyTimeout(500*time.Millisecond),
				testastic.WithReadyInterval(10*time.Millisecond),
			)
		})

		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "process not ready")

		mt.cleanup()

		// then: the registered cleanup stopped the process without relying on parent context cancellation
		testastic.Eventually(t, func() bool {
			return !processResponds(t, port)
		}, 2*time.Second, testastic.WithInterval(25*time.Millisecond))
	})
}

func TestBinaryStartReadyTimeoutBoundsChecker(t *testing.T) {
	// given: a readiness checker that blocks after inspecting its context
	mt := newProcessMockT()
	defer mt.cleanup()

	blocked := make(chan struct{})
	defer close(blocked)

	hasDeadline := make(chan bool, 1)

	// when: starting a process with a short readiness timeout
	runExpectingFatal(func() {
		testCLI.Start(context.Background(), mt,
			testastic.ReadyCheckFunc(func(ctx context.Context) bool {
				_, ok := ctx.Deadline()
				hasDeadline <- ok

				<-blocked

				return false
			}),
			testastic.WithArgs("sleep", "2s"),
			testastic.WithReadyTimeout(100*time.Millisecond),
			testastic.WithShutdownTimeout(100*time.Millisecond),
		)
	})

	// then: startup times out and the checker received a deadline
	fatal, msg := mt.result()
	testastic.True(t, fatal)
	testastic.True(t, <-hasDeadline)
	testastic.Contains(t, msg, "process not ready after 100ms")
}

func processResponds(t *testing.T, port int) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}

	resp.Body.Close() //nolint:errcheck // test cleanup

	return resp.StatusCode == http.StatusOK
}

func TestCollectSubprocessCoverage(t *testing.T) {
	t.Run("exported helper collects subprocess coverage through TestMain", func(t *testing.T) {
		// given: a harness package that uses CollectSubprocessCoverage in TestMain
		outputPath := filepath.Join(t.TempDir(), "process.out")
		cleanupPath := filepath.Join(t.TempDir(), "cleanup.txt")
		cmd := exec.CommandContext(t.Context(),
			"go", "test", "-count=1", "-run", "^TestHarnessCollectsProcessCoverage$", "./testdata/coverageharness",
		)

		cmd.Env = append(
			os.Environ(),
			"TESTASTIC_PROCESS_COVERAGE_OUT="+outputPath,
			"TESTASTIC_PROCESS_CLEANUP_MARK="+cleanupPath,
		)

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

		cleanupContent, cleanupErr := os.ReadFile(cleanupPath)
		testastic.NoError(t, cleanupErr)
		testastic.Equal(t, "cleaned\n", string(cleanupContent))
	})

	t.Run("produces text profile from subprocess coverage", func(t *testing.T) {
		// given: a coverage output path
		outputPath := filepath.Join(t.TempDir(), "process.out")
		port := nextPort()
		binary := testastic.BuildBinary(t, "./testdata/testservice")

		// Use a mock testing.M-like flow: set shared dir, run process, convert.
		// We can't use CollectSubprocessCoverage directly (needs *testing.M),
		// so we test the same flow manually.
		sharedDir := filepath.Join(t.TempDir(), "shared-coverage")
		testastic.NoError(t, os.MkdirAll(sharedDir, 0o750))

		// when: starting a process with WithCoverDir pointing to the shared dir
		proc := binary.Start(t.Context(), t,
			testastic.HTTPCheck(port, "/health"),
			testastic.WithPort(port),
			testastic.WithEnv(fmt.Sprintf("PORT=%d", port)),
			testastic.WithCoverDir(sharedDir),
			testastic.WithReadyTimeout(10*time.Second),
		)

		resp := doGet(t, proc.URL()+"/data")
		resp.Body.Close() //nolint:errcheck // test cleanup
		proc.Stop()

		// then: converting coverage produces a valid text profile
		cmd := exec.CommandContext(t.Context(), "go", "tool", "covdata", "textfmt",
			"-i="+sharedDir, "-o="+outputPath)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("covdata textfmt failed: %s", output)
		}

		content, err := os.ReadFile(outputPath)
		testastic.NoError(t, err)
		testastic.True(t, len(content) > 0)
		testastic.HasPrefix(t, string(content), "mode:")
	})
}

func TestBinaryStart_validation(t *testing.T) {
	readyCheck := testastic.ReadyCheckFunc(func(context.Context) bool { return true })

	t.Run("rejects GOCOVERDIR in env", func(t *testing.T) {
		// given: a config that includes GOCOVERDIR in the Env slice
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testCLI.Start(context.Background(), mt,
				readyCheck,
				testastic.WithEnv("GOCOVERDIR=/tmp/cover"),
			)
		})

		// then: the test fails with a GOCOVERDIR rejection message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "must not include GOCOVERDIR")
	})

	t.Run("requires import path for BuildBinary", func(t *testing.T) {
		// given: a missing import path
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: building a binary without an import path
		runExpectingFatal(func() {
			testastic.BuildBinary(mt, "")
		})

		// then: the test fails with a missing import path message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "requires importPath")
	})

	t.Run("requires binary path for NewBinary", func(t *testing.T) {
		// given: a NewBinary with an empty path
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting a binary with an empty path
		runExpectingFatal(func() {
			testastic.NewBinary("").Start(context.Background(), mt, readyCheck)
		})

		// then: the test fails with an empty binary path message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "binary path is empty")
	})

	t.Run("requires ready check", func(t *testing.T) {
		// given: a nil ready check
		mt := newProcessMockT()
		defer mt.cleanup()

		var nilReadyCheck testastic.ReadyChecker

		// when: starting a process without a ready check
		runExpectingFatal(func() {
			testCLI.Start(context.Background(), mt, nilReadyCheck)
		})

		// then: the test fails with a missing ready check message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "requires readyCheck")
	})

	t.Run("rejects negative ready timeout", func(t *testing.T) {
		// given: a config with a negative ReadyTimeout
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testCLI.Start(context.Background(), mt,
				readyCheck,
				testastic.WithReadyTimeout(-1*time.Second),
			)
		})

		// then: the test fails with a negative timeout message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "WithReadyTimeout must not be negative")
	})

	t.Run("rejects negative ready interval", func(t *testing.T) {
		// given: a config with a negative ReadyInterval
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testCLI.Start(context.Background(), mt,
				readyCheck,
				testastic.WithReadyInterval(-100*time.Millisecond),
			)
		})

		// then: the test fails with a negative interval message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "WithReadyInterval must not be negative")
	})

	t.Run("rejects negative shutdown timeout", func(t *testing.T) {
		// given: a config with a negative ShutdownTimeout
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testCLI.Start(context.Background(), mt,
				readyCheck,
				testastic.WithShutdownTimeout(-1*time.Second),
			)
		})

		// then: the test fails with a negative timeout message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "WithShutdownTimeout must not be negative")
	})
}
