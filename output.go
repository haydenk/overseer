package main

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// ANSI color codes (foreground).
var colors = []string{
	"\033[36m", // cyan
	"\033[33m", // yellow
	"\033[32m", // green
	"\033[35m", // magenta
	"\033[31m", // red
	"\033[34m", // blue
}

const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"
)

// Writer multiplexes labeled, colored, optionally timestamped output to w.
type Writer struct {
	mu          sync.Mutex
	w           io.Writer
	maxWidth    int
	noTimestamp bool
	colorOutput bool
}

// NewWriter creates a Writer. maxWidth is the pad width for process labels.
func NewWriter(w io.Writer, maxWidth int, noTimestamp, colorOutput bool) *Writer {
	return &Writer{
		w:           w,
		maxWidth:    maxWidth,
		noTimestamp: noTimestamp,
		colorOutput: colorOutput,
	}
}

// WriteLine writes a single output line for a named process with its color index.
func (ow *Writer) WriteLine(colorIdx int, name, line string) {
	ow.mu.Lock()
	defer ow.mu.Unlock()

	prefix := ow.buildPrefix(colorIdx, name, false)
	fmt.Fprintf(ow.w, "%s%s\n", prefix, line)
}

// WriteSystem writes a system/status message (bold white).
func (ow *Writer) WriteSystem(name, line string) {
	ow.mu.Lock()
	defer ow.mu.Unlock()

	prefix := ow.buildPrefix(-1, name, true)
	fmt.Fprintf(ow.w, "%s%s\n", prefix, line)
}

func (ow *Writer) buildPrefix(colorIdx int, name string, system bool) string {
	ts := ""
	if !ow.noTimestamp {
		ts = time.Now().Format("15:04:05") + " "
	}

	padded := fmt.Sprintf("%-*s", ow.maxWidth, name)

	if !ow.colorOutput {
		return fmt.Sprintf("%s%s | ", ts, padded)
	}

	var color string
	if system {
		color = colorBold
	} else {
		color = colors[colorIdx%len(colors)]
	}

	return fmt.Sprintf("%s%s%s%s | ", ts, color, padded, colorReset)
}
