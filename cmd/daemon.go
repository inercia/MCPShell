package root

import (
	"fmt"
	"os"
	"os/exec"
)

func prepareDaemonCommand() (*exec.Cmd, error) {
	// Get the current executable path
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}

	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
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

	return cmd, nil
}
