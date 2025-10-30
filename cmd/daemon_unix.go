//go:build !windows
// +build !windows

package root

import (
	"fmt"
	"os"
	"syscall"
)

// daemonize forks the process to run in the background
func daemonize() error {
	cmd, err := prepareDaemonCommand()
	if err != nil {
		return err
	}

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
