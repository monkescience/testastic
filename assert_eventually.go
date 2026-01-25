package testastic

import (
	"testing"
	"time"
)

// Default configuration values for Eventually.
const (
	defaultEventuallyInterval = 100 * time.Millisecond
)

// eventuallyConfig holds configuration for Eventually assertions.
type eventuallyConfig struct {
	Interval time.Duration
	Message  string
}

// EventuallyOption configures Eventually behavior.
type EventuallyOption func(*eventuallyConfig)

// WithInterval sets the polling interval between condition checks.
// Default is 100ms.
func WithInterval(d time.Duration) EventuallyOption {
	return func(c *eventuallyConfig) {
		c.Interval = d
	}
}

// WithMessage adds context to failure messages.
func WithMessage(msg string) EventuallyOption {
	return func(c *eventuallyConfig) {
		c.Message = msg
	}
}

// newEventuallyConfig creates a config with defaults and applies options.
func newEventuallyConfig(opts ...EventuallyOption) *eventuallyConfig {
	cfg := &eventuallyConfig{
		Interval: defaultEventuallyInterval,
	}

	for _, opt := range opts {
		opt(cfg)
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
	tb testing.TB,
	name string,
	getValue func() T,
	condition func(T) bool,
	formatFailure func(lastValue T) string,
	timeout time.Duration,
	cfg *eventuallyConfig,
) {
	tb.Helper()

	var lastValue T

	check := func() bool {
		lastValue = getValue()

		return condition(lastValue)
	}

	// Check immediately first.
	if check() {
		return
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(cfg.Interval)

	defer ticker.Stop()

	for {
		<-ticker.C

		if check() {
			return
		}

		if time.Now().After(deadline) {
			msg := ""
			if cfg.Message != "" {
				msg = "\n    message:  " + cfg.Message
			}

			tb.Errorf(
				"testastic: assertion failed\n\n  %s%s\n    timed out after %v%s",
				name, formatFailure(lastValue), timeout, msg,
			)

			return
		}
	}
}

// Eventually retries a condition function until it returns true or timeout is reached.
// The condition is checked immediately, then at regular intervals (default 100ms).
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
