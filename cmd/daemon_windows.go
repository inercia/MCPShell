//go:build windows
// +build windows

package root

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// daemonize forks the process to run in the background
func daemonize() error {
	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Build command arguments, excluding the daemon flag
	args := os.Args[1:]
	var newArgs []string
	for _, arg := range args {
		if arg != "--daemon" {
			newArgs = append(newArgs, arg)
		}
	}

	// Create the command
	cmd := exec.Command(executable, newArgs...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	// Set up process attributes for daemon behavior
	// On Windows, we don't have Setsid.
	// An empty SysProcAttr is sufficient to make it compile.
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Exit the parent process
	os.Exit(0)

	// This line should never be reached, but Go requires it
	return nil
}
