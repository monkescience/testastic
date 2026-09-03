package testastic

import (
	"io"
	"strings"
	"sync"
)

const (
	maxCapturedOutputBytes = 1 << 20
	outputTruncatedMark    = "\n...[output truncated after 1048576 bytes]"
)

type capturedOutput struct {
	mu        sync.Mutex
	buf       strings.Builder
	truncated bool
}

var _ io.Writer = (*capturedOutput)(nil)

func (o *capturedOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.truncated {
		return len(p), nil
	}

	remaining := maxCapturedOutputBytes - o.buf.Len()
	if remaining > 0 {
		_, _ = o.buf.Write(p[:min(len(p), remaining)])
	}

	if len(p) > remaining {
		_, _ = o.buf.WriteString(outputTruncatedMark)
		o.truncated = true
	}

	return len(p), nil
}

func (o *capturedOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.buf.String()
}

func (o *capturedOutput) Len() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.buf.Len()
}
