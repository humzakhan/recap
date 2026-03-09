// Package log provides a simple leveled logger for recap.
// When verbose mode is enabled, debug messages are printed to stderr.
// Errors and warnings are always printed.
package log

import (
	"fmt"
	"os"
	"time"
)

var verbose bool

// SetVerbose enables or disables verbose (debug) output.
func SetVerbose(v bool) {
	verbose = v
}

// Verbose returns whether verbose mode is enabled.
func Verbose() bool {
	return verbose
}

// Debug prints a message only when verbose mode is enabled.
func Debug(format string, args ...any) {
	if !verbose {
		return
	}
	ts := time.Now().Format("15:04:05.000")
	fmt.Fprintf(os.Stderr, "[DEBUG %s] %s\n", ts, fmt.Sprintf(format, args...))
}

// Warn prints a warning message to stderr.
func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[WARN] %s\n", fmt.Sprintf(format, args...))
}

// Error prints an error message to stderr.
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] %s\n", fmt.Sprintf(format, args...))
}
