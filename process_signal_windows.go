//go:build windows

package testastic

import (
	"fmt"
	"os"
)

func interruptProcess(p *os.Process) error {
	// os.Process.Signal does not support os.Interrupt on Windows.
	err := p.Kill()
	if err != nil {
		return fmt.Errorf("killing process: %w", err)
	}

	return nil
}
