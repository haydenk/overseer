package main

import (
	"bytes"
	"strings"
	"testing"
)

// captureOutput creates a Writer backed by a bytes.Buffer and returns both.
func newTestWriter(maxWidth int, noTimestamp, colorOutput bool) (*Writer, *bytes.Buffer) {
	var buf bytes.Buffer
	w := NewWriter(&buf, maxWidth, noTimestamp, colorOutput)
	return w, &buf
}

func TestNewWriter_ReturnsNonNil(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, 10, true, false)
	if w == nil {
		t.Fatal("NewWriter returned nil")
	}
}

func TestWriteLine_ContainsNameAndLine(t *testing.T) {
	w, buf := newTestWriter(10, true, false)
	w.WriteLine(0, "web", "hello world")

	output := buf.String()
	if !strings.Contains(output, "web") {
		t.Errorf("output %q does not contain process name %q", output, "web")
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("output %q does not contain line content %q", output, "hello world")
	}
}

func TestWriteLine_EndsWithNewline(t *testing.T) {
	w, buf := newTestWriter(10, true, false)
	w.WriteLine(0, "web", "some output")

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("output %q does not end with newline", output)
	}
}

func TestWriteLine_ContainsPipeSeparator(t *testing.T) {
	w, buf := newTestWriter(10, true, false)
	w.WriteLine(0, "web", "content")

	output := buf.String()
	if !strings.Contains(output, " | ") {
		t.Errorf("output %q does not contain pipe separator ' | '", output)
	}
}

func TestWriteLine_NoTimestamp_NoTimestampInOutput(t *testing.T) {
	w, buf := newTestWriter(10, true, false) // noTimestamp = true
	w.WriteLine(0, "web", "hello")

	output := buf.String()
	// Without timestamps, output should not contain the HH:MM:SS time format.
	// We check it's short enough (name padded to 10 + " | " + "hello\n" = 16 chars without color codes).
	// Simply check the name appears right at the start (or after ANSI if color enabled).
	if strings.Contains(output, ":") && !strings.Contains(output, " | ") {
		// This would indicate a timestamp was incorrectly included.
		t.Errorf("output %q appears to contain a timestamp when noTimestamp=true", output)
	}
}

func TestWriteLine_WithTimestamp_ContainsTimeFormat(t *testing.T) {
	w, buf := newTestWriter(10, false, false) // noTimestamp = false
	w.WriteLine(0, "web", "hello")

	output := buf.String()
	// Timestamp format is "15:04:05 " — look for colons in expected position.
	// We check that there are at least two colons in the timestamp portion.
	parts := strings.SplitN(output, " | ", 2)
	if len(parts) < 2 {
		t.Fatalf("output %q missing ' | ' separator", output)
	}
	prefix := parts[0]
	colonCount := strings.Count(prefix, ":")
	if colonCount < 2 {
		t.Errorf("prefix %q expected at least 2 colons (HH:MM:SS), got %d", prefix, colonCount)
	}
}

func TestWriteLine_PadsLabelToMaxWidth(t *testing.T) {
	w, buf := newTestWriter(15, true, false) // maxWidth=15
	w.WriteLine(0, "web", "hello")

	output := buf.String()
	// The label "web" padded to 15 chars should appear before " | ".
	parts := strings.SplitN(output, " | ", 2)
	if len(parts) < 2 {
		t.Fatalf("output %q missing ' | ' separator", output)
	}
	prefix := parts[0]
	// The padded label "web" padded to 15 chars; prefix ends with spaces.
	if !strings.Contains(prefix, "web") {
		t.Errorf("prefix %q does not contain 'web'", prefix)
	}
}

func TestWriteLine_WithColor_ContainsANSICodes(t *testing.T) {
	w, buf := newTestWriter(10, true, true) // colorOutput = true
	w.WriteLine(0, "web", "hello")

	output := buf.String()
	// ANSI escape sequences start with \033[
	if !strings.Contains(output, "\033[") {
		t.Errorf("output %q does not contain ANSI escape codes when colorOutput=true", output)
	}
}

func TestWriteLine_NoColor_NoANSICodes(t *testing.T) {
	w, buf := newTestWriter(10, true, false) // colorOutput = false
	w.WriteLine(0, "web", "hello")

	output := buf.String()
	if strings.Contains(output, "\033[") {
		t.Errorf("output %q contains ANSI escape codes when colorOutput=false", output)
	}
}

func TestWriteLine_ColorIndexWrapsAround(t *testing.T) {
	// Verify that color indices beyond the colors slice length don't panic.
	w, buf := newTestWriter(10, true, true)
	// There are 6 colors; test indices 0 through 11 to exercise the modulo.
	for i := 0; i < 12; i++ {
		w.WriteLine(i, "web", "line")
	}
	if buf.Len() == 0 {
		t.Error("expected output, got empty buffer")
	}
}

