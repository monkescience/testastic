package testastic

import (
	"os"

	"golang.org/x/term"
)

// ANSI color codes.
const (
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorReset = "\033[0m"
)

// colorsEnabled caches the color detection result.
var colorsEnabled *bool

// useColors returns true if colored output should be used.
// Colors are enabled when stdout is a terminal (not piped),
// NO_COLOR env var is not set, CI env var is not set,
// and TERM is not "dumb".
func useColors() bool {
	if colorsEnabled != nil {
		return *colorsEnabled
	}

	result := detectColors()
	colorsEnabled = &result

	return result
}

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
