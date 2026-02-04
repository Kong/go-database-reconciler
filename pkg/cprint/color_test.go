package cprint

import (
	"bytes"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

// captureOutput captures color.Output and returns the recorded output as
// f runs.
// It is not thread-safe.
func captureOutput(f func()) string {
	backupOutput := color.Output
	defer func() {
		color.Output = backupOutput
	}()
	var out bytes.Buffer
	color.Output = &out
	f()
	return out.String()
}

// captureStderr captures os.Stderr and returns the recorded output as f runs.
// It is not thread-safe.
func captureStderr(f func()) string {
	// Create a pipe to capture stderr
	r, w, _ := os.Pipe()
	backupStderr := os.Stderr
	os.Stderr = w

	f()

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stderr = backupStderr

	return buf.String()
}

func TestMain(m *testing.M) {
	backup := color.NoColor
	color.NoColor = false
	exitVal := m.Run()
	color.NoColor = backup
	os.Exit(exitVal)
}

func TestPrint(t *testing.T) {
	tests := []struct {
		name          string
		DisableOutput bool
		Run           func()
		Expected      string
	}{
		{
			name:          "println prints colored output",
			DisableOutput: false,
			Run: func() {
				CreatePrintln("foo")
				UpdatePrintln("bar")
				DeletePrintln("fubaz")
			},
			Expected: "\x1b[32mfoo\x1b[0m\n\x1b[33mbar\x1b[0m\n\x1b[31mfubaz\x1b[0m\n",
		},
		{
			name:          "println doesn't output anything when disabled",
			DisableOutput: true,
			Run: func() {
				CreatePrintln("foo")
				UpdatePrintln("bar")
				DeletePrintln("fubaz")
			},
			Expected: "",
		},
		{
			name:          "printf prints colored output",
			DisableOutput: false,
			Run: func() {
				CreatePrintf("%s", "foo")
				UpdatePrintf("%s", "bar")
				DeletePrintf("%s", "fubaz")
			},
			Expected: "\x1b[32mfoo\x1b[0m\x1b[33mbar\x1b[0m\x1b[31mfubaz\x1b[0m",
		},
		{
			name:          "printf doesn't output anything when disabled",
			DisableOutput: true,
			Run: func() {
				CreatePrintln("foo")
				UpdatePrintln("bar")
				DeletePrintln("fubaz")
			},
			Expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DisableOutput = tt.DisableOutput
			defer func() {
				DisableOutput = false
			}()

			output := captureOutput(func() {
				tt.Run()
			})
			assert.Equal(t, tt.Expected, output)
		})
	}
}

func TestPrintStdErr(t *testing.T) {
	tests := []struct {
		name          string
		DisableOutput bool
		Run           func()
		Expected      string
	}{
		{
			name:          "UpdatePrintlnStdErr prints colored output to stderr",
			DisableOutput: false,
			Run: func() {
				UpdatePrintlnStdErr("warning message")
			},
			Expected: "\x1b[33mwarning message\x1b[0m\n",
		},
		{
			name:          "UpdatePrintlnStdErr doesn't output anything when disabled",
			DisableOutput: true,
			Run: func() {
				UpdatePrintlnStdErr("warning message")
			},
			Expected: "",
		},
		{
			name:          "UpdatePrintfStdErr prints colored formatted output to stderr",
			DisableOutput: false,
			Run: func() {
				UpdatePrintfStdErr("warning: %s %d", "count", 42)
			},
			Expected: "\x1b[33mwarning: count 42\x1b[0m",
		},
		{
			name:          "UpdatePrintfStdErr doesn't output anything when disabled",
			DisableOutput: true,
			Run: func() {
				UpdatePrintfStdErr("warning: %s", "test")
			},
			Expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			DisableOutput = tt.DisableOutput
			defer func() {
				DisableOutput = false
			}()

			output := captureStderr(func() {
				tt.Run()
			})
			assert.Equal(t, tt.Expected, output)
		})
	}
}

