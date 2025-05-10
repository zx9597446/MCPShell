// Package common provides shared utilities and types used across the MCP CLI Adapter.
package common

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
)

// RecoverPanic recovers from a panic and logs it to the provided logger.
// It returns true if a panic was recovered, false otherwise.
//
// Parameters:
//   - logger: The logger to use for logging the panic. If nil, logs only to stderr.
//   - logFile: The path to the log file. Only used for informational messages to stderr.
//
// This function should be used in deferred calls to catch panics.
func RecoverPanic(logger *log.Logger, logFile string) bool {
	if r := recover(); r != nil {
		stackTrace := debug.Stack()

		// Log panic information to the logger if provided
		if logger != nil {
			logger.Printf("PANIC RECOVERED: %v", r)
			logger.Printf("Stack trace:\n%s", stackTrace)
		}

		// Always log to stderr for immediate visibility
		fmt.Fprintf(os.Stderr, "PANIC RECOVERED: %v\n", r)
		if logFile != "" {
			fmt.Fprintf(os.Stderr, "Stack trace has been written to the log file: %s\n", logFile)
		} else {
			fmt.Fprintf(os.Stderr, "Stack trace:\n%s\n", stackTrace)
		}

		return true
	}

	return false
}
