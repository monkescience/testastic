package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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
	case "large-output":
		count, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid byte count: %v", err)
			os.Exit(2)
		}

		output := io.Discard
		switch os.Args[2] {
		case "stdout":
			output = os.Stdout
		case "stderr":
			output = os.Stderr
		case "both":
			output = io.MultiWriter(os.Stdout, os.Stderr)
		default:
			fmt.Fprintf(os.Stderr, "invalid output stream: %s", os.Args[2])
			os.Exit(2)
		}

		_, err = io.CopyN(output, zeroReader{}, int64(count))
		if err != nil {
			fmt.Fprintf(os.Stderr, "write output: %v", err)
			os.Exit(1)
		}
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
	case "spawn-inherited-output":
		duration, err := time.ParseDuration(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid duration: %v", err)
			os.Exit(2)
		}

		executable, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "find executable: %v", err)
			os.Exit(1)
		}

		child := exec.Command(executable, "sleep", os.Args[2])
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr

		err = child.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "start child: %v", err)
			os.Exit(1)
		}

		go func() {
			_ = child.Wait()
		}()

		fmt.Fprint(os.Stdout, "child started")
		time.Sleep(duration)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s", os.Args[1])
		os.Exit(2)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}

	return len(p), nil
}
