//go:build !windows
// +build !windows

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
	for i, arg := range args {
		if arg == "--daemon" {
			// Skip the daemon flag
			continue
		}
		if i > 0 && args[i-1] == "--daemon" {
			// Skip the value if it was a separate argument
			continue
		}
		newArgs = append(newArgs, arg)
	}

	// Create the command
	cmd := exec.Command(executable, newArgs...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	// Set up process attributes for daemon behavior
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon process: %w", err)
	}

	// Exit the parent process
	os.Exit(0)

	// This line should never be reached, but Go requires it
	return nil
}
