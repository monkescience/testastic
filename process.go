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

// Default configuration values for [Binary.Start].
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
// Implementations are polled repeatedly by [Binary.Start] until they return
// true or the readiness timeout expires.
type ReadyChecker interface {
	Check(ctx context.Context) bool
}

// ReadyCheckFunc is an adapter to allow use of ordinary functions as [ReadyChecker].
// This follows the same pattern as [http.HandlerFunc] for [http.Handler].
type ReadyCheckFunc func(ctx context.Context) bool

var _ ReadyChecker = ReadyCheckFunc(nil)

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

// BuildOption configures optional behavior for [BuildBinary] and
// [BuildBinaryMain]. Use the provided option constructors (such as
// [WithBuildArgs] and [WithWorkDir]) to create options.
type BuildOption interface {
	applyBuild(cfg *buildConfig)
}

// ProcessOption configures optional behavior for [Binary.Start]. Use the
// provided option constructors (such as [WithPort] and [WithEnv]) to create
// options.
type ProcessOption interface {
	applyProcess(cfg *processConfig)
}

// buildConfig holds configuration for `go build`.
type buildConfig struct {
	importPath string
	buildArgs  []string
	workDir    string
}

// processConfig holds configuration for process startup.
type processConfig struct {
	binaryPath      string
	args            []string
	env             []string
	port            int
	readyCheck      ReadyChecker
	readyTimeout    time.Duration
	readyInterval   time.Duration
	shutdownTimeout time.Duration
	coverDir        string
	workDir         string
}

type argsOption struct{ args []string }

func (o argsOption) applyProcess(c *processConfig) {
	c.args = o.args
}

type envOption struct{ env []string }

func (o envOption) applyProcess(c *processConfig) {
	c.env = o.env
}

type portOption struct{ port int }

func (o portOption) applyProcess(c *processConfig) {
	c.port = o.port
}

type readyTimeoutOption struct{ duration time.Duration }

func (o readyTimeoutOption) applyProcess(c *processConfig) {
	c.readyTimeout = o.duration
}

type readyIntervalOption struct{ duration time.Duration }

func (o readyIntervalOption) applyProcess(c *processConfig) {
	c.readyInterval = o.duration
}

type shutdownTimeoutOption struct{ duration time.Duration }

func (o shutdownTimeoutOption) applyProcess(c *processConfig) {
	c.shutdownTimeout = o.duration
}

type coverDirOption struct{ dir string }

func (o coverDirOption) applyProcess(c *processConfig) {
	c.coverDir = o.dir
}

type buildArgsOption struct{ args []string }

func (o buildArgsOption) applyBuild(c *buildConfig) {
	c.buildArgs = o.args
}

type workDirOption struct{ dir string }

func (o workDirOption) applyBuild(c *buildConfig) {
	c.workDir = o.dir
}

func (o workDirOption) applyProcess(c *processConfig) {
	c.workDir = o.dir
}

// WithArgs sets command-line arguments passed to the binary.
func WithArgs(args ...string) ProcessOption {
	return argsOption{args: args}
}

// WithEnv sets additional environment variables in "KEY=VALUE" format.
// These are appended to the current process environment.
// GOCOVERDIR is set automatically and must not be included here.
func WithEnv(env ...string) ProcessOption {
	return envOption{env: env}
}

// WithPort sets the TCP port the process listens on.
// When set, [Process.URL] returns http://localhost:<port>.
// When not set, [Process.URL] returns an empty string.
func WithPort(port int) ProcessOption {
	return portOption{port: port}
}

// WithReadyTimeout sets how long to wait for the process to become ready.
// Default: 10 seconds.
func WithReadyTimeout(d time.Duration) ProcessOption {
	return readyTimeoutOption{duration: d}
}

// WithReadyInterval sets the polling interval between readiness checks.
// Default: 100 milliseconds.
func WithReadyInterval(d time.Duration) ProcessOption {
	return readyIntervalOption{duration: d}
}