func TestStdErrFunctionsDoNotWriteToStdout(t *testing.T) {
	t.Run("UpdatePrintlnStdErr does not write to stdout", func(t *testing.T) {
		stdoutOutput := captureOutput(func() {
			UpdatePrintlnStdErr("this should not appear in stdout")
		})
		assert.Empty(t, stdoutOutput, "UpdatePrintlnStdErr should not write to stdout")
	})

	t.Run("UpdatePrintfStdErr does not write to stdout", func(t *testing.T) {
		stdoutOutput := captureOutput(func() {
			UpdatePrintfStdErr("this should not appear in stdout: %s", "test")
		})
		assert.Empty(t, stdoutOutput, "UpdatePrintfStdErr should not write to stdout")
	})
}

// captureStdoutAndStderr captures both stdout (via color.Output) and stderr simultaneously.
// Returns (stdout, stderr) content.
func captureStdoutAndStderr(f func()) (string, string) {
	// Capture stdout via color.Output
	backupOutput := color.Output
	var stdoutBuf bytes.Buffer
	color.Output = &stdoutBuf

	// Capture stderr via os.Stderr
	stderrR, stderrW, _ := os.Pipe()
	backupStderr := os.Stderr
	os.Stderr = stderrW

	f()

	// Restore and collect
	color.Output = backupOutput
	stderrW.Close()
	var stderrBuf bytes.Buffer
	stderrBuf.ReadFrom(stderrR)
	os.Stderr = backupStderr

	return stdoutBuf.String(), stderrBuf.String()
}

func TestStdErrOutputWithJSONMode(t *testing.T) {
	t.Run("stderr warnings appear while stdout contains JSON", func(t *testing.T) {
		// Simulate JSON output mode: stdout gets uncolored JSON, stderr gets warnings
		// This verifies that stderr functions don't corrupt JSON output on stdout
		// In JSON mode, color.NoColor should be true to disable color codes in stdout
		backup := color.NoColor
		color.NoColor = true
		defer func() { color.NoColor = backup }()

		stdout, stderr := captureStdoutAndStderr(func() {
			// Simulate JSON being written to stdout
			CreatePrintf(`{"changes": {"creating": []}}`)

			// Warning should go to stderr, not polluting the JSON
			UpdatePrintfStdErr("warning: %d unsupported routes detected", 1)
		})

		// Verify JSON on stdout is not corrupted by stderr output or color codes
		assert.Equal(t, `{"changes": {"creating": []}}`, stdout,
			"stdout should contain valid JSON without color codes")
		assert.NotContains(t, stdout, "warning",
			"stdout should not contain warning messages")
		assert.NotContains(t, stdout, "\x1b[",
			"stdout should not contain ANSI color codes in JSON mode")

		// Verify warning appears on stderr (also without color codes when NoColor is set)
		assert.Contains(t, stderr, "unsupported routes detected",
			"stderr should contain warning message")
	})

	t.Run("multiple stderr warnings don't affect stdout JSON", func(t *testing.T) {
		// Simulate JSON output mode with NoColor enabled
		backup := color.NoColor
		color.NoColor = true
		defer func() { color.NoColor = backup }()

		stdout, stderr := captureStdoutAndStderr(func() {
			// JSON output
			CreatePrintf(`{"status": "ok"}`)

			// Multiple warnings to stderr
			UpdatePrintlnStdErr("warning: deprecated feature used")
			UpdatePrintfStdErr("warning: %s configuration issue", "route")
		})

		// Stdout should only have JSON without color codes
		assert.Equal(t, `{"status": "ok"}`, stdout,
			"stdout should only contain JSON output without color codes")

		// Stderr should have all warnings
		assert.Contains(t, stderr, "deprecated feature used")
		assert.Contains(t, stderr, "route configuration issue")
	})
}
