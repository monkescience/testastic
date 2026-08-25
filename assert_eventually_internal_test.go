package testastic

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRunEventuallyCheck_DoesNotStartAfterDeadline(t *testing.T) {
	t.Parallel()

	// given: an Eventually check whose deadline has already passed
	var calls atomic.Int32

	results := make(chan eventuallyCheckResult[bool], 1)
	done := make(chan struct{})
	check := &eventuallyCheckState{}

	// when: the check worker tries to execute the callback
	runEventuallyCheck(func() bool {
		calls.Add(1)

		return false
	}, results, done, check, time.Now().Add(-time.Second))

	// then: the callback does not run or produce a result
	Equal(t, int32(0), calls.Load())

	select {
	case <-results:
		t.Error("expired check produced a result")
	default:
	}
}
