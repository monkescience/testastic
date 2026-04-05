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

// serviceMockT implements testing.TB for StartService tests.
// Unlike mockT, it calls runtime.Goexit in Fatalf because StartService
// performs real work (building, starting processes) after validation calls
// to tb.Fatalf. Without Goexit, execution would continue past the fatal
// into build/exec, overwriting the error message.
type serviceMockT struct {
	testing.TB
	mu       sync.Mutex
	failed   bool
	fatal    bool
	message  string
	cleanups []func()
	tempDirs []string
}

func newServiceMockT() *serviceMockT {
	return &serviceMockT{}
}

func (m *serviceMockT) Helper() {}

func (m *serviceMockT) Fatalf(format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failed = true
	m.fatal = true
	m.message = fmt.Sprintf(format, args...)

	runtime.Goexit()
}

func (m *serviceMockT) Errorf(format string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failed = true
	m.message = fmt.Sprintf(format, args...)
}

func (m *serviceMockT) Logf(string, ...any) {}

func (m *serviceMockT) Log(...any) {}

func (m *serviceMockT) Name() string { return "serviceMockT" }

func (m *serviceMockT) Failed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.failed
}

func (m *serviceMockT) Cleanup(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cleanups = append(m.cleanups, f)
}

func (m *serviceMockT) TempDir() string {
	dir, err := os.MkdirTemp("", "testastic-service-test-*")
	if err != nil {
		panic(fmt.Sprintf("TempDir: %v", err))
	}

	m.mu.Lock()
	m.tempDirs = append(m.tempDirs, dir)
	m.mu.Unlock()

	return dir
}

func (m *serviceMockT) cleanup() {
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

func (m *serviceMockT) result() (bool, string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.fatal, m.message
}

// runExpectingFatal runs fn in a separate goroutine so that runtime.Goexit
// from serviceMockT.Fatalf terminates only that goroutine.
func runExpectingFatal(fn func()) {
	done := make(chan struct{})

	go func() {
		defer close(done)

		fn()
	}()

	<-done
}

// testServicePort tracks the next available port for test services.
var testServicePort = struct {
	mu   sync.Mutex
	next int
}{next: 18900}

func nextPort() int {
	testServicePort.mu.Lock()
	defer testServicePort.mu.Unlock()

	p := testServicePort.next
	testServicePort.next++

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

func TestStartService(t *testing.T) {
	t.Run("serves requests after startup", func(t *testing.T) {
		// given: a valid service config with an import path
		port := nextPort()

		// when: starting the service and making a request
		svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
			ImportPath:    "./testdata/testservice",
			Port:          port,
			ReadyEndpoint: "/health",
			Env:           []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout:  10 * time.Second,
		})

		resp := doGet(t, svc.URL()+"/data")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the response matches the expected JSON
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.AssertJSON(t, "testdata/service_data.expected.json", resp.Body)
	})

	t.Run("collects coverage data", func(t *testing.T) {
		// given: a running service with coverage instrumentation
		port := nextPort()
		svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
			ImportPath:    "./testdata/testservice",
			Port:          port,
			ReadyEndpoint: "/health",
			Env:           []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout:  10 * time.Second,
		})

		// when: making a request and stopping the service to flush coverage
		resp := doGet(t, svc.URL()+"/data")
		resp.Body.Close() //nolint:errcheck // test cleanup
		svc.Stop()

		// then: coverage data is written and contains actual counter data
		cmd := exec.CommandContext(t.Context(), "go", "tool", "covdata", "percent", "-i="+svc.CoverDir())
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
		svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
			BinaryPath:    binaryPath,
			Port:          port,
			ReadyEndpoint: "/health",
			Env:           []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout:  10 * time.Second,
		})

		resp := doGet(t, svc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the service responds successfully
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("custom ready func", func(t *testing.T) {
		// given: a config with a custom readiness check function
		port := nextPort()

		// when: starting with ReadyFunc instead of ReadyEndpoint
		svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
			ImportPath: "./testdata/testservice",
			Port:       port,
			Env:        []string{fmt.Sprintf("PORT=%d", port)},
			ReadyFunc: func() bool {
				client := &http.Client{Timeout: 500 * time.Millisecond}

				req, err := http.NewRequestWithContext(
					context.Background(), http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil,
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
			},
			ReadyTimeout: 10 * time.Second,
		})

		resp := doGet(t, svc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup

		// then: the service is reachable
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		// given: a running service
		port := nextPort()
		svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
			ImportPath:    "./testdata/testservice",
			Port:          port,
			ReadyEndpoint: "/health",
			Env:           []string{fmt.Sprintf("PORT=%d", port)},
			ReadyTimeout:  10 * time.Second,
		})

		// when: calling Stop twice
		svc.Stop()
		svc.Stop()

		// then: no panic occurs (test passes)
	})

	t.Run("detects early exit", func(t *testing.T) {
		// given: a service that exits immediately on startup
		mt := newServiceMockT()
		defer mt.cleanup()

		port := nextPort()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath:    "./testdata/testservice",
				Port:          port,
				ReadyEndpoint: "/health",
				Env:           []string{fmt.Sprintf("PORT=%d", port), "EXIT_EARLY=true"},
				ReadyTimeout:  5 * time.Second,
			})
		})

		// then: the test fails with an early exit message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "service exited before becoming ready")
	})

	t.Run("reports build failure", func(t *testing.T) {
		// given: a config pointing to a nonexistent package
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath:    "./nonexistent/package",
				Port:          nextPort(),
				ReadyEndpoint: "/health",
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

		svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
			ImportPath:    "./testdata/testservice",
			Port:          port,
			ReadyEndpoint: "/health",
			Env:           []string{fmt.Sprintf("PORT=%d", port)},
			CoverDir:      coverDir,
			ReadyTimeout:  10 * time.Second,
		})

		// when: making a request and stopping to flush coverage
		resp := doGet(t, svc.URL()+"/health")
		resp.Body.Close() //nolint:errcheck // test cleanup
		svc.Stop()

		// then: the returned CoverDir matches and contains coverage data
		testastic.Equal(t, coverDir, svc.CoverDir())

		entries, err := os.ReadDir(coverDir)
		testastic.NoError(t, err)
		testastic.True(t, len(entries) > 0)
	})

	t.Run("custom env passed to service", func(t *testing.T) {
		// given: a config with a custom environment variable
		port := nextPort()
		svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
			ImportPath:    "./testdata/testservice",
			Port:          port,
			ReadyEndpoint: "/health",
			Env:           []string{fmt.Sprintf("PORT=%d", port), "MY_TEST_VAR=hello-from-test"},
			ReadyTimeout:  10 * time.Second,
		})

		// when: querying the service for the env var value
		resp := doGet(t, svc.URL()+"/env?key=MY_TEST_VAR")
		defer resp.Body.Close() //nolint:errcheck // test cleanup

		body, err := io.ReadAll(resp.Body)
		testastic.NoError(t, err)

		// then: the service sees the custom env var
		testastic.Equal(t, http.StatusOK, resp.StatusCode)
		testastic.Equal(t, "hello-from-test", string(body))
	})

	t.Run("pre-cancelled context", func(t *testing.T) {
		// given: a context that is already cancelled
		mt := newServiceMockT()
		defer mt.cleanup()

		port := nextPort()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// when: starting the service with the cancelled context
		runExpectingFatal(func() {
			testastic.StartService(ctx, mt, testastic.ServiceConfig{
				ImportPath:    "./testdata/testservice",
				Port:          port,
				ReadyEndpoint: "/health",
				Env:           []string{fmt.Sprintf("PORT=%d", port)},
				ReadyTimeout:  2 * time.Second,
			})
		})

		// then: the test fails (service never becomes ready)
		fatal, _ := mt.result()
		testastic.True(t, fatal)
	})

	t.Run("context timeout during readiness", func(t *testing.T) {
		// given: a context that expires before the service can become ready
		mt := newServiceMockT()
		defer mt.cleanup()

		port := nextPort()

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		// when: starting a slow service with a short context timeout
		runExpectingFatal(func() {
			testastic.StartService(ctx, mt, testastic.ServiceConfig{
				ImportPath:    "./testdata/testservice",
				Port:          port,
				ReadyEndpoint: "/health",
				Env:           []string{fmt.Sprintf("PORT=%d", port), "SLOW_START=10s"},
				ReadyTimeout:  10 * time.Second,
			})
		})

		// then: the test fails because the context expired during polling
		fatal, _ := mt.result()
		testastic.True(t, fatal)
	})
}

