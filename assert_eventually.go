package testastic

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// Default configuration values for Eventually.
const defaultEventuallyInterval = 100 * time.Millisecond

const (
	eventuallyCheckPending uint32 = iota
	eventuallyCheckRunning
	eventuallyCheckCanceled
)

type eventuallyConfig struct {
	Interval time.Duration
	Message  string
}

type eventuallyCheckResult[T any] struct {
	value      T
	panicValue any
	panicked   bool
	returned   bool
}

type eventuallyCheckState struct {
	state atomic.Uint32
}

func (s *eventuallyCheckState) claim() bool {
	return s.state.CompareAndSwap(eventuallyCheckPending, eventuallyCheckRunning)
}

func (s *eventuallyCheckState) cancel() {
	s.state.CompareAndSwap(eventuallyCheckPending, eventuallyCheckCanceled)
}

// EventuallyOption configures Eventually behavior using functional options
// (see [WithInterval] and [WithMessage]).
type EventuallyOption func(*eventuallyConfig)

// WithInterval sets the polling interval between condition checks.
// Default is 100ms.
func WithInterval(d time.Duration) EventuallyOption {
	return func(c *eventuallyConfig) {
		c.Interval = d
	}
}

// WithMessage adds context to the timeout failure message, helping identify
// which Eventually assertion failed when a test has multiple.
func WithMessage(msg string) EventuallyOption {
	return func(c *eventuallyConfig) {
		c.Message = msg
	}
}

func newEventuallyConfig(opts ...EventuallyOption) *eventuallyConfig {
	cfg := &eventuallyConfig{
		Interval: defaultEventuallyInterval,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.Interval <= 0 {
		cfg.Interval = defaultEventuallyInterval
	}

	return cfg
}

// eventually is the core retry logic shared by all Eventually variants.
func eventually(
	tb testing.TB, name string, condition func() bool, timeout time.Duration, cfg *eventuallyConfig,
) {
	tb.Helper()

	eventuallyWithValue(tb, name, condition, func(v bool) bool { return v }, func(bool) string { return "" }, timeout, cfg)
}

// eventuallyWithValue is a generic retry helper that captures the last value for error formatting.
func eventuallyWithValue[T any](
	tb testing.TB, name string, getValue func() T, condition func(T) bool,
	formatFailure func(lastValue T) string, timeout time.Duration, cfg *eventuallyConfig,
) {
	tb.Helper()

	if timeout <= 0 {
		reportEventuallyInvalidTimeout(tb, name, timeout)

		return
	}

	pollEventuallyWithValue(tb, name, getValue, condition, formatFailure, timeout, cfg)
}

func pollEventuallyWithValue[T any](
	tb testing.TB, name string, getValue func() T, condition func(T) bool,
	formatFailure func(lastValue T) string, timeout time.Duration, cfg *eventuallyConfig,
) {
	tb.Helper()

	var lastValue T

	results := make(chan eventuallyCheckResult[T])
	done := make(chan struct{})

	defer close(done)

	checking := true

	// Start the timer before the first check so it bounds the whole assertion.
	deadline := time.Now().Add(timeout)

	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()

	ticker := time.NewTicker(cfg.Interval)

	ticker.Stop()
	defer ticker.Stop()

	var ticks <-chan time.Time

	currentCheck := launchEventuallyCheck(getValue, results, done, deadline)

	for {
		select {
		case result := <-results:
			checking = false

			if eventuallyResultMatches(result, &lastValue, condition) {
				return
			}

			if ticks == nil {
				ticks = activateEventuallyTicker(ticker, cfg.Interval)
			}

		case <-ticks:
			if !checking && time.Now().Before(deadline) {
				checking = true
				currentCheck = launchEventuallyCheck(getValue, results, done, deadline)
			}

		case <-timer.C:
			currentCheck.cancel()

			if availableEventuallyResultMatches(results, &lastValue, condition) {
				return
			}

			reportEventuallyTimeout(tb, name, lastValue, formatFailure, timeout, cfg.Message)

			return
		}
	}
}

func activateEventuallyTicker(ticker *time.Ticker, interval time.Duration) <-chan time.Time {
	ticker.Reset(interval)

	return ticker.C
}

func launchEventuallyCheck[T any](
	getValue func() T, results chan<- eventuallyCheckResult[T], done <-chan struct{}, deadline time.Time,
) *eventuallyCheckState {
	check := &eventuallyCheckState{}

	go runEventuallyCheck(getValue, results, done, check, deadline)

	return check
}

func runEventuallyCheck[T any](
	getValue func() T,
	results chan<- eventuallyCheckResult[T],
	done <-chan struct{},
	check *eventuallyCheckState,
	deadline time.Time,
) {
	if !check.claim() {
		return
	}

	if !time.Now().Before(deadline) {
		return
	}

	result := eventuallyCheckResult[T]{}

	defer func() {
		select {
		case results <- result:
		case <-done:
			if result.panicked {
				panic(result.panicValue)
			}
		}
	}()

	func() {
		defer func() {
			result.panicValue = recover()
		}()

		result.value = getValue()
		result.returned = true
	}()

	result.panicked = !result.returned
}

func eventuallyResultMatches[T any](
	result eventuallyCheckResult[T], lastValue *T, condition func(T) bool,
) bool {
	if result.panicked {
		panic(result.panicValue)
	}

	if !result.returned {
		runtime.Goexit()
	}

	*lastValue = result.value

	return condition(result.value)
}

func availableEventuallyResultMatches[T any](
	results <-chan eventuallyCheckResult[T], lastValue *T, condition func(T) bool,
) bool {
	select {
	case result := <-results:
		return eventuallyResultMatches(result, lastValue, condition)
	default:
		return false
	}
}

func reportEventuallyInvalidTimeout(tb testing.TB, name string, timeout time.Duration) {
	tb.Helper()

	tb.Errorf(
		"testastic: assertion failed\n\n  %s\n    timeout must be greater than zero, got %v",
		name,
		timeout,
	)
}

func reportEventuallyTimeout[T any](
	tb testing.TB,
	name string,
	lastValue T,
	formatFailure func(lastValue T) string,
	timeout time.Duration,
	message string,
) {
	tb.Helper()

	if message != "" {
		message = "\n    message:  " + message
	}

	tb.Errorf(
		"testastic: assertion failed\n\n  %s%s\n    timed out after %v%s",
		name, formatFailure(lastValue), timeout, message,
	)
}

// Eventually retries a condition function until it returns true or timeout is reached.
// Timeout must be greater than zero. The first condition check is scheduled
// immediately, then checks run at regular intervals (default 100ms).
// A check already in progress at the deadline may continue after this function returns.
//
// Example:
//
//	testastic.Eventually(t, func() bool {
//	    return server.IsReady()
//	}, 5*time.Second)
//
//	testastic.Eventually(t, func() bool {
//	    return len(queue) > 0
//	}, 2*time.Second, testastic.WithInterval(50*time.Millisecond))
func Eventually(tb testing.TB, condition func() bool, timeout time.Duration, opts ...EventuallyOption) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)
	eventually(tb, "Eventually", condition, timeout, cfg)
}

