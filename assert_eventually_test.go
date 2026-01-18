package testastic_test

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/monkescience/testastic"
)

// --- Eventually Tests ---

func TestEventually_ImmediateSuccess(t *testing.T) {
	// given: a condition that is immediately true
	// when: calling Eventually
	// then: it succeeds without waiting
	start := time.Now()

	testastic.Eventually(t, func() bool {
		return true
	}, 1*time.Second)

	elapsed := time.Since(start)
	if elapsed > 50*time.Millisecond {
		t.Errorf("expected immediate success, took %v", elapsed)
	}
}

func TestEventually_SuccessAfterRetries(t *testing.T) {
	// given: a condition that becomes true after a few attempts
	var counter atomic.Int32

	condition := func() bool {
		return counter.Add(1) >= 3
	}

	// when: calling Eventually with short interval
	testastic.Eventually(t, condition, 1*time.Second, testastic.WithInterval(10*time.Millisecond))

	// then: it succeeds after retries
	if counter.Load() < 3 {
		t.Errorf("expected at least 3 calls, got %d", counter.Load())
	}
}

func TestEventually_Timeout(t *testing.T) {
	// given: a condition that never becomes true
	mt := newMockT()

	// when: calling Eventually with short timeout
	testastic.Eventually(mt, func() bool {
		return false
	}, 100*time.Millisecond, testastic.WithInterval(20*time.Millisecond))

	// then: it fails with timeout error
	if !mt.failed {
		t.Error("expected Eventually to fail on timeout")
	}

	if !strings.Contains(mt.message, "timed out") {
		t.Errorf("expected timeout message, got: %s", mt.message)
	}
}

func TestEventually_WithMessage(t *testing.T) {
	// given: a condition that fails with a custom message
	mt := newMockT()

	// when: calling Eventually with WithMessage
	testastic.Eventually(mt, func() bool {
		return false
	}, 50*time.Millisecond, testastic.WithMessage("waiting for server"))

	// then: the failure message includes the custom message
	if !mt.failed {
		t.Error("expected Eventually to fail")
	}

	if !strings.Contains(mt.message, "waiting for server") {
		t.Errorf("expected custom message, got: %s", mt.message)
	}
}

func TestEventually_WithInterval(t *testing.T) {
	// given: a condition that counts calls
	var counter atomic.Int32

	condition := func() bool {
		counter.Add(1)

		return false
	}

	mt := newMockT()

	// when: calling Eventually with 50ms interval for 150ms
	testastic.Eventually(mt, condition, 150*time.Millisecond, testastic.WithInterval(50*time.Millisecond))

	// then: it should be called approximately 3-4 times (initial + 2-3 retries)
	calls := counter.Load()
	if calls < 2 || calls > 5 {
		t.Errorf("expected 2-5 calls with 50ms interval over 150ms, got %d", calls)
	}
}

// --- EventuallyTrue Tests ---

