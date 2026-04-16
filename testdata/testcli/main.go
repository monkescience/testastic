package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, "missing command")
		os.Exit(2)
	}

	switch os.Args[1] {
	case "stdout":
		fmt.Fprint(os.Stdout, os.Args[2])
	case "fail":
		exitCode, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid exit code: %v", err)
			os.Exit(2)
		}

		fmt.Fprint(os.Stderr, os.Args[3])
		os.Exit(exitCode)
	case "stdin":
		_, err := io.Copy(os.Stdout, os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "copy stdin: %v", err)
			os.Exit(1)
		}
	case "env":
		fmt.Fprint(os.Stdout, os.Getenv(os.Args[2]))
	case "cwd":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "getwd: %v", err)
			os.Exit(1)
		}

		fmt.Fprint(os.Stdout, wd)
	case "sleep":
		duration, err := time.ParseDuration(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid duration: %v", err)
			os.Exit(2)
		}

		time.Sleep(duration)
		fmt.Fprint(os.Stdout, "slept")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s", os.Args[1])
		os.Exit(2)
	}
}
