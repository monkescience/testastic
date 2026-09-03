package testastic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const defaultRunTimeout = 30 * time.Second

var errBuildBinaryRequiresImportPath = errors.New("BuildBinary requires importPath")

// Binary is a coverage-instrumented Go binary that can be reused across tests.
//
// Binaries built with [BuildBinaryMain] create a temporary directory that
// outlives individual tests. Call [Binary.Cleanup] after [testing.M.Run] to
// remove it. Binaries built with [BuildBinary] use [testing.TB.TempDir] and
// are cleaned up automatically.
type Binary struct {
	path    string
	workDir string
	tempDir string // non-empty only for BuildBinaryMain, removed by Cleanup
}

// RunResult contains the captured stdout, stderr, and exit code from a CLI run.
type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunOption configures optional behavior for a single [Binary.Run] invocation.
// Use the provided option constructors (such as [WithRunEnv] and
// [WithRunTimeout]) to create options.
type RunOption interface {
	applyRun(cfg *runConfig)
}

type runConfig struct {
	env     []string
	stdin   io.Reader
	timeout time.Duration
	workDir string
}

// BuildBinary builds a coverage-instrumented Go binary during a regular test
// and returns a reusable handle.
func BuildBinary(tb testing.TB, importPath string, opts ...BuildOption) *Binary {
	tb.Helper()

	cfg := newBuildConfig(opts)
	cfg.importPath = importPath

	if cfg.workDir == "" {
		cfg.workDir = defaultWorkDir()
	}

	validateBuildConfig(tb, cfg)

	outputPath := filepath.Join(tb.TempDir(), binaryOutputName("binary"))

	output, err := runGoBuild(testingContext(tb), outputPath, cfg)
	if err != nil {
		tb.Fatalf("testastic: go build failed:\n%s", output)
	}

	return &Binary{path: outputPath, workDir: cfg.workDir}
}

// BuildBinaryMain builds a coverage-instrumented Go binary in TestMain and
// returns a reusable handle. Build failures print to stderr and exit the test
// process, because TestMain does not have a testing.TB.
func BuildBinaryMain(m *testing.M, importPath string, opts ...BuildOption) *Binary {
	cfg := newBuildConfig(opts)
	cfg.importPath = importPath

	if cfg.workDir == "" {
		cfg.workDir = defaultWorkDir()
	}

	err := validateBuildConfigError(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testastic: %v\n", err)
		os.Exit(1)
	}

	buildDir, err := os.MkdirTemp("", "testastic-binary-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "testastic: create binary temp dir: %v\n", err)
		os.Exit(1)
	}

	outputPath := filepath.Join(buildDir, binaryOutputName("binary"))

	output, err := runGoBuild(context.Background(), outputPath, cfg)
	if err != nil {
		_ = os.RemoveAll(buildDir)

		fmt.Fprintf(os.Stderr, "testastic: go build failed:\n%s", output)
		os.Exit(1)
	}

	return &Binary{path: outputPath, workDir: cfg.workDir, tempDir: buildDir}
}

// NewBinary returns a reusable handle for a pre-built binary path.
func NewBinary(binaryPath string) *Binary {
	return &Binary{path: binaryPath, workDir: defaultWorkDir()}
}

// Path returns the filesystem path to the binary.
func (b *Binary) Path() string {
	return b.path
}

// Cleanup removes the temporary build directory created by [BuildBinaryMain].
// It is a no-op for binaries built with [BuildBinary] or wrapped with
// [NewBinary].
//
// Call Cleanup after [testing.M.Run] returns:
//
//	func TestMain(m *testing.M) {
//	    bin := testastic.BuildBinaryMain(m, "./cmd/api")
//	    code := m.Run()
//	    bin.Cleanup()
//	    os.Exit(code)
//	}
func (b *Binary) Cleanup() {
	if b.tempDir != "" {
		_ = os.RemoveAll(b.tempDir)
	}
}

// Start launches the binary as a long-running process and waits for readiness.
// It captures up to 1 MiB each of stdout and stderr for failure diagnostics.
// Output beyond that limit is replaced with a truncation marker.
func (b *Binary) Start(
	ctx context.Context, tb testing.TB,
	readyCheck ReadyChecker, opts ...ProcessOption,
) *Process {
	tb.Helper()

	if b == nil || b.path == "" {
		tb.Fatalf("testastic: binary path is empty")
	}

	cfg := newProcessConfig(opts)
	cfg.binaryPath = b.path
	cfg.readyCheck = readyCheck

	if cfg.workDir == "" {
		cfg.workDir = b.workDir
	}

	if cfg.workDir == "" {
		cfg.workDir = defaultWorkDir()
	}

	return startProcess(ctx, tb, cfg)
}

// Run executes the binary with the provided args and default run options.
func (b *Binary) Run(tb testing.TB, args ...string) *RunResult {
	tb.Helper()

	return b.RunWithOptions(tb, args)
}