func TestStartService_validation(t *testing.T) {
	t.Run("requires import or binary path", func(t *testing.T) {
		// given: a config with neither ImportPath nor BinaryPath
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				Port: nextPort(),
			})
		})

		// then: the test fails with a missing path message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "requires ImportPath or BinaryPath")
	})

	t.Run("rejects both import and binary path", func(t *testing.T) {
		// given: a config with both ImportPath and BinaryPath set
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath: "./cmd/foo",
				BinaryPath: "/bin/foo",
				Port:       nextPort(),
			})
		})

		// then: the test fails with a mutual exclusivity message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "must not set both ImportPath and BinaryPath")
	})

	t.Run("rejects both ready endpoint and func", func(t *testing.T) {
		// given: a config with both ReadyEndpoint and ReadyFunc set
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath:    "./cmd/foo",
				Port:          nextPort(),
				ReadyEndpoint: "/health",
				ReadyFunc:     func() bool { return true },
			})
		})

		// then: the test fails with a mutual exclusivity message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "must not set both ReadyEndpoint and ReadyFunc")
	})

	t.Run("requires port", func(t *testing.T) {
		// given: a config without a port
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath: "./cmd/foo",
			})
		})

		// then: the test fails with a missing port message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "requires Port")
	})

	t.Run("rejects GOCOVERDIR in env", func(t *testing.T) {
		// given: a config that includes GOCOVERDIR in the Env slice
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath: "./cmd/foo",
				Port:       nextPort(),
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
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath:   "./cmd/foo",
				Port:         nextPort(),
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
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath:    "./cmd/foo",
				Port:          nextPort(),
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
		mt := newServiceMockT()
		defer mt.cleanup()

		// when: starting the service
		runExpectingFatal(func() {
			testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
				ImportPath:      "./cmd/foo",
				Port:            nextPort(),
				ShutdownTimeout: -1 * time.Second,
			})
		})

		// then: the test fails with a negative timeout message
		fatal, msg := mt.result()
		testastic.True(t, fatal)
		testastic.Contains(t, msg, "ShutdownTimeout must not be negative")
	})
}