// EventuallyTrue is an alias for Eventually for readability.
// Timeout behavior matches [Eventually].
//
// Example:
//
//	testastic.EventuallyTrue(t, func() bool {
//	    return isReady
//	}, 3*time.Second)
func EventuallyTrue(tb testing.TB, condition func() bool, timeout time.Duration, opts ...EventuallyOption) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)
	eventually(tb, "EventuallyTrue", condition, timeout, cfg)
}

// EventuallyFalse retries until condition returns false.
// Timeout behavior matches [Eventually].
//
// Example:
//
//	testastic.EventuallyFalse(t, func() bool {
//	    return server.IsProcessing()
//	}, 5*time.Second)
func EventuallyFalse(tb testing.TB, condition func() bool, timeout time.Duration, opts ...EventuallyOption) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)
	eventually(tb, "EventuallyFalse", func() bool { return !condition() }, timeout, cfg)
}

// EventuallyEqual retries until expected equals the result of getValue.
// Timeout behavior matches [Eventually].
//
// Example:
//
//	testastic.EventuallyEqual(t, "ready", func() string {
//	    return service.Status()
//	}, 3*time.Second)
func EventuallyEqual[T comparable](
	tb testing.TB, expected T, getValue func() T, timeout time.Duration, opts ...EventuallyOption,
) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)

	eventuallyWithValue(
		tb,
		"EventuallyEqual",
		getValue,
		func(v T) bool { return v == expected },
		func(lastValue T) string {
			return "\n    expected: " + red(formatVal(expected)) + "\n    actual:   " + green(formatVal(lastValue))
		},
		timeout,
		cfg,
	)
}

// EventuallyNil retries until getValue returns nil.
// Timeout behavior matches [Eventually].
//
// Example:
//
//	testastic.EventuallyNil(t, func() any {
//	    return cache.Get("key")
//	}, 2*time.Second)
func EventuallyNil(tb testing.TB, getValue func() any, timeout time.Duration, opts ...EventuallyOption) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)
	eventually(tb, "EventuallyNil", func() bool { return isNil(getValue()) }, timeout, cfg)
}

// EventuallyNotNil retries until getValue returns a non-nil value.
// Timeout behavior matches [Eventually].
//
// Example:
//
//	testastic.EventuallyNotNil(t, func() any {
//	    return cache.Get("key")
//	}, 2*time.Second)
func EventuallyNotNil(tb testing.TB, getValue func() any, timeout time.Duration, opts ...EventuallyOption) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)
	eventually(tb, "EventuallyNotNil", func() bool { return !isNil(getValue()) }, timeout, cfg)
}

// EventuallyNoError retries until getErr returns nil.
// Timeout behavior matches [Eventually].
//
// Example:
//
//	testastic.EventuallyNoError(t, func() error {
//	    _, err := client.Ping()
//	    return err
//	}, 5*time.Second)
func EventuallyNoError(tb testing.TB, getErr func() error, timeout time.Duration, opts ...EventuallyOption) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)

	eventuallyWithValue(
		tb,
		"EventuallyNoError",
		getErr,
		func(err error) bool { return err == nil },
		func(lastErr error) string {
			errStr := "nil"
			if lastErr != nil {
				errStr = lastErr.Error()
			}

			return "\n    last error: " + red(errStr)
		},
		timeout,
		cfg,
	)
}

// EventuallyError retries until getErr returns a non-nil error.
// Timeout behavior matches [Eventually].
//
// Example:
//
//	testastic.EventuallyError(t, func() error {
//	    return service.HealthCheck()
//	}, 3*time.Second)
func EventuallyError(tb testing.TB, getErr func() error, timeout time.Duration, opts ...EventuallyOption) {
	tb.Helper()

	cfg := newEventuallyConfig(opts...)
	eventually(tb, "EventuallyError", func() bool { return getErr() != nil }, timeout, cfg)
}
