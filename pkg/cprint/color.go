package cprint

import (
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
)

var (
	// mu is used to synchronize writes from multiple goroutines.
	mu sync.Mutex
	// DisableOutput disables all output.
	DisableOutput bool
)

func conditionalPrintf(fn func(string, ...any), format string, a ...any) {
	if DisableOutput {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn(format, a...)
}

func conditionalPrintln(fn func(...any), a ...any) {
	if DisableOutput {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn(a...)
}

func conditionalPrintlnCustomWriter(fn func(io.Writer, ...any), w io.Writer, a ...any) {
	if DisableOutput {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn(w, a...)
}

func conditionalPrintfCustomWriter(
	fn func(io.Writer, string, ...any), w io.Writer, format string, a ...any,
) {
	if DisableOutput {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn(w, format, a...)
}

var (
	createPrintf = color.New(color.FgGreen).PrintfFunc()
	deletePrintf = color.New(color.FgRed).PrintfFunc()
	updatePrintf = color.New(color.FgYellow).PrintfFunc()

	// CreatePrintf is fmt.Printf with red as foreground color.
	CreatePrintf = func(format string, a ...any) {
		conditionalPrintf(createPrintf, format, a...)
	}

	// DeletePrintf is fmt.Printf with green as foreground color.
	DeletePrintf = func(format string, a ...any) {
		conditionalPrintf(deletePrintf, format, a...)
	}

	// UpdatePrintf is fmt.Printf with yellow as foreground color.
	UpdatePrintf = func(format string, a ...any) {
		conditionalPrintf(updatePrintf, format, a...)
	}

	createPrintln  = color.New(color.FgGreen).PrintlnFunc()
	deletePrintln  = color.New(color.FgRed).PrintlnFunc()
	updatePrintln  = color.New(color.FgYellow).PrintlnFunc()
	bluePrintln    = color.New(color.BgBlue).PrintlnFunc()
	updateFprintln = color.New(color.FgYellow).FprintlnFunc()
	updateFprintf  = color.New(color.FgYellow).FprintfFunc()

	// CreatePrintln is fmt.Println with red as foreground color.
	CreatePrintln = func(a ...any) {
		conditionalPrintln(createPrintln, a...)
	}

	// DeletePrintln is fmt.Println with green as foreground color.
	DeletePrintln = func(a ...any) {
		conditionalPrintln(deletePrintln, a...)
	}

	// UpdatePrintln is fmt.Println with yellow as foreground color.
	UpdatePrintln = func(a ...any) {
		conditionalPrintln(updatePrintln, a...)
	}

	BluePrintLn = func(a ...any) {
		conditionalPrintln(bluePrintln, a...)
	}

	// UpdatePrintlnStdErr is fmt.Println with yellow as foreground color.
	// It prints to stderr, instead of stdout
	UpdatePrintlnStdErr = func(a ...any) {
		conditionalPrintlnCustomWriter(updateFprintln, os.Stderr, a...)
	}

	// UpdatePrintfStdErr is fmt.Printf with yellow as foreground color.
	// It prints to stderr, instead of stdout
	UpdatePrintfStdErr = func(format string, a ...any) {
		conditionalPrintfCustomWriter(updateFprintf, os.Stderr, format, a...)
	}
)
