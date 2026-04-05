//go:build !windows

package testastic

import (
	"fmt"
	"os"
	"syscall"
)

func interruptProcess(p *os.Process) error {
	err := p.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("sending SIGTERM: %w", err)
	}

	return nil
}