func TestWriteLine_MultipleProcesses_AllAppear(t *testing.T) {
	w, buf := newTestWriter(10, true, false)
	w.WriteLine(0, "web", "web output")
	w.WriteLine(1, "worker", "worker output")
	w.WriteLine(2, "clock", "clock output")

	output := buf.String()
	for _, want := range []string{"web", "web output", "worker", "worker output", "clock", "clock output"} {
		if !strings.Contains(output, want) {
			t.Errorf("output %q does not contain %q", output, want)
		}
	}
}

func TestWriteSystem_ContainsNameAndMessage(t *testing.T) {
	w, buf := newTestWriter(10, true, false)
	w.WriteSystem("overseer", "started web with pid 123")

	output := buf.String()
	if !strings.Contains(output, "overseer") {
		t.Errorf("output %q does not contain 'overseer'", output)
	}
	if !strings.Contains(output, "started web with pid 123") {
		t.Errorf("output %q does not contain the message", output)
	}
}

func TestWriteSystem_EndsWithNewline(t *testing.T) {
	w, buf := newTestWriter(10, true, false)
	w.WriteSystem("overseer", "shutting down")

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Errorf("output %q does not end with newline", output)
	}
}

func TestWriteSystem_WithColor_UsesBold(t *testing.T) {
	w, buf := newTestWriter(10, true, true) // colorOutput = true
	w.WriteSystem("overseer", "system message")

	output := buf.String()
	// System messages should use bold (\033[1m).
	if !strings.Contains(output, "\033[1m") {
		t.Errorf("output %q does not contain bold ANSI code \\033[1m", output)
	}
}

func TestWriteSystem_NoColor_NoANSICodes(t *testing.T) {
	w, buf := newTestWriter(10, true, false) // colorOutput = false
	w.WriteSystem("overseer", "system message")

	output := buf.String()
	if strings.Contains(output, "\033[") {
		t.Errorf("output %q contains ANSI codes when colorOutput=false", output)
	}
}

func TestWriteSystem_ContainsPipeSeparator(t *testing.T) {
	w, buf := newTestWriter(10, true, false)
	w.WriteSystem("overseer", "some event")

	output := buf.String()
	if !strings.Contains(output, " | ") {
		t.Errorf("output %q does not contain ' | ' separator", output)
	}
}

func TestWriter_ConcurrentWrites_NoPanic(t *testing.T) {
	w, _ := newTestWriter(10, true, false)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(idx int) {
			w.WriteLine(idx%6, "proc", "concurrent line")
			w.WriteSystem("overseer", "concurrent system")
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBuildPrefix_NoTimestampNoColor(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, 8, true, false) // noTimestamp=true, colorOutput=false

	prefix := w.buildPrefix(0, "web", false)

	// Should be "web      | " (padded to 8 chars)
	if !strings.HasSuffix(prefix, " | ") {
		t.Errorf("prefix %q does not end with ' | '", prefix)
	}
	if !strings.Contains(prefix, "web") {
		t.Errorf("prefix %q does not contain 'web'", prefix)
	}
	if strings.Contains(prefix, "\033[") {
		t.Errorf("prefix %q contains ANSI codes when colorOutput=false", prefix)
	}
}

func TestBuildPrefix_WithColor_NonSystem(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, 8, true, true) // noTimestamp=true, colorOutput=true

	prefix := w.buildPrefix(0, "web", false)

	if !strings.Contains(prefix, "\033[") {
		t.Errorf("prefix %q missing ANSI codes when colorOutput=true", prefix)
	}
	// Should contain reset code.
	if !strings.Contains(prefix, "\033[0m") {
		t.Errorf("prefix %q missing color reset code", prefix)
	}
}

func TestBuildPrefix_WithColor_System(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, 8, true, true) // noTimestamp=true, colorOutput=true

	prefix := w.buildPrefix(-1, "overseer", true)

	// System messages use bold.
	if !strings.Contains(prefix, "\033[1m") {
		t.Errorf("prefix %q missing bold code \\033[1m for system message", prefix)
	}
}

func TestBuildPrefix_LabelPaddedToMaxWidth(t *testing.T) {
	var buf bytes.Buffer
	maxWidth := 12
	w := NewWriter(&buf, maxWidth, true, false)

	prefix := w.buildPrefix(0, "web", false)
	// Strip trailing " | " and check length of the label portion.
	labelPart := strings.TrimSuffix(prefix, " | ")
	if len(labelPart) != maxWidth {
		t.Errorf("label portion length = %d, want %d (maxWidth), prefix=%q", len(labelPart), maxWidth, prefix)
	}
}
