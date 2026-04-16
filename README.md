# testastic

A Go testing toolkit with structured document comparison (JSON, YAML, HTML) and expressive assertions.

## Install

```bash
go get github.com/monkescience/testastic
```

## Document Assertions

Compare API responses or rendered documents against expected files with template matchers.

### JSON

```go
testastic.AssertJSON(t, "testdata/user.expected.json", resp.Body)
```

### YAML

```go
testastic.AssertYAML(t, "testdata/config.expected.yaml", configBytes)
```

### HTML

```go
testastic.AssertHTML(t, "testdata/page.expected.html", renderedHTML)
```

### Plain Text Files

```go
testastic.AssertFile(t, "testdata/output.expected.txt", actualString)
```

Input types by assertion:

- `AssertJSON` and `AssertYAML` accept `string`, `[]byte`, `io.Reader`, or any struct (auto-marshaled).
- `AssertHTML` accepts `string`, `[]byte`, `io.Reader`, or `fmt.Stringer`.
- `AssertFile` accepts `string`, `[]byte`, or `io.Reader`.

## Matchers

Expected files support template matchers for dynamic values:

```json
{
  "id": "{{anyUUID}}",
  "count": "{{anyInt}}",
  "email": "{{regex `^[a-z]+@example\\.com$`}}",
  "status": "{{oneOf \"pending\" \"active\"}}",
  "timestamp": "{{ignore}}"
}
```

**Built-in matchers:**

| Matcher | Description |
|---|---|
| `{{anyString}}` | Matches any string |
| `{{anyInt}}` | Matches any integer |
| `{{anyFloat}}` | Matches any number |
| `{{anyBool}}` | Matches any boolean |
| `{{anyValue}}` | Matches any value including null |
| `{{anyUUID}}` | Matches UUID strings (RFC 4122) |
| `{{anyDateTime}}` | Matches ISO 8601 datetime strings |
| `{{anyURL}}` | Matches URL strings |
| `{{ignore}}` | Skips the field during comparison |
| `{{regex \`pattern\`}}` | Matches against a regular expression |
| `{{oneOf "a" "b"}}` | Matches one of the specified values |

### Custom Matchers

Register custom matchers for domain-specific validation:

```go
testastic.RegisterMatcher("orderID", func(args string) (testastic.Matcher, error) {
    return &orderIDMatcher{}, nil
})
```

Then use in expected files: `"id": "{{orderID}}"`

## Options

### General Options

```go
// Ignore specific fields by name or path
AssertJSON(t, expected, actual, IgnoreFields("id", "timestamp"))
AssertJSON(t, expected, actual, IgnoreFields("$.user.id"))

// Ignore array order globally or at specific paths
AssertJSON(t, expected, actual, IgnoreArrayOrder())
AssertJSON(t, expected, actual, IgnoreArrayOrderAt("$.items"))

// Add context to failure messages
AssertJSON(t, expected, actual, Message("user creation response"))

// Force update expected file programmatically
AssertJSON(t, expected, actual, Update())
```

### HTML-Specific Options

```go
AssertHTML(t, expected, actual, IgnoreHTMLComments())
AssertHTML(t, expected, actual, PreserveWhitespace())
AssertHTML(t, expected, actual, IgnoreChildOrder())
AssertHTML(t, expected, actual, IgnoreChildOrderAt("html > body > ul"))
AssertHTML(t, expected, actual, IgnoreElements("script", "style"))
AssertHTML(t, expected, actual, IgnoreAttributes("class", "style"))
AssertHTML(t, expected, actual, IgnoreAttributeAt("html > body > div@id"))
```

## Updating Expected Files

When responses change, update expected files automatically:

```bash
go test -update
# or
TESTASTIC_UPDATE=true go test
```

## General Assertions

### Equality

```go
testastic.Equal(t, expected, actual)
testastic.NotEqual(t, unexpected, actual)
testastic.DeepEqual(t, expected, actual)
```

### Nil / Boolean

```go
testastic.Nil(t, value)
testastic.NotNil(t, value)
testastic.True(t, value)
testastic.False(t, value)
```

### Errors

```go
testastic.NoError(t, err)
testastic.Error(t, err)
testastic.ErrorIs(t, err, target)
testastic.ErrorAs(t, err, &pathErr)
testastic.ErrorContains(t, err, "substring")
```

### Panics

```go
testastic.Panics(t, func() { panic("boom") })
testastic.NotPanics(t, func() { safeFunc() })
```

### Comparison

```go
testastic.Greater(t, a, b)
testastic.GreaterOrEqual(t, a, b)
testastic.Less(t, a, b)
testastic.LessOrEqual(t, a, b)
testastic.Between(t, value, min, max)
```

### Strings

```go
testastic.Contains(t, s, substring)
testastic.NotContains(t, s, substring)
testastic.HasPrefix(t, s, prefix)
testastic.NotHasPrefix(t, s, prefix)
testastic.HasSuffix(t, s, suffix)
testastic.NotHasSuffix(t, s, suffix)
testastic.Matches(t, s, `^\d+$`)
testastic.StringEmpty(t, s)
testastic.StringNotEmpty(t, s)
```

`Contains`, `NotContains`, `HasPrefix`, `NotHasPrefix`, `HasSuffix`, `NotHasSuffix`, and `Matches` accept `string`, `[]byte`, or `fmt.Stringer` inputs.

### Collections

