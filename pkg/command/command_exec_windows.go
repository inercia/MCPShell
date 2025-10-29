//go:build windows
// +build windows

// Package command provides functions for creating and executing command handlers.
package command

// hasUnixTimeoutCommand returns whether the system has a Unix-style timeout command
func hasUnixTimeoutCommand() bool {
	// On Windows, we don't use Unix-style timeout command even if a 'timeout' command exists
	// because Windows 'timeout' is for pausing, not for limiting execution time
	return false
}