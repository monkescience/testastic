//go:build windows

package testastic

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestProcessCancellation_DoesNotWaitOnWindows(t *testing.T) {
	const helperEnv = "TESTASTIC_WINDOWS_CANCELLATION_HELPER"

	if os.Getenv(helperEnv) == "1" {
		time.Sleep(time.Hour)

		return
	}

	t.Parallel()

	// given: a Windows subprocess using the production cancellation path
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessCancellation_DoesNotWaitOnWindows$")
	cmd.Env = append(os.Environ(), helperEnv+"=1")
	cmd.Cancel = func() error {
		return interruptProcess(cmd.Process)
	}
	cmd.WaitDelay = defaultShutdownTimeout

	err := cmd.Start()
	if err != nil {
		t.Fatalf("start helper process: %v", err)
	}

	exited := make(chan struct{})

	go func() {
		_ = cmd.Wait()
		close(exited)
	}()

	// when: context cancellation requests process shutdown
	cancel()

	// then: the process exits before the shutdown timeout fallback
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	select {
	case <-exited:
	case <-timer.C:
		_ = cmd.Process.Kill()
		<-exited
		t.Fatal("Windows process cancellation waited for the shutdown timeout")
	}
}
