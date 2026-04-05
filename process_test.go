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
	"sync"
	"testing"
	"time"

	"github.com/monkescience/testastic"
)

// processMockT implements testing.TB for StartProcess tests.
// Unlike mockT, it calls runtime.Goexit in Fatalf because StartProcess
// performs real work (building, starting processes) after validation calls
// to tb.Fatalf. Without Goexit, execution would continue past the fatal
// into build/exec, overwriting the error message.
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

	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
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

func TestStartProcess(t *testing.T) {
	t.Run("serves requests after startup", func(t *testing.T) {
		// given: a valid process config with an import path
		port := nextPort()

		// when: starting the process and making a request
		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			ImportPath:   "./testdata/testservice",
			Port:         port,
			ReadyCheck:   testastic.HTTPCheck(port, "/health"),
			Env:          []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout: 10 * time.Second,
		})

		resp := doGet(t, proc.URL()+"/data")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the response matches the expected JSON
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.AssertJSON(t, "testdata/service_data.expected.json", resp.Body)
	})

	t.Run("collects coverage data", func(t *testing.T) {
		// given: a running process with coverage instrumentation
		port := nextPort()
		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			ImportPath:   "./testdata/testservice",
			Port:         port,
			ReadyCheck:   testastic.HTTPCheck(port, "/health"),
			Env:          []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout: 10 * time.Second,
		})

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

		// when: starting with BinaryPath instead of ImportPath
		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			BinaryPath:   binaryPath,
			Port:         port,
			ReadyCheck:   testastic.HTTPCheck(port, "/health"),
			Env:          []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout: 10 * time.Second,
		})

		resp := doGet(t, proc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the process responds successfully
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("custom ready check func", func(t *testing.T) {
		// given: a config with a custom ReadyCheckFunc
		port := nextPort()

		// when: starting with a ReadyCheckFunc
		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			ImportPath: "./testdata/testservice",
			Port:       port,
			Env:        []string{fmt.Sprintf("PORT=%d", port)},
			ReadyCheck: testastic.ReadyCheckFunc(func(ctx context.Context) bool {
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
			ReadyTimeout: 10 * time.Second,
		})

		resp := doGet(t, proc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the process is reachable
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("port-less process with ready func", func(t *testing.T) {
		// given: a config without a port, using a custom ReadyCheckFunc
		port := nextPort()

		// when: starting without Port
		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			ImportPath: "./testdata/testservice",
			Env:        []string{fmt.Sprintf("PORT=%d", port)},
			ReadyCheck: testastic.ReadyCheckFunc(func(ctx context.Context) bool {
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
			ReadyTimeout: 10 * time.Second,
		})

		// then: URL returns empty and the process is running
		testastic.Equal(t, "", proc.URL())
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		// given: a running process
		port := nextPort()
		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			ImportPath:   "./testdata/testservice",
			Port:         port,
			ReadyCheck:   testastic.HTTPCheck(port, "/health"),
			Env:          []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout: 10 * time.Second,
		})

		// when: calling Stop twice
		proc.Stop()
		proc.Stop()

		// then: no panic occurs (test passes)
	})

	t.Run("detects early exit", func(t *testing.T) {
		// given: a process that exits immediately on startup
		mt := newProcessMockT()
		defer mt.cleanup()

		port := nextPort()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath:   "./testdata/testservice",
				Port:         port,
				ReadyCheck:   testastic.HTTPCheck(port, "/health"),
				Env:          []string{fmt.Sprintf("PORT=%d", port), "EXIT_EARLY=true"},
				ReadyTimeout: 5 * time.Second,
			})
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

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath: "./nonexistent/package",
				ReadyCheck: testastic.ReadyCheckFunc(func(context.Context) bool { return true }),
			})
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

		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			ImportPath:   "./testdata/testservice",
			Port:         port,
			ReadyCheck:   testastic.HTTPCheck(port, "/health"),
			Env:          []string{fmt.Sprintf("PORT=%d", port)},
			CoverDir:     coverDir,
			ReadyTimeout: 10 * time.Second,
		})

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

	t.Run("custom env passed to process", func(t *testing.T) {
		// given: a config with a custom environment variable
		port := nextPort()
		proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
			ImportPath:   "./testdata/testservice",
			Port:         port,
			ReadyCheck:   testastic.HTTPCheck(port, "/health"),
			Env:          []string{fmt.Sprintf("PORT=%d", port), "MY_TEST_VAR=hello-from-test"},
			ReadyTimeout: 10 * time.Second,
		})

		// when: querying the process for the env var value
		resp := doGet(t, proc.URL()+"/env?key=MY_TEST_VAR")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		body, err := io.ReadAll(resp.Body)
		testastic.NoError(t, err)

		// then: the process sees the custom env var
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.Equal(t, "hello-from-test", string(body))
	})

	t.Run("pre-cancelled context", func(t *testing.T) {
		// given: a context that is already cancelled
		mt := newProcessMockT()
		defer mt.cleanup()

		port := nextPort()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// when: starting the process with the cancelled context
		runExpectingFatal(func() {
			testastic.StartProcess(ctx, mt, testastic.ProcessConfig{
				ImportPath:   "./testdata/testservice",
				Port:         port,
				ReadyCheck:   testastic.HTTPCheck(port, "/health"),
				Env:          []string{fmt.Sprintf("PORT=%d", port)},
				ReadyTimeout: 2 * time.Second,
			})
		})

		// then: the test fails (process never becomes ready)
		fatal, _ := mt.result()
		testastic.True(t, fatal)
	})

	t.Run("context timeout during readiness", func(t *testing.T) {
		// given: a context that expires before the process can become ready
		mt := newProcessMockT()
		defer mt.cleanup()

		port := nextPort()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// when: starting a slow process with a short context timeout
		runExpectingFatal(func() {
			testastic.StartProcess(ctx, mt, testastic.ProcessConfig{
				ImportPath:   "./testdata/testservice",
				Port:         port,
				ReadyCheck:   testastic.HTTPCheck(port, "/health"),
				Env:          []string{fmt.Sprintf("PORT=%d", port), "SLOW_START=10s"},
				ReadyTimeout: 10 * time.Second,
			})
		})

		// then: the test fails because the context expired during polling
		fatal, _ := mt.result()
		testastic.True(t, fatal)
	})
}

