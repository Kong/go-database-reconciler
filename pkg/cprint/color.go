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

func conditionalPrintf(fn func(string, ...interface{}), format string, a ...interface{}) {
	if DisableOutput {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn(format, a...)
}

func conditionalPrintln(fn func(...interface{}), a ...interface{}) {
	if DisableOutput {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn(a...)
}

func conditionalPrintlnCustomWriter(fn func(io.Writer, ...interface{}), w io.Writer, a ...interface{}) {
	if DisableOutput {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn(w, a...)
}

var (
	createPrintf = color.New(color.FgGreen).PrintfFunc()
	deletePrintf = color.New(color.FgRed).PrintfFunc()
	updatePrintf = color.New(color.FgYellow).PrintfFunc()

	// CreatePrintf is fmt.Printf with red as foreground color.
	CreatePrintf = func(format string, a ...interface{}) {
		conditionalPrintf(createPrintf, format, a...)
	}

	// DeletePrintf is fmt.Printf with green as foreground color.
	DeletePrintf = func(format string, a ...interface{}) {
		conditionalPrintf(deletePrintf, format, a...)
	}

	// UpdatePrintf is fmt.Printf with yellow as foreground color.
	UpdatePrintf = func(format string, a ...interface{}) {
		conditionalPrintf(updatePrintf, format, a...)
	}

	createPrintln  = color.New(color.FgGreen).PrintlnFunc()
	deletePrintln  = color.New(color.FgRed).PrintlnFunc()
	updatePrintln  = color.New(color.FgYellow).PrintlnFunc()
	bluePrintln    = color.New(color.BgBlue).PrintlnFunc()
	updateFprintln = color.New(color.FgYellow).FprintlnFunc()

	// CreatePrintln is fmt.Println with red as foreground color.
	CreatePrintln = func(a ...interface{}) {
		conditionalPrintln(createPrintln, a...)
	}

	// DeletePrintln is fmt.Println with green as foreground color.
	DeletePrintln = func(a ...interface{}) {
		conditionalPrintln(deletePrintln, a...)
	}

	// UpdatePrintln is fmt.Println with yellow as foreground color.
	UpdatePrintln = func(a ...interface{}) {
		conditionalPrintln(updatePrintln, a...)
	}

	BluePrintLn = func(a ...interface{}) {
		conditionalPrintln(bluePrintln, a...)
	}

	// UpdatePrintlnStdErr is fmt.Println with yellow as foreground color.
	// It prints to stderr, instead of stdout
	UpdatePrintlnStdErr = func(a ...interface{}) {
		conditionalPrintlnCustomWriter(updateFprintln, os.Stderr, a...)
	}
)