func TestEventuallyTrue_Pass(t *testing.T) {
	// given: a condition that becomes true
	var ready atomic.Bool

	go func() {
		time.Sleep(50 * time.Millisecond)
		ready.Store(true)
	}()

	// when: calling EventuallyTrue
	// then: it succeeds when the condition becomes true
	testastic.EventuallyTrue(t, ready.Load, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

// --- EventuallyFalse Tests ---

func TestEventuallyFalse_Pass(t *testing.T) {
	// given: a condition that becomes false
	var processing atomic.Bool

	processing.Store(true)

	go func() {
		time.Sleep(50 * time.Millisecond)
		processing.Store(false)
	}()

	// when: calling EventuallyFalse
	// then: it succeeds when the condition becomes false
	testastic.EventuallyFalse(t, processing.Load, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

func TestEventuallyFalse_Timeout(t *testing.T) {
	// given: a condition that stays true
	mt := newMockT()

	// when: calling EventuallyFalse
	testastic.EventuallyFalse(mt, func() bool {
		return true
	}, 50*time.Millisecond)

	// then: it fails on timeout
	if !mt.failed {
		t.Error("expected EventuallyFalse to fail")
	}
}

// --- EventuallyEqual Tests ---

func TestEventuallyEqual_ImmediateSuccess(t *testing.T) {
	// given: a value that immediately matches
	// when: calling EventuallyEqual
	// then: it succeeds immediately
	testastic.EventuallyEqual(t, "ready", func() string {
		return "ready"
	}, 1*time.Second)
}

func TestEventuallyEqual_SuccessAfterChange(t *testing.T) {
	// given: a value that changes to match expected
	var status atomic.Value

	status.Store("starting")

	go func() {
		time.Sleep(50 * time.Millisecond)
		status.Store("ready")
	}()

	// when: calling EventuallyEqual
	// then: it succeeds when the value matches
	testastic.EventuallyEqual(t, "ready", func() string {
		v, ok := status.Load().(string)
		if !ok {
			return ""
		}

		return v
	}, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

func TestEventuallyEqual_Timeout(t *testing.T) {
	// given: a value that never matches
	mt := newMockT()

	// when: calling EventuallyEqual
	testastic.EventuallyEqual(mt, "expected", func() string {
		return "actual"
	}, 50*time.Millisecond)

	// then: it fails with expected and actual values
	if !mt.failed {
		t.Error("expected EventuallyEqual to fail")
	}

	if !strings.Contains(mt.message, "expected") {
		t.Errorf("expected message to contain 'expected', got: %s", mt.message)
	}

	if !strings.Contains(mt.message, "actual") {
		t.Errorf("expected message to contain 'actual', got: %s", mt.message)
	}
}

func TestEventuallyEqual_Integer(t *testing.T) {
	// given: an integer counter
	var counter atomic.Int32

	go func() {
		for range 5 {
			time.Sleep(20 * time.Millisecond)
			counter.Add(1)
		}
	}()

	// when: calling EventuallyEqual with integer type
	// then: it succeeds when the value matches
	testastic.EventuallyEqual(t, int32(5), counter.Load, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

// --- EventuallyNil Tests ---

func TestEventuallyNil_ImmediateSuccess(t *testing.T) {
	// given: a value that is immediately nil
	// when: calling EventuallyNil
	// then: it succeeds immediately
	testastic.EventuallyNil(t, func() any {
		return nil
	}, 1*time.Second)
}

func TestEventuallyNil_SuccessAfterClear(t *testing.T) {
	// given: a value that becomes nil (using pointer to track nil state)
	var cleared atomic.Bool

	go func() {
		time.Sleep(50 * time.Millisecond)
		cleared.Store(true)
	}()

	// when: calling EventuallyNil
	// then: it succeeds when the value becomes nil
	testastic.EventuallyNil(t, func() any {
		if cleared.Load() {
			return nil
		}

		return "something"
	}, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

func TestEventuallyNil_Timeout(t *testing.T) {
	// given: a value that is never nil
	mt := newMockT()

	// when: calling EventuallyNil
	testastic.EventuallyNil(mt, func() any {
		return "not nil"
	}, 50*time.Millisecond)

	// then: it fails on timeout
	if !mt.failed {
		t.Error("expected EventuallyNil to fail")
	}
}

// --- EventuallyNotNil Tests ---

func TestEventuallyNotNil_ImmediateSuccess(t *testing.T) {
	// given: a value that is immediately not nil
	// when: calling EventuallyNotNil
	// then: it succeeds immediately
	testastic.EventuallyNotNil(t, func() any {
		return "value"
	}, 1*time.Second)
}

func TestEventuallyNotNil_SuccessAfterSet(t *testing.T) {
	// given: a value that becomes non-nil
	var value atomic.Value

	go func() {
		time.Sleep(50 * time.Millisecond)
		value.Store("something")
	}()

	// when: calling EventuallyNotNil
	// then: it succeeds when the value becomes non-nil
	testastic.EventuallyNotNil(t, value.Load, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

func TestEventuallyNotNil_Timeout(t *testing.T) {
	// given: a value that is always nil
	mt := newMockT()

	// when: calling EventuallyNotNil
	testastic.EventuallyNotNil(mt, func() any {
		return nil
	}, 50*time.Millisecond)

	// then: it fails on timeout
	if !mt.failed {
		t.Error("expected EventuallyNotNil to fail")
	}
}

// --- EventuallyNoError Tests ---

func TestEventuallyNoError_ImmediateSuccess(t *testing.T) {
	// given: a function that returns no error
	// when: calling EventuallyNoError
	// then: it succeeds immediately
	testastic.EventuallyNoError(t, func() error {
		return nil
	}, 1*time.Second)
}

func TestEventuallyNoError_SuccessAfterRecovery(t *testing.T) {
	// given: a function that eventually succeeds
	var counter atomic.Int32

	getErr := func() error {
		if counter.Add(1) < 3 {
			return errors.New("not ready")
		}

		return nil
	}

	// when: calling EventuallyNoError
	// then: it succeeds after retries
	testastic.EventuallyNoError(t, getErr, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

func TestEventuallyNoError_Timeout(t *testing.T) {
	// given: a function that always returns an error
	mt := newMockT()

	// when: calling EventuallyNoError
	testastic.EventuallyNoError(mt, func() error {
		return errors.New("persistent error")
	}, 50*time.Millisecond)

	// then: it fails and shows the last error
	if !mt.failed {
		t.Error("expected EventuallyNoError to fail")
	}

	if !strings.Contains(mt.message, "persistent error") {
		t.Errorf("expected message to contain last error, got: %s", mt.message)
	}
}

// --- EventuallyError Tests ---

func TestEventuallyError_ImmediateSuccess(t *testing.T) {
	// given: a function that returns an error
	// when: calling EventuallyError
	// then: it succeeds immediately
	testastic.EventuallyError(t, func() error {
		return errors.New("expected error")
	}, 1*time.Second)
}

func TestEventuallyError_SuccessAfterFailure(t *testing.T) {
	// given: a function that eventually fails
	var counter atomic.Int32

	getErr := func() error {
		if counter.Add(1) < 3 {
			return nil
		}

		return errors.New("service unhealthy")
	}

	// when: calling EventuallyError
	// then: it succeeds when an error is returned
	testastic.EventuallyError(t, getErr, 1*time.Second, testastic.WithInterval(10*time.Millisecond))
}

func TestEventuallyError_Timeout(t *testing.T) {
	// given: a function that never returns an error
	mt := newMockT()

	// when: calling EventuallyError
	testastic.EventuallyError(mt, func() error {
		return nil
	}, 50*time.Millisecond)

	// then: it fails on timeout
	if !mt.failed {
		t.Error("expected EventuallyError to fail")
	}
}