func TestStartProcess_validation(t *testing.T) {
	t.Run("requires import or binary path", func(t *testing.T) {
		// given: a config with neither ImportPath nor BinaryPath
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ReadyCheck: testastic.ReadyCheckFunc(func(context.Context) bool { return true }),
			})
		})

		// then: the test fails with a missing path message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "requires ImportPath or BinaryPath")
	})

	t.Run("rejects both import and binary path", func(t *testing.T) {
		// given: a config with both ImportPath and BinaryPath set
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath: "./cmd/foo",
				BinaryPath: "/bin/foo",
				ReadyCheck: testastic.ReadyCheckFunc(func(context.Context) bool { return true }),
			})
		})

		// then: the test fails with a mutual exclusivity message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "must not set both ImportPath and BinaryPath")
	})

	t.Run("requires ready check", func(t *testing.T) {
		// given: a config without a ReadyCheck
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath: "./cmd/foo",
			})
		})

		// then: the test fails with a missing ready check message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "requires ReadyCheck")
	})

	t.Run("rejects GOCOVERDIR in env", func(t *testing.T) {
		// given: a config that includes GOCOVERDIR in the Env slice
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath: "./cmd/foo",
				ReadyCheck: testastic.ReadyCheckFunc(func(context.Context) bool { return true }),
				Env:        []string{"GOCOVERDIR=/tmp/cover"},
			})
		})

		// then: the test fails with a GOCOVERDIR rejection message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "must not include GOCOVERDIR")
	})

	t.Run("rejects negative ready timeout", func(t *testing.T) {
		// given: a config with a negative ReadyTimeout
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath:   "./cmd/foo",
				ReadyCheck:   testastic.ReadyCheckFunc(func(context.Context) bool { return true }),
				ReadyTimeout: -1 * time.Second,
			})
		})

		// then: the test fails with a negative timeout message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "ReadyTimeout must not be negative")
	})

	t.Run("rejects negative ready interval", func(t *testing.T) {
		// given: a config with a negative ReadyInterval
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath:    "./cmd/foo",
				ReadyCheck:    testastic.ReadyCheckFunc(func(context.Context) bool { return true }),
				ReadyInterval: -100 * time.Millisecond,
			})
		})

		// then: the test fails with a negative interval message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "ReadyInterval must not be negative")
	})

	t.Run("rejects negative shutdown timeout", func(t *testing.T) {
		// given: a config with a negative ShutdownTimeout
		mt := newProcessMockT()
		defer mt.cleanup()

		// when: starting the process
		runExpectingFatal(func() {
			testastic.StartProcess(context.Background(), mt, testastic.ProcessConfig{
				ImportPath:      "./cmd/foo",
				ReadyCheck:      testastic.ReadyCheckFunc(func(context.Context) bool { return true }),
				ShutdownTimeout: -1 * time.Second,
			})
		})

		// then: the test fails with a negative timeout message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "ShutdownTimeout must not be negative")
	})
}
