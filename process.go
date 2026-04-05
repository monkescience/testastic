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

// Default configuration values for StartProcess.
const (
	defaultReadyTimeout    = 10 * time.Second
	defaultReadyInterval   = 100 * time.Millisecond
	defaultShutdownTimeout = 5 * time.Second
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

// ReadyChecker determines whether a process is ready to accept work.
// Implementations are polled repeatedly by [StartProcess] until they return
// true or the readiness timeout expires.
type ReadyChecker interface {
	Check(ctx context.Context) bool
}

// ReadyCheckFunc is an adapter to allow use of ordinary functions as [ReadyChecker].
// This follows the same pattern as [http.HandlerFunc] for [http.Handler].
type ReadyCheckFunc func(ctx context.Context) bool

// Check calls f(ctx).
func (f ReadyCheckFunc) Check(ctx context.Context) bool {
	return f(ctx)
}

// HTTPCheck returns a [ReadyChecker] that polls an HTTP endpoint until it
// returns a 2xx status code. The endpoint is constructed as
// http://localhost:<port><path> (e.g., HTTPCheck(8080, "/health")).
func HTTPCheck(port int, path string) ReadyChecker {
	url := fmt.Sprintf("http://localhost:%d%s", port, path)
	client := &http.Client{Timeout: httpCheckTimeout}

	return ReadyCheckFunc(func(ctx context.Context) bool {
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
	})
}

// ProcessConfig configures how a Go binary is built, started, and monitored.
type ProcessConfig struct {
	// ImportPath is the Go import path of the binary to build (e.g., "./cmd/myservice").
	// Mutually exclusive with BinaryPath. One of ImportPath or BinaryPath must be set.
	ImportPath string

	// BinaryPath is the path to a pre-built coverage-instrumented binary.
	// When set, the build step is skipped entirely. The caller is responsible
	// for ensuring the binary was built with `go build -cover`.
	// Mutually exclusive with ImportPath.
	BinaryPath string

	// Args are command-line arguments passed to the binary.
	Args []string

	// Env is a list of additional environment variables in "KEY=VALUE" format.
	// These are appended to the current process environment.
	// GOCOVERDIR is set automatically and must not be included here.
	Env []string

	// Port is the TCP port the process listens on. Optional.
	// When set, URL() returns http://localhost:<Port>.
	// When zero, URL() returns an empty string.
	Port int

	// ReadyCheck determines when the process is ready to accept work.
	// It is polled repeatedly until it returns true or ReadyTimeout expires.
	// Required. Use [HTTPCheck] for HTTP endpoints or [ReadyCheckFunc] for custom logic.
	ReadyCheck ReadyChecker

	// ReadyTimeout is how long to wait for the process to become ready.
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

	// WorkDir is the working directory for both `go build` and the process.
	// Default: the directory containing the test file (detected via runtime.Caller).
	WorkDir string
}

// Process represents a running process with coverage instrumentation.
// It is created by [StartProcess] and should not be constructed directly.
type Process struct {
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

// StartProcess builds (if needed) and starts a Go binary with coverage
// instrumentation, waits for it to become ready, and registers t.Cleanup for
// automatic shutdown and coverage collection.
//
// The provided context controls the process lifetime. When the context is
// cancelled, the process receives a graceful shutdown signal (SIGTERM on Unix,
// interrupt on Windows). If it does not exit within ShutdownTimeout, it is
// forcefully killed. Use [testing.T.Context] to tie the process lifetime to
// the test, or wrap it with a timeout for a maximum test duration:
//
//	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
//	defer cancel()
//	proc := testastic.StartProcess(ctx, t, testastic.ProcessConfig{...})
//
// When ImportPath is set, the binary is built with `go build -cover`.
// When BinaryPath is set, the build step is skipped. The binary is started as
// a subprocess with GOCOVERDIR set so that coverage data is written on shutdown.
//
// The process must handle SIGTERM (Unix) or interrupt (Windows) for graceful
// shutdown and coverage data flushing. Processes that only handle SIGINT will
// be forcefully killed after ShutdownTimeout, losing coverage data.
//
// The process is polled for readiness using the provided [ReadyChecker].
// If the process exits before becoming ready, StartProcess calls t.Fatal
// immediately with captured output.
//
// Example:
//
//	func TestAPI(t *testing.T) {
//	    proc := testastic.StartProcess(t.Context(), t, testastic.ProcessConfig{
//	        ImportPath: "./cmd/api",
//	        Port:       8080,
//	        ReadyCheck: testastic.HTTPCheck(8080, "/health"),
//	        Env:        []string{"DATABASE_URL=postgres://localhost/test"},
//	    })
//
//	    resp, err := http.Get(proc.URL() + "/api/users")
//	    testastic.NoError(t, err)
//	    defer resp.Body.Close()
//
//	    testastic.AssertJSON(t, "testdata/users.expected.json", resp.Body)
//	}
func StartProcess(ctx context.Context, tb testing.TB, cfg ProcessConfig) *Process {
	tb.Helper()
	validateProcessConfig(tb, cfg)

	if cfg.WorkDir == "" {
		if _, file, _, ok := runtime.Caller(1); ok {
			cfg.WorkDir = filepath.Dir(file)
		}
	}

	applyProcessDefaults(&cfg)
	binaryPath := cfg.BinaryPath

	if cfg.ImportPath != "" {
		binaryPath = buildProcess(ctx, tb, cfg)
	}

	coverDir := setupCoverDir(tb, cfg.CoverDir)
	ctx, cancel := context.WithCancel(ctx)
	proc := &Process{
		tb:       tb,
		cancel:   cancel,
		baseURL:  formatBaseURL(cfg.Port),
		coverDir: coverDir,
		stdout:   &syncBuffer{},
		stderr:   &syncBuffer{},
		exited:   make(chan struct{}),
	}

	//nolint:gosec // args are from test config, not user input
	proc.cmd = exec.CommandContext(ctx, binaryPath, cfg.Args...)
	proc.cmd.Cancel = func() error {
		return interruptProcess(proc.cmd.Process)
	}
	proc.cmd.WaitDelay = cfg.ShutdownTimeout
	proc.cmd.Stdout = proc.stdout
	proc.cmd.Stderr = proc.stderr

	proc.cmd.Env = append(append(os.Environ(), cfg.Env...), "GOCOVERDIR="+coverDir)

	if cfg.WorkDir != "" {
		proc.cmd.Dir = cfg.WorkDir
	}

	err := proc.cmd.Start()
	if err != nil {
		tb.Fatalf("testastic: failed to start process: %v", err)
	}

	go func() {
		_ = proc.cmd.Wait()
		close(proc.exited)
	}()

	waitForReady(ctx, tb, proc, cfg)
	tb.Cleanup(proc.Stop)

	return proc
}

// URL returns the base URL of the running process (e.g., "http://localhost:8080").
// Returns an empty string when Port is not set in ProcessConfig.
func (p *Process) URL() string {
	return p.baseURL
}

// CoverDir returns the path to the directory containing raw coverage data files.
// Use `go tool covdata` to process the files:
//
//	go tool covdata textfmt -i=<coverdir> -o=coverage.out
func (p *Process) CoverDir() string {
	return p.coverDir
}

// Stop cancels the process context, triggering a graceful shutdown signal
// (SIGTERM on Unix, interrupt on Windows). If the process does not exit within
// ShutdownTimeout, it is forcefully killed.
// Coverage data is written to CoverDir on graceful shutdown.
//
// Stop is idempotent; calling it multiple times is safe.
// It is called automatically by the t.Cleanup handler registered by [StartProcess].
func (p *Process) Stop() {
	p.mu.Lock()

	if p.stopped {
		p.mu.Unlock()

		return
	}

	p.stopped = true
	p.mu.Unlock()

	p.cancel()
	<-p.exited

	p.logOutput()
}

func (p *Process) logOutput() {
	if !p.tb.Failed() {
		return
	}

	if p.stdout.Len() > 0 {
		p.tb.Logf("testastic: process stdout:\n%s", p.stdout.String())
	}

	if p.stderr.Len() > 0 {
		p.tb.Logf("testastic: process stderr:\n%s", p.stderr.String())
	}
}

func validateProcessConfig(tb testing.TB, cfg ProcessConfig) {
	tb.Helper()

	if cfg.ImportPath == "" && cfg.BinaryPath == "" {
		tb.Fatalf("testastic: ProcessConfig requires ImportPath or BinaryPath")
	}

	if cfg.ImportPath != "" && cfg.BinaryPath != "" {
		tb.Fatalf("testastic: ProcessConfig must not set both ImportPath and BinaryPath")
	}

	if cfg.ReadyCheck == nil {
		tb.Fatalf("testastic: ProcessConfig requires ReadyCheck")
	}

	for _, e := range cfg.Env {
		if strings.HasPrefix(e, "GOCOVERDIR=") {
			tb.Fatalf("testastic: ProcessConfig.Env must not include GOCOVERDIR; use CoverDir instead")
		}
	}

	if cfg.ReadyTimeout < 0 {
		tb.Fatalf("testastic: ProcessConfig.ReadyTimeout must not be negative")
	}

	if cfg.ReadyInterval < 0 {
		tb.Fatalf("testastic: ProcessConfig.ReadyInterval must not be negative")
	}

	if cfg.ShutdownTimeout < 0 {
		tb.Fatalf("testastic: ProcessConfig.ShutdownTimeout must not be negative")
	}
}

func applyProcessDefaults(cfg *ProcessConfig) {
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

func formatBaseURL(port int) string {
	if port > 0 {
		return fmt.Sprintf("http://localhost:%d", port)
	}

	return ""
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

func buildProcess(ctx context.Context, tb testing.TB, cfg ProcessConfig) string {
	tb.Helper()

	outputName := "process"
	if runtime.GOOS == "windows" {
		outputName = "process.exe"
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

func waitForReady(ctx context.Context, tb testing.TB, proc *Process, cfg ProcessConfig) {
	tb.Helper()

	deadline := time.Now().Add(cfg.ReadyTimeout)
	ticker := time.NewTicker(cfg.ReadyInterval)

	defer ticker.Stop()

	for {
		select {
		case <-proc.exited:
			tb.Fatalf(
				"testastic: process exited before becoming ready\nstdout:\n%s\nstderr:\n%s",
				proc.stdout.String(), proc.stderr.String(),
			)

			return
		default:
		}

		if cfg.ReadyCheck.Check(ctx) {
			return
		}

		if time.Now().After(deadline) {
			tb.Fatalf(
				"testastic: process not ready after %v\nstdout:\n%s\nstderr:\n%s",
				cfg.ReadyTimeout, proc.stdout.String(), proc.stderr.String(),
			)

			return
		}

		select {
		case <-ticker.C:
		case <-proc.exited:
			tb.Fatalf(
				"testastic: process exited before becoming ready\nstdout:\n%s\nstderr:\n%s",
				proc.stdout.String(), proc.stderr.String(),
			)

			return
		}
	}
}
