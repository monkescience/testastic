package testastic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Default configuration values for StartService.
const (
	defaultReadyTimeout    = 10 * time.Second
	defaultReadyInterval   = 100 * time.Millisecond
	defaultShutdownTimeout = 5 * time.Second
	defaultReadyEndpoint   = "/"
	httpCheckTimeout       = 1 * time.Second
)

// syncBuffer is a thread-safe wrapper around bytes.Buffer that implements
// io.Writer. It allows os/exec pipe goroutines to write concurrently while
// other goroutines read via String().
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

var _ io.Writer = (*syncBuffer)(nil)

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p) //nolint:wrapcheck // internal type, wrapping adds no value
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func (b *syncBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Len()
}

// ServiceConfig configures how a service binary is built, started, and monitored.
type ServiceConfig struct {
	// ImportPath is the Go import path of the service to build (e.g., "./cmd/myservice").
	// Mutually exclusive with BinaryPath. One of ImportPath or BinaryPath must be set.
	ImportPath string

	// BinaryPath is the path to a pre-built coverage-instrumented binary.
	// When set, the build step is skipped entirely. The caller is responsible
	// for ensuring the binary was built with `go build -cover`.
	// Mutually exclusive with ImportPath.
	BinaryPath string

	// Args are command-line arguments passed to the service binary.
	Args []string

	// Env is a list of additional environment variables in "KEY=VALUE" format.
	// These are appended to the current process environment.
	// GOCOVERDIR is set automatically and must not be included here.
	Env []string

	// Port is the TCP port the service listens on.
	// Used to construct the base URL (http://localhost:<Port>) and for the default
	// HTTP readiness check. Required.
	Port int

	// ReadyEndpoint is the HTTP path to poll for readiness (e.g., "/health", "/ready").
	// The service is considered ready when a GET request returns an HTTP 2xx response.
	// Mutually exclusive with ReadyFunc. If neither is set, defaults to "/".
	ReadyEndpoint string

	// ReadyFunc is a custom readiness check function. It is called repeatedly
	// until it returns true or the readiness timeout expires.
	// Mutually exclusive with ReadyEndpoint.
	ReadyFunc func() bool

	// ReadyTimeout is how long to wait for the service to become ready.
	// Default: 10 seconds.
	ReadyTimeout time.Duration

	// ReadyInterval is the polling interval between readiness checks.
	// Default: 100 milliseconds.
	ReadyInterval time.Duration

	// ShutdownTimeout is how long to wait for graceful shutdown after sending
	// an interrupt signal before sending a kill signal.
	// Default: 5 seconds.
	ShutdownTimeout time.Duration

	// CoverDir overrides the directory where coverage data files are written.
	// Default: a subdirectory created under t.TempDir().
	CoverDir string

	// BuildArgs are additional arguments passed to `go build` (e.g., "-ldflags", "-tags").
	// The flags -cover and -o are always added automatically.
	BuildArgs []string

	// WorkDir is the working directory for both `go build` and the service process.
	// Default: the directory containing the test file (detected via runtime.Caller).
	WorkDir string
}

// Service represents a running service process with coverage instrumentation.
// It is created by [StartService] and should not be constructed directly.
type Service struct {
	tb       testing.TB
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	baseURL  string
	coverDir string
	stdout   *syncBuffer
	stderr   *syncBuffer
	exited   chan struct{}
	mu       sync.Mutex
	stopped  bool
}

