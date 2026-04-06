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

// ProcessOption configures optional behavior of [StartProcess] and
// [StartProcessBinary]. Use the provided option constructors (WithPort,
// WithEnv, etc.) to create options.
type ProcessOption func(*processConfig)

// processConfig holds all configuration for process startup.
type processConfig struct {
	importPath      string
	binaryPath      string
	args            []string
	env             []string
	port            int
	readyCheck      ReadyChecker
	readyTimeout    time.Duration
	readyInterval   time.Duration
	shutdownTimeout time.Duration
	coverDir        string
	buildArgs       []string
	workDir         string
}

// WithArgs sets command-line arguments passed to the binary.
func WithArgs(args ...string) ProcessOption {
	return func(c *processConfig) {
		c.args = args
	}
}

// WithEnv sets additional environment variables in "KEY=VALUE" format.
// These are appended to the current process environment.
// GOCOVERDIR is set automatically and must not be included here.
func WithEnv(env ...string) ProcessOption {
	return func(c *processConfig) {
		c.env = env
	}
}

// WithPort sets the TCP port the process listens on.
// When set, [Process.URL] returns http://localhost:<port>.
// When not set, [Process.URL] returns an empty string.
func WithPort(port int) ProcessOption {
	return func(c *processConfig) {
		c.port = port
	}
}

// WithReadyTimeout sets how long to wait for the process to become ready.
// Default: 10 seconds.
func WithReadyTimeout(d time.Duration) ProcessOption {
	return func(c *processConfig) {
		c.readyTimeout = d
	}
}

// WithReadyInterval sets the polling interval between readiness checks.
// Default: 100 milliseconds.
func WithReadyInterval(d time.Duration) ProcessOption {
	return func(c *processConfig) {
		c.readyInterval = d
	}
}

// WithShutdownTimeout sets how long to wait for graceful shutdown after
// sending an interrupt signal before sending a kill signal.
// Default: 5 seconds.
func WithShutdownTimeout(d time.Duration) ProcessOption {
	return func(c *processConfig) {
		c.shutdownTimeout = d
	}
}

// WithCoverDir overrides the directory where coverage data files are written.
// Default: a subdirectory created under t.TempDir().
func WithCoverDir(dir string) ProcessOption {
	return func(c *processConfig) {
		c.coverDir = dir
	}
}

// WithBuildArgs sets additional arguments passed to `go build`
// (e.g., "-ldflags", "-tags"). The flags -cover and -o are always added
// automatically. Only meaningful with [StartProcess]; ignored by
// [StartProcessBinary].
func WithBuildArgs(args ...string) ProcessOption {
	return func(c *processConfig) {
		c.buildArgs = args
	}
}

