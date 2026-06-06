package testastic

import (
	"os"
	"sync"

	"golang.org/x/term"
)

// ANSI color codes.
const (
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorReset = "\033[0m"
)

// useColors reports whether colored output should be used. Precedence: NO_COLOR
// disables, then FORCE_COLOR forces on, then CI disables, then a "dumb" TERM
// disables, otherwise color is enabled only when stderr is a terminal (diff
// output is written on the stderr side of the test log).
// The result is detected once on first call and cached for the process lifetime.
var useColors = sync.OnceValue(detectColors)

func detectColors() bool {
	// Check NO_COLOR (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}

	if os.Getenv("CI") != "" {
		return false
	}

	if os.Getenv("TERM") == "dumb" {
		return false
	}

	fd, ok := stderrFD()
	if !ok {
		return false
	}

	return term.IsTerminal(fd)
}

func stderrFD() (int, bool) {
	fd := os.Stderr.Fd()
	if fd > uintptr(^uint(0)>>1) {
		return 0, false
	}

	return int(fd), true
}

func colorize(text, color string) string {
	if !useColors() {
		return text
	}

	return color + text + colorReset
}

// red returns text colored red (for removed lines).
func red(text string) string {
	return colorize(text, colorRed)
}

// green returns text colored green (for added lines).
func green(text string) string {
	return colorize(text, colorGreen)
}