```go
testastic.Len(t, collection, expected)
testastic.Empty(t, collection)
testastic.NotEmpty(t, collection)
testastic.SliceContains(t, slice, element)
testastic.SliceNotContains(t, slice, element)
testastic.SliceEqual(t, expected, actual)
testastic.MapHasKey(t, m, key)
testastic.MapNotHasKey(t, m, key)
testastic.MapHasValue(t, m, value)
testastic.MapNotHasValue(t, m, value)
testastic.MapEqual(t, expected, actual)
```

## Eventual Assertions

For asynchronous operations, retry until a condition is met or timeout is reached. The condition is checked immediately, then at regular intervals (default 100ms).

```go
testastic.Eventually(t, func() bool {
    return server.IsReady()
}, 5*time.Second)

testastic.EventuallyEqual(t, "ready", func() string {
    return service.Status()
}, 3*time.Second)

testastic.EventuallyNoError(t, func() error {
    _, err := client.Ping()
    return err
}, 5*time.Second)
```

**All Eventually variants:**

```go
testastic.Eventually(t, conditionFn, timeout)
testastic.EventuallyTrue(t, conditionFn, timeout)
testastic.EventuallyFalse(t, conditionFn, timeout)
testastic.EventuallyEqual(t, expected, getValueFn, timeout)
testastic.EventuallyNil(t, getValueFn, timeout)
testastic.EventuallyNotNil(t, getValueFn, timeout)
testastic.EventuallyNoError(t, getErrFn, timeout)
testastic.EventuallyError(t, getErrFn, timeout)
```

**Options:**

```go
testastic.Eventually(t, condition, 5*time.Second,
    testastic.WithInterval(50*time.Millisecond),
    testastic.WithMessage("waiting for server"),
)
```

## Process Testing

Start and test Go binaries as subprocesses with automatic coverage instrumentation.

### Starting a Process

Build the binary once, then reuse it across tests:

```go
var apiBinary *testastic.Binary

func TestMain(m *testing.M) {
    apiBinary = testastic.BuildBinaryMain(m, "./cmd/api",
        testastic.WithBuildArgs("-tags", "integration"),
    )

    code := testastic.CollectSubprocessCoverage(m, "coverage/process.out")
    apiBinary.Cleanup()
    os.Exit(code)
}

func TestAPI(t *testing.T) {
    proc := apiBinary.Start(t.Context(), t,
        testastic.HTTPCheck(8080, "/health"),
        testastic.WithPort(8080),
        testastic.WithEnv("DATABASE_URL=postgres://localhost/test"),
    )

    resp, err := http.Get(proc.URL() + "/api/users")
    testastic.NoError(t, err)
    defer resp.Body.Close()

    testastic.AssertJSON(t, "testdata/users.expected.json", resp.Body)
}
```

For pre-built binaries, open them and start them explicitly:

```go
proc := testastic.NewBinary(binaryPath).Start(t.Context(), t,
    testastic.HTTPCheck(8080, "/health"),
    testastic.WithPort(8080),
)
```

### Process Options

| Option | Description |
|---|---|
| `WithPort(port)` | TCP port; enables `proc.URL()` |
| `WithEnv(vars...)` | Environment variables (`"KEY=VALUE"`) |
| `WithArgs(args...)` | Command-line arguments |
| `WithReadyTimeout(d)` | Readiness timeout (default: 10s) |
| `WithReadyInterval(d)` | Readiness poll interval (default: 100ms) |
| `WithShutdownTimeout(d)` | Graceful shutdown timeout (default: 5s) |
| `WithCoverDir(dir)` | Override coverage data directory |
| `WithWorkDir(dir)` | Working directory for build and process start |

Build options:

| Option | Description |
|---|---|
| `WithBuildArgs(args...)` | Additional `go build` flags |
| `WithWorkDir(dir)` | Working directory for `go build` |

### Custom Ready Checks

Use `ReadyCheckFunc` for custom readiness logic:

```go
workerBinary := testastic.BuildBinary(t, "./cmd/worker")

proc := workerBinary.Start(t.Context(), t,
    testastic.ReadyCheckFunc(func(ctx context.Context) bool {
        conn, err := net.DialTimeout("tcp", "localhost:6379", time.Second)
        if err != nil {
            return false
        }
        conn.Close()
        return true
    }),
)
```

### Collecting Coverage

By default, subprocess coverage data is written to a temp directory and cleaned up. To collect it into a standard Go coverage profile, add a `TestMain`:

```go
func TestMain(m *testing.M) {
    exitCode := testastic.CollectSubprocessCoverage(m, "coverage/process.out")
    cleanup()
    os.Exit(exitCode)
}
```

Run any package-level cleanup after `CollectSubprocessCoverage` returns and before `os.Exit(code)`. Avoid relying on `defer` for that cleanup, because `os.Exit` exits immediately without running deferred functions.

This produces a text profile compatible with `go tool cover`, codecov, and coveralls:

```bash
go tool cover -html=coverage/process.out
```

## Output

Colored diff output (red for expected, green for actual):

```
testastic: assertion failed

  Equal
    expected: "Alice"
    actual:   "Bob"
```

JSON/YAML mismatches show git-style inline diff:

```diff
testastic: assertion failed

  AssertJSON (testdata/user.expected.json)
  {
-   "name": "Alice",
+   "name": "Bob",
    "age": 30
  }
```