// RunWithOptions executes the binary with the provided args and options,
// capturing up to 1 MiB each of stdout and stderr, plus the exit code.
// Output beyond that limit is replaced with a truncation marker.
func (b *Binary) RunWithOptions(tb testing.TB, args []string, opts ...RunOption) *RunResult {
	tb.Helper()

	if b == nil || b.path == "" {
		tb.Fatalf("testastic: binary path is empty")
	}

	cfg := newRunConfig(b, opts)
	validateRunConfig(tb, cfg)

	baseCtx := testingContext(tb)

	ctx, cancel := context.WithTimeout(baseCtx, cfg.timeout)
	defer cancel()

	stdout := &capturedOutput{}
	stderr := &capturedOutput{}
	coverDir := setupCoverDir(tb, "")

	cmd := exec.CommandContext(ctx, b.path, args...) //nolint:gosec // args are from test config
	cmd.WaitDelay = cfg.timeout
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = cfg.stdin
	cmd.Env = append(append(os.Environ(), cfg.env...), "GOCOVERDIR="+coverDir)
	cmd.Dir = cfg.workDir

	err := cmd.Run()
	result := &RunResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if err == nil {
		return result
	}

	return resolveRunOutcome(ctx, tb, err, result, cfg.timeout)
}

// resolveRunOutcome classifies a failed run. A timeout and an infrastructure
// cancellation are fatal, but a genuine non-zero exit (a real exit code, not a
// kill) returns the result even if the parent context was cancelled in the same
// window. A killed process has an exit code below zero and is treated as
// cancellation.
func resolveRunOutcome(
	ctx context.Context, tb testing.TB, err error, result *RunResult, timeout time.Duration,
) *RunResult {
	tb.Helper()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		tb.Fatalf(
			"testastic: binary run timed out after %v\nstdout:\n%s\nstderr:\n%s",
			timeout, result.Stdout, result.Stderr,
		)

		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && result.ExitCode >= 0 {
		return result
	}

	ctxErr := ctx.Err()
	if ctxErr != nil {
		tb.Fatalf("testastic: binary run failed: %v", ctxErr)

		return nil
	}

	tb.Fatalf("testastic: failed to run binary: %v", err)

	return nil
}

type runEnvOption struct{ env []string }

func (o runEnvOption) applyRun(c *runConfig) { c.env = o.env }

type stdinOption struct{ reader io.Reader }

func (o stdinOption) applyRun(c *runConfig) { c.stdin = o.reader }

type runTimeoutOption struct{ duration time.Duration }

func (o runTimeoutOption) applyRun(c *runConfig) { c.timeout = o.duration }

type runWorkDirOption struct{ dir string }

func (o runWorkDirOption) applyRun(c *runConfig) { c.workDir = o.dir }

// WithRunEnv sets additional environment variables in KEY=VALUE format for a
// single CLI run. When called multiple times, only the last value is used.
func WithRunEnv(env ...string) RunOption {
	return runEnvOption{env: env}
}

// WithStdin sets stdin for a single CLI run.
func WithStdin(r io.Reader) RunOption {
	return stdinOption{reader: r}
}

// WithRunTimeout sets the maximum duration for a single CLI run.
// Default: 30 seconds.
func WithRunTimeout(d time.Duration) RunOption {
	return runTimeoutOption{duration: d}
}

// WithRunWorkDir sets the working directory for a single CLI run.
func WithRunWorkDir(dir string) RunOption {
	return runWorkDirOption{dir: dir}
}

func newRunConfig(binary *Binary, opts []RunOption) *runConfig {
	cfg := &runConfig{timeout: defaultRunTimeout, workDir: binary.workDir}
	for _, opt := range opts {
		opt.applyRun(cfg)
	}

	if cfg.timeout == 0 {
		cfg.timeout = defaultRunTimeout
	}

	return cfg
}

func validateBuildConfig(tb testing.TB, cfg *buildConfig) {
	tb.Helper()

	err := validateBuildConfigError(cfg)
	if err != nil {
		tb.Fatalf("testastic: %v", err)
	}
}

func validateBuildConfigError(cfg *buildConfig) error {
	if cfg.importPath == "" {
		return errBuildBinaryRequiresImportPath
	}

	return nil
}

func validateRunConfig(tb testing.TB, cfg *runConfig) {
	tb.Helper()

	for _, e := range cfg.env {
		if strings.HasPrefix(e, "GOCOVERDIR=") {
			tb.Fatalf("testastic: WithRunEnv must not include GOCOVERDIR")
		}
	}

	if cfg.timeout < 0 {
		tb.Fatalf("testastic: WithRunTimeout must not be negative")
	}
}

// testingContext extracts the test context from tb. A type assertion is
// required because [testing.TB] does not include Context() in its interface,
// even though the concrete types [testing.T], [testing.B], and [testing.F]
// all implement it.
func testingContext(tb testing.TB) context.Context {
	tb.Helper()

	provider, ok := any(tb).(interface{ Context() context.Context })
	if !ok {
		return context.Background()
	}

	ctx := provider.Context()
	if ctx == nil {
		return context.Background()
	}

	return ctx
}

func binaryOutputName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}

	return base
}

func runGoBuild(ctx context.Context, outputPath string, cfg *buildConfig) ([]byte, error) {
	args := make([]string, 0, 4+len(cfg.buildArgs)+1)
	args = append(args, "build", "-cover", "-o", outputPath)
	args = append(args, cfg.buildArgs...)
	args = append(args, cfg.importPath)

	cmd := exec.CommandContext(ctx, "go", args...) //nolint:gosec // args are from test config
	cmd.Dir = cfg.workDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("go build: %w", err)
	}

	return output, nil
}
