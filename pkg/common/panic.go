// Package common provides shared utilities and types used across the MCPShell.
package common

import (
	"fmt"
	"os"
	"runtime/debug"
)

// RecoverPanic recovers from a panic and logs it to the provided logger.
// It returns true if a panic was recovered, false otherwise.
//
// This function should be used in deferred calls to catch panics.
func RecoverPanic() bool {
	logger := GetLogger()

	if r := recover(); r != nil {
		stackTrace := debug.Stack()

		// Log panic information to the logger if provided
		if logger != nil {
			logger.Debug("PANIC RECOVERED: %v", r)
			logger.Debug("Stack trace:\n%s", stackTrace)
		}

		// Always log to stderr for immediate visibility
		fmt.Fprintf(os.Stderr, "PANIC RECOVERED: %v\n", r)

		return true
	}

	return false
}