// StartService builds (if needed) and starts a Go service binary with coverage
// instrumentation, waits for it to become ready, and registers t.Cleanup for
// automatic shutdown and coverage collection.
//
// The provided context controls the service lifetime. When the context is
// cancelled, the service receives a graceful shutdown signal (SIGTERM on Unix,
// interrupt on Windows). If it does not exit within ShutdownTimeout, it is
// forcefully killed. Use [testing.T.Context] to tie the service lifetime to
// the test, or wrap it with a timeout for a maximum test duration:
//
//	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
//	defer cancel()
//	svc := testastic.StartService(ctx, t, testastic.ServiceConfig{...})
//
// When ImportPath is set, the service is built with `go build -cover`.
// When BinaryPath is set, the build step is skipped. The binary is started as
// a subprocess with GOCOVERDIR set so that coverage data is written on shutdown.
//
// The service must handle SIGTERM (Unix) or interrupt (Windows) for graceful
// shutdown and coverage data flushing. Services that only handle SIGINT will
// be forcefully killed after ShutdownTimeout, losing coverage data.
//
// The service is polled for readiness using either ReadyEndpoint (HTTP GET
// returning 2xx) or ReadyFunc. If the process exits before becoming ready,
// StartService calls t.Fatal immediately with captured output.
//
// Example:
//
//	func TestAPI(t *testing.T) {
//	    svc := testastic.StartService(t.Context(), t, testastic.ServiceConfig{
//	        ImportPath:    "./cmd/api",
//	        Port:          8080,
//	        ReadyEndpoint: "/health",
//	        Env:           []string{"DATABASE_URL=postgres://localhost/test"},
//	    })
//
//	    resp, err := http.Get(svc.URL() + "/api/users")
//	    testastic.NoError(t, err)
//	    defer resp.Body.Close()
//
//	    testastic.AssertJSON(t, "testdata/users.expected.json", resp.Body)
//	}
func StartService(ctx context.Context, tb testing.TB, cfg ServiceConfig) *Service {
	tb.Helper()
	validateServiceConfig(tb, cfg)

	if cfg.WorkDir == "" {
		if _, file, _, ok := runtime.Caller(1); ok {
			cfg.WorkDir = filepath.Dir(file)
		}
	}

	applyServiceDefaults(&cfg)
	binaryPath := cfg.BinaryPath

	if cfg.ImportPath != "" {
		binaryPath = buildService(ctx, tb, cfg)
	}

	coverDir := setupCoverDir(tb, cfg.CoverDir)
	ctx, cancel := context.WithCancel(ctx)
	svc := &Service{
		tb:       tb,
		cancel:   cancel,
		baseURL:  fmt.Sprintf("http://localhost:%d", cfg.Port),
		coverDir: coverDir,
		stdout:   &syncBuffer{},
		stderr:   &syncBuffer{},
		exited:   make(chan struct{}),
	}

	//nolint:gosec // args are from test config, not user input
	svc.cmd = exec.CommandContext(ctx, binaryPath, cfg.Args...)
	svc.cmd.Cancel = func() error {
		return interruptProcess(svc.cmd.Process)
	}
	svc.cmd.WaitDelay = cfg.ShutdownTimeout
	svc.cmd.Stdout = svc.stdout
	svc.cmd.Stderr = svc.stderr

	svc.cmd.Env = append(append(os.Environ(), cfg.Env...), "GOCOVERDIR="+coverDir)

	if cfg.WorkDir != "" {
		svc.cmd.Dir = cfg.WorkDir
	}

	err := svc.cmd.Start()
	if err != nil {
		tb.Fatalf("testastic: failed to start service: %v", err)
	}

	go func() {
		_ = svc.cmd.Wait()
		close(svc.exited)
	}()

	waitForReady(ctx, tb, svc, cfg)
	tb.Cleanup(svc.Stop)

	return svc
}

// URL returns the base URL of the running service (e.g., "http://localhost:8080").
func (s *Service) URL() string {
	return s.baseURL
}

// CoverDir returns the path to the directory containing raw coverage data files.
// Use `go tool covdata` to process the files:
//
//	go tool covdata textfmt -i=<coverdir> -o=coverage.out
func (s *Service) CoverDir() string {
	return s.coverDir
}

// Stop cancels the service context, triggering a graceful shutdown signal
// (SIGTERM on Unix, interrupt on Windows). If the service does not exit within
// ShutdownTimeout, it is forcefully killed.
// Coverage data is written to CoverDir on graceful shutdown.
//
// Stop is idempotent; calling it multiple times is safe.
// It is called automatically by the t.Cleanup handler registered by [StartService].
func (s *Service) Stop() {
	s.mu.Lock()

	if s.stopped {
		s.mu.Unlock()

		return
	}

	s.stopped = true
	s.mu.Unlock()

	s.cancel()
	<-s.exited

	s.logOutput()
}

func (s *Service) logOutput() {
	if !s.tb.Failed() {
		return
	}

	if s.stdout.Len() > 0 {
		s.tb.Logf("testastic: service stdout:\n%s", s.stdout.String())
	}

	if s.stderr.Len() > 0 {
		s.tb.Logf("testastic: service stderr:\n%s", s.stderr.String())
	}
}

