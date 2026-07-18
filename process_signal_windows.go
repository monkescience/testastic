//go:build windows

package testastic

import (
	"fmt"
	"os"
)

func interruptProcess(p *os.Process) error {
	if err := p.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("sending interrupt: %w", err)
	}

	return nil
}
