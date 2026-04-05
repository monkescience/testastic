//go:build windows

package testastic

import (
	"fmt"
	"os"
)

func interruptProcess(p *os.Process) error {
	err := p.Signal(os.Interrupt)
	if err != nil {
		return fmt.Errorf("sending interrupt: %w", err)
	}

	return nil
}