// WithShutdownTimeout sets how long to wait for graceful shutdown after
// sending an interrupt signal before sending a kill signal.
// Default: 5 seconds.
func WithShutdownTimeout(d time.Duration) ProcessOption {
	return shutdownTimeoutOption{duration: d}
}

// WithCoverDir overrides the directory where coverage data files are written.
// Default: a subdirectory created under t.TempDir().
func WithCoverDir(dir string) ProcessOption {
	return coverDirOption{dir: dir}
}

// WithBuildArgs sets additional arguments passed to `go build`
// (e.g., "-ldflags", "-tags"). The flags -cover and -o are always added
// automatically.
func WithBuildArgs(args ...string) BuildOption {
	return buildArgsOption{args: args}
}

// WithWorkDir sets the working directory for `go build` and [Binary.Start].
// Default: the directory containing the test file (detected via
// runtime.Caller).
func WithWorkDir(dir string) interface {
	BuildOption
	ProcessOption
} {
	return workDirOption{dir: dir}
}

// Process represents a running process with coverage instrumentation.
// It is created by [Binary.Start] and should not be constructed directly.
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

func newProcessConfig(opts []ProcessOption) *processConfig {
	cfg := &processConfig{}
	for _, opt := range opts {
		opt.applyProcess(cfg)
	}

	return cfg
}

func newBuildConfig(opts []BuildOption) *buildConfig {
	cfg := &buildConfig{}
	for _, opt := range opts {
		opt.applyBuild(cfg)
	}

	return cfg
}

// defaultWorkDir returns the directory of the file that called the public
// function which called defaultWorkDir. The caller depth of 2 is correct
// because every call site follows the pattern: user code → public API
// (BuildBinary, NewBinary, Binary.Start, etc.) → defaultWorkDir.
// If a new internal wrapper is added between the public API and this function,
// the depth must be adjusted or the caller should resolve the directory at the
// public boundary and pass it explicitly.
func defaultWorkDir() string {
	const callerDepth = 2

	_, file, _, ok := runtime.Caller(callerDepth)
	if !ok {
		return ""
	}

	return filepath.Dir(file)
}

func startProcess(ctx context.Context, tb testing.TB, cfg *processConfig) *Process {
	tb.Helper()
	validateProcessConfig(tb, cfg)
	applyProcessDefaults(cfg)

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
	proc.cmd = exec.CommandContext(ctx, cfg.binaryPath, cfg.args...)
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
		cancel()
		tb.Fatalf("testastic: failed to start process: %v", err)

		return nil
	}

	go func() {
		_ = proc.cmd.Wait()
		close(proc.exited)
	}()

	tb.Cleanup(proc.Stop)
	waitForReady(ctx, tb, proc, cfg)

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

// Stop cancels the process context to shut the process down. On Unix it sends
// SIGTERM and the process has until ShutdownTimeout to exit gracefully, writing
// coverage data to CoverDir. On Windows, graceful interrupt is not currently
// delivered, so the process is terminated forcefully once ShutdownTimeout
// elapses and coverage data may not be flushed.
//
// Stop is idempotent, so calling it multiple times is safe.
// It is called automatically by the t.Cleanup handler registered by [Binary.Start].
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

	if cfg.binaryPath == "" {
		tb.Fatalf("testastic: Binary.Start requires a non-empty binary path")
	}

	if cfg.readyCheck == nil {
		tb.Fatalf("testastic: Binary.Start requires readyCheck")
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

	if shared := getSharedCoverDir(); shared != "" {
		// Per-run subdir avoids covmeta.<hash> race between parallel subprocesses.
		sub, err := os.MkdirTemp(shared, "run-*") //nolint:usetesting // must live under sharedCoverDir
		if err != nil {
			tb.Fatalf("testastic: failed to create per-run coverage directory: %v", err)
		}

		return sub
	}

	coverDir = filepath.Join(tb.TempDir(), "coverage")

	const dirPerm = 0o750

	err := os.MkdirAll(coverDir, dirPerm)
	if err != nil {
		tb.Fatalf("testastic: failed to create coverage directory: %v", err)
	}

	return coverDir
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