func validateServiceConfig(tb testing.TB, cfg ServiceConfig) {
	tb.Helper()

	if cfg.ImportPath == "" && cfg.BinaryPath == "" {
		tb.Fatalf("testastic: ServiceConfig requires ImportPath or BinaryPath")
	}

	if cfg.ImportPath != "" && cfg.BinaryPath != "" {
		tb.Fatalf("testastic: ServiceConfig must not set both ImportPath and BinaryPath")
	}

	if cfg.ReadyEndpoint != "" && cfg.ReadyFunc != nil {
		tb.Fatalf("testastic: ServiceConfig must not set both ReadyEndpoint and ReadyFunc")
	}

	if cfg.Port == 0 {
		tb.Fatalf("testastic: ServiceConfig requires Port")
	}

	for _, e := range cfg.Env {
		if strings.HasPrefix(e, "GOCOVERDIR=") {
			tb.Fatalf("testastic: ServiceConfig.Env must not include GOCOVERDIR; use CoverDir instead")
		}
	}

	if cfg.ReadyTimeout < 0 {
		tb.Fatalf("testastic: ServiceConfig.ReadyTimeout must not be negative")
	}

	if cfg.ReadyInterval < 0 {
		tb.Fatalf("testastic: ServiceConfig.ReadyInterval must not be negative")
	}

	if cfg.ShutdownTimeout < 0 {
		tb.Fatalf("testastic: ServiceConfig.ShutdownTimeout must not be negative")
	}
}

func applyServiceDefaults(cfg *ServiceConfig) {
	if cfg.ReadyTimeout == 0 {
		cfg.ReadyTimeout = defaultReadyTimeout
	}

	if cfg.ReadyInterval == 0 {
		cfg.ReadyInterval = defaultReadyInterval
	}

	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeout
	}
}

func setupCoverDir(tb testing.TB, coverDir string) string {
	tb.Helper()

	if coverDir != "" {
		return coverDir
	}

	coverDir = filepath.Join(tb.TempDir(), "coverage")

	const dirPerm = 0o750

	err := os.MkdirAll(coverDir, dirPerm)
	if err != nil {
		tb.Fatalf("testastic: failed to create coverage directory: %v", err)
	}

	return coverDir
}

func buildService(ctx context.Context, tb testing.TB, cfg ServiceConfig) string {
	tb.Helper()

	outputName := "service"
	if runtime.GOOS == "windows" {
		outputName = "service.exe"
	}

	outputPath := filepath.Join(tb.TempDir(), outputName)

	args := make([]string, 0, 4+len(cfg.BuildArgs)+1)
	args = append(args, "build", "-cover", "-o", outputPath)
	args = append(args, cfg.BuildArgs...)
	args = append(args, cfg.ImportPath)

	cmd := exec.CommandContext(ctx, "go", args...) //nolint:gosec // args are from test config
	cmd.Dir = cfg.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("testastic: go build failed:\n%s", output)
	}

	return outputPath
}

func waitForReady(ctx context.Context, tb testing.TB, svc *Service, cfg ServiceConfig) {
	tb.Helper()

	var check func() bool

	if cfg.ReadyFunc != nil {
		check = cfg.ReadyFunc
	} else {
		endpoint := cfg.ReadyEndpoint
		if endpoint == "" {
			endpoint = defaultReadyEndpoint
		}

		check = httpReadyCheck(ctx, svc.baseURL+endpoint)
	}

	deadline := time.Now().Add(cfg.ReadyTimeout)
	ticker := time.NewTicker(cfg.ReadyInterval)

	defer ticker.Stop()

	for {
		select {
		case <-svc.exited:
			tb.Fatalf(
				"testastic: service exited before becoming ready\nstdout:\n%s\nstderr:\n%s",
				svc.stdout.String(), svc.stderr.String(),
			)

			return
		default:
		}

		if check() {
			return
		}

		if time.Now().After(deadline) {
			tb.Fatalf(
				"testastic: service not ready after %v\nstdout:\n%s\nstderr:\n%s",
				cfg.ReadyTimeout, svc.stdout.String(), svc.stderr.String(),
			)

			return
		}

		select {
		case <-ticker.C:
		case <-svc.exited:
			tb.Fatalf(
				"testastic: service exited before becoming ready\nstdout:\n%s\nstderr:\n%s",
				svc.stdout.String(), svc.stderr.String(),
			)

			return
		}
	}
}

func httpReadyCheck(ctx context.Context, url string) func() bool {
	client := &http.Client{Timeout: httpCheckTimeout}

	return func() bool {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false
		}

		resp, err := client.Do(req)
		if err != nil {
			return false
		}

		_ = resp.Body.Close()

		return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	}
}