// WithWorkDir sets the working directory for both `go build` and the process.
// Default: the directory containing the test file (detected via runtime.Caller).
func WithWorkDir(dir string) ProcessOption {
	return func(c *processConfig) {
		c.workDir = dir
	}
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

// StartProcess builds and starts a Go binary from importPath with coverage
// instrumentation, waits for it to become ready, and registers t.Cleanup for
// automatic shutdown and coverage collection.
//
// The binary is built with `go build -cover` and started as a subprocess with
// GOCOVERDIR set so that coverage data is written on shutdown.
//
// The provided context controls the process lifetime. When the context is
// cancelled, the process receives a graceful shutdown signal (SIGTERM on Unix,
// interrupt on Windows). If it does not exit within ShutdownTimeout, it is
// forcefully killed. Use [testing.T.Context] to tie the process lifetime to
// the test, or wrap it with a timeout for a maximum test duration:
//
//	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
//	defer cancel()
//	proc := testastic.StartProcess(ctx, t, "./cmd/api",
//	    testastic.HTTPCheck(8080, "/health"),
//	    testastic.WithPort(8080),
//	)
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
//	    proc := testastic.StartProcess(t.Context(), t, "./cmd/api",
//	        testastic.HTTPCheck(8080, "/health"),
//	        testastic.WithPort(8080),
//	        testastic.WithEnv("DATABASE_URL=postgres://localhost/test"),
//	    )
//
//	    resp, err := http.Get(proc.URL() + "/api/users")
//	    testastic.NoError(t, err)
//	    defer resp.Body.Close()
//
//	    testastic.AssertJSON(t, "testdata/users.expected.json", resp.Body)
//	}
func StartProcess(
	ctx context.Context, tb testing.TB,
	importPath string, readyCheck ReadyChecker, opts ...ProcessOption,
) *Process {
	tb.Helper()

	cfg := newProcessConfig(opts)
	cfg.importPath = importPath
	cfg.readyCheck = readyCheck

	if cfg.workDir == "" {
		if _, file, _, ok := runtime.Caller(1); ok {
			cfg.workDir = filepath.Dir(file)
		}
	}

	return startProcess(ctx, tb, cfg)
}

// StartProcessBinary starts a pre-built coverage-instrumented binary, waits
// for it to become ready, and registers t.Cleanup for automatic shutdown and
// coverage collection. The caller is responsible for ensuring the binary was
// built with `go build -cover`.
//
// See [StartProcess] for full lifecycle documentation.
//
// Example:
//
//	proc := testastic.StartProcessBinary(t.Context(), t, binaryPath,
//	    testastic.HTTPCheck(8080, "/health"),
//	    testastic.WithPort(8080),
//	)
func StartProcessBinary(
	ctx context.Context, tb testing.TB,
	binaryPath string, readyCheck ReadyChecker, opts ...ProcessOption,
) *Process {
	tb.Helper()

	cfg := newProcessConfig(opts)
	cfg.binaryPath = binaryPath
	cfg.readyCheck = readyCheck

	if cfg.workDir == "" {
		if _, file, _, ok := runtime.Caller(1); ok {
			cfg.workDir = filepath.Dir(file)
		}
	}

	return startProcess(ctx, tb, cfg)
}

func newProcessConfig(opts []ProcessOption) *processConfig {
	cfg := &processConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

func startProcess(ctx context.Context, tb testing.TB, cfg *processConfig) *Process {
	tb.Helper()
	validateProcessConfig(tb, cfg)
	applyProcessDefaults(cfg)

	binaryPath := cfg.binaryPath

	if cfg.importPath != "" {
		binaryPath = buildProcess(ctx, tb, cfg)
	}

	coverDir := setupCoverDir(tb, cfg.coverDir)
	ctx, cancel := context.WithCancel(ctx)
	proc := &Process{
		tb:       tb,
		cancel:   cancel,
		baseURL:  formatBaseURL(cfg.port),
		coverDir: coverDir,
		stdout:   &syncBuffer{},
		stderr:   &syncBuffer{},
		exited:   make(chan struct{}),
	}

	//nolint:gosec // args are from test config, not user input
	proc.cmd = exec.CommandContext(ctx, binaryPath, cfg.args...)
	proc.cmd.Cancel = func() error {
		return interruptProcess(proc.cmd.Process)
	}
	proc.cmd.WaitDelay = cfg.shutdownTimeout
	proc.cmd.Stdout = proc.stdout
	proc.cmd.Stderr = proc.stderr

	proc.cmd.Env = append(append(os.Environ(), cfg.env...), "GOCOVERDIR="+coverDir)

	if cfg.workDir != "" {
		proc.cmd.Dir = cfg.workDir
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
// Returns an empty string when [WithPort] is not set.
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

func validateProcessConfig(tb testing.TB, cfg *processConfig) {
	tb.Helper()

	if cfg.importPath == "" && cfg.binaryPath == "" {
		tb.Fatalf("testastic: StartProcess requires importPath or binaryPath")
	}

	if cfg.readyCheck == nil {
		tb.Fatalf("testastic: StartProcess requires readyCheck")
	}

	for _, e := range cfg.env {
		if strings.HasPrefix(e, "GOCOVERDIR=") {
			tb.Fatalf("testastic: WithEnv must not include GOCOVERDIR; use WithCoverDir instead")
		}
	}

	if cfg.readyTimeout < 0 {
		tb.Fatalf("testastic: WithReadyTimeout must not be negative")
	}

	if cfg.readyInterval < 0 {
		tb.Fatalf("testastic: WithReadyInterval must not be negative")
	}

	if cfg.shutdownTimeout < 0 {
		tb.Fatalf("testastic: WithShutdownTimeout must not be negative")
	}
}

func applyProcessDefaults(cfg *processConfig) {
	if cfg.readyTimeout == 0 {
		cfg.readyTimeout = defaultReadyTimeout
	}

	if cfg.readyInterval == 0 {
		cfg.readyInterval = defaultReadyInterval
	}

	if cfg.shutdownTimeout == 0 {
		cfg.shutdownTimeout = defaultShutdownTimeout
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

	if sharedCoverDir != "" {
		return sharedCoverDir
	}

	coverDir = filepath.Join(tb.TempDir(), "coverage")

	const dirPerm = 0o750

	err := os.MkdirAll(coverDir, dirPerm)
	if err != nil {
		tb.Fatalf("testastic: failed to create coverage directory: %v", err)
	}

	return coverDir
}

func buildProcess(ctx context.Context, tb testing.TB, cfg *processConfig) string {
	tb.Helper()

	outputName := "process"
	if runtime.GOOS == "windows" {
		outputName = "process.exe"
	}

	outputPath := filepath.Join(tb.TempDir(), outputName)

	args := make([]string, 0, 4+len(cfg.buildArgs)+1)
	args = append(args, "build", "-cover", "-o", outputPath)
	args = append(args, cfg.buildArgs...)
	args = append(args, cfg.importPath)

	cmd := exec.CommandContext(ctx, "go", args...) //nolint:gosec // args are from test config
	cmd.Dir = cfg.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("testastic: go build failed:\n%s", output)
	}

	return outputPath
}

func waitForReady(ctx context.Context, tb testing.TB, proc *Process, cfg *processConfig) {
	tb.Helper()

	deadline := time.Now().Add(cfg.readyTimeout)
	ticker := time.NewTicker(cfg.readyInterval)

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

		if cfg.readyCheck.Check(ctx) {
			return
		}

		if time.Now().After(deadline) {
			tb.Fatalf(
				"testastic: process not ready after %v\nstdout:\n%s\nstderr:\n%s",
				cfg.readyTimeout, proc.stdout.String(), proc.stderr.String(),
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
