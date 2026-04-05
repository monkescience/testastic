package testastic_test

import (
	"context"
	"fmt"
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

// serviceMockT extends mockT with methods needed by StartService.
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	testastic.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	testastic.NoError(t, err)

	return resp
}

func TestStartService_basic(t *testing.T) {
	port := nextPort()

	svc := testastic.StartService(context.Background(), t, testastic.ServiceConfig{
		ImportPath:    "./testdata/testservice",
		Port:          port,
		ReadyEndpoint: "/health",
		Env:           []string{fmt.Sprintf("PORT=%d", port)},
		ReadyTimeout:  10 * time.Second,
	})

	resp := doGet(t, svc.URL()+"/data")
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	testastic.Equal(t, http.StatusOK, resp.StatusCode)
	testastic.AssertJSON(t, "testdata/service_data.expected.json", resp.Body)
}

func TestStartService_coverage_collected(t *testing.T) {
	port := nextPort()

	svc := testastic.StartService(context.Background(), t, testastic.ServiceConfig{
		ImportPath:    "./testdata/testservice",
		Port:          port,
		ReadyEndpoint: "/health",
		Env:           []string{fmt.Sprintf("PORT=%d", port)},
		ReadyTimeout:  10 * time.Second,
	})

	// Make a request to generate some coverage.
	resp := doGet(t, svc.URL()+"/data")
	resp.Body.Close() //nolint:errcheck // test cleanup

	// Stop explicitly to flush coverage data.
	svc.Stop()

	entries, err := os.ReadDir(svc.CoverDir())
	testastic.NoError(t, err)
	testastic.True(t, len(entries) > 0)
}

func TestStartService_pre_built_binary(t *testing.T) {
	// Build the binary manually.
	outputName := "testservice"
	if runtime.GOOS == "windows" {
		outputName = "testservice.exe"
	}

	binaryPath := filepath.Join(t.TempDir(), outputName)
	cmd := exec.CommandContext(context.Background(), "go", "build", "-cover", "-o", binaryPath, "./testdata/testservice")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("manual build failed: %s", output)
	}

	port := nextPort()

	svc := testastic.StartService(context.Background(), t, testastic.ServiceConfig{
		BinaryPath:    binaryPath,
		Port:          port,
		ReadyEndpoint: "/health",
		Env:           []string{fmt.Sprintf("PORT=%d", port)},
		ReadyTimeout:  10 * time.Second,
	})

	resp := doGet(t, svc.URL()+"/health")
	resp.Body.Close() //nolint:errcheck // test cleanup

	testastic.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStartService_custom_ready_func(t *testing.T) {
	port := nextPort()

	svc := testastic.StartService(context.Background(), t, testastic.ServiceConfig{
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

	testastic.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestStartService_stop_idempotent(t *testing.T) {
	port := nextPort()

	svc := testastic.StartService(context.Background(), t, testastic.ServiceConfig{
		ImportPath:    "./testdata/testservice",
		Port:          port,
		ReadyEndpoint: "/health",
		Env:           []string{fmt.Sprintf("PORT=%d", port)},
		ReadyTimeout:  10 * time.Second,
	})

	svc.Stop()
	svc.Stop() // Should not panic.
}

func TestStartService_early_exit(t *testing.T) {
	mt := newServiceMockT()
	defer mt.cleanup()

	port := nextPort()

	done := make(chan struct{})

	go func() {
		defer close(done)

		testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
			ImportPath:    "./testdata/testservice",
			Port:          port,
			ReadyEndpoint: "/health",
			Env:           []string{fmt.Sprintf("PORT=%d", port), "EXIT_EARLY=true"},
			ReadyTimeout:  5 * time.Second,
		})
	}()

	<-done

	fatal, msg := mt.result()
	testastic.True(t, fatal)
	testastic.Contains(t, msg, "service exited before becoming ready")
}

func TestStartService_build_failure(t *testing.T) {
	mt := newServiceMockT()
	defer mt.cleanup()

	done := make(chan struct{})

	go func() {
		defer close(done)

		testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
			ImportPath:    "./nonexistent/package",
			Port:          nextPort(),
			ReadyEndpoint: "/health",
		})
	}()

	<-done

	fatal, msg := mt.result()
	testastic.True(t, fatal)
	testastic.Contains(t, msg, "go build failed")
}

func TestStartService_validation_no_path(t *testing.T) {
	mt := newServiceMockT()
	defer mt.cleanup()

	done := make(chan struct{})

	go func() {
		defer close(done)

		testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
			Port: nextPort(),
		})
	}()

	<-done

	fatal, msg := mt.result()
	testastic.True(t, fatal)
	testastic.Contains(t, msg, "requires ImportPath or BinaryPath")
}

func TestStartService_validation_both_paths(t *testing.T) {
	mt := newServiceMockT()
	defer mt.cleanup()

	done := make(chan struct{})

	go func() {
		defer close(done)

		testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
			ImportPath: "./cmd/foo",
			BinaryPath: "/bin/foo",
			Port:       nextPort(),
		})
	}()

	<-done

	fatal, msg := mt.result()
	testastic.True(t, fatal)
	testastic.Contains(t, msg, "must not set both ImportPath and BinaryPath")
}

func TestStartService_validation_both_ready(t *testing.T) {
	mt := newServiceMockT()
	defer mt.cleanup()

	done := make(chan struct{})

	go func() {
		defer close(done)

		testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
			ImportPath:    "./cmd/foo",
			Port:          nextPort(),
			ReadyEndpoint: "/health",
			ReadyFunc:     func() bool { return true },
		})
	}()

	<-done

	fatal, msg := mt.result()
	testastic.True(t, fatal)
	testastic.Contains(t, msg, "must not set both ReadyEndpoint and ReadyFunc")
}

func TestStartService_validation_no_port(t *testing.T) {
	mt := newServiceMockT()
	defer mt.cleanup()

	done := make(chan struct{})

	go func() {
		defer close(done)

		testastic.StartService(context.Background(), mt, testastic.ServiceConfig{
			ImportPath: "./cmd/foo",
		})
	}()

	<-done

	fatal, msg := mt.result()
	testastic.True(t, fatal)
	testastic.Contains(t, msg, "requires Port")
}
