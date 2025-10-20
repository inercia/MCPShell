package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inercia/MCPShell/pkg/common"
)

// RunnerExec implements the Runner interface
type RunnerExec struct {
	logger  *common.Logger
	options RunnerExecOptions
}

// RunnerExecOptions is the options for the RunnerExec
type RunnerExecOptions struct {
	Shell string `json:"shell"`
}

// NewRunnerExecOptions creates a new RunnerExecOptions from a RunnerOptions
func NewRunnerExecOptions(options RunnerOptions) (RunnerExecOptions, error) {
	var reopts RunnerExecOptions
	opts, err := options.ToJSON()
	if err != nil {
		return RunnerExecOptions{}, err
	}
	err = json.Unmarshal([]byte(opts), &reopts)
	return reopts, err
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// NewRunnerExec creates a new ExecRunner with the provided logger
// If logger is nil, a default logger is created
func NewRunnerExec(options RunnerOptions, logger *common.Logger) (*RunnerExec, error) {
	if logger == nil {
		logger = common.GetLogger()
	}

	execOptions, err := NewRunnerExecOptions(options)
	if err != nil {
		return nil, err
	}

	return &RunnerExec{
		logger:  logger,
		options: execOptions,
	}, nil
}

// Run executes a command with the given shell and returns the output
// It implements the Runner interface
func (r *RunnerExec) Run(ctx context.Context, shell string,
	command string,
	env []string, params map[string]interface{},
	tmpfile bool,
) (string, error) {
	// Check if context is done
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue execution
	}

	var execCmd *exec.Cmd
	var tmpDir string

	if isSingleExecutableCommand(command) {
		r.logger.Debug("Optimization: running single executable command directly: %s", command)
		execCmd = exec.CommandContext(ctx, command)
		if len(env) > 0 {
			r.logger.Debug("Adding %d environment variables to command", len(env))
			for _, e := range env {
				r.logger.Debug("... adding environment variable: %s", e)
			}
			execCmd.Env = append(os.Environ(), env...)
		}
		r.logger.Debug("Created command: %s", command)
	} else if tmpfile {
		// Create a temporary file for the command
		var err error
		tmpDir, err = os.MkdirTemp("", "mcpshell")
		if err != nil {
			r.logger.Debug("Failed to create temp directory: %v", err)
			return "", err
		}
		defer func() {
			if err := os.RemoveAll(tmpDir); err != nil {
				r.logger.Debug("Failed to remove temporary directory: %v", err)
			}
		}()

		// Format the command with proper shell syntax
		var scriptContent strings.Builder
		scriptContent.WriteString("#!/bin/sh\n")
		scriptContent.WriteString(command)

		tmpFile := filepath.Join(tmpDir, "script.sh")
		err = os.WriteFile(tmpFile, []byte(scriptContent.String()), 0o700)
		if err != nil {
			r.logger.Debug("Failed to write temporary file: %v", err)
			return "", err
		}

		r.logger.Debug("Created temporary script file at: %s", tmpFile)

		// Set up the command
		configShell := getShell(shell)
		r.logger.Debug("Using shell: %s", configShell)

		// Create the command to execute the script file
		execCmd = exec.CommandContext(ctx, configShell, tmpFile)
		r.logger.Debug("Created command: %s %s", configShell, tmpFile)
	} else {
		// Execute the command directly without a temporary file
		configShell := getShell(shell)
		r.logger.Debug("Using shell: %s", configShell)

		// Simple command without arguments
		execCmd = exec.CommandContext(ctx, configShell, "-c", command)
		r.logger.Debug("Created command: %s -c %s", configShell, command)
	}

	// Set environment variables if provided
	if len(env) > 0 {
		r.logger.Debug("Adding %d environment variables to command", len(env))
		for _, e := range env {
			r.logger.Debug("... adding environment variable: %s", e)
		}
		execCmd.Env = append(os.Environ(), env...)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Run the command
	r.logger.Debug("Executing command")

	err := execCmd.Run()
	if err != nil {
		// If there's error output, include it in the error
		if stderr.Len() > 0 {
			errMsg := strings.TrimSpace(stderr.String())
			r.logger.Debug("Command failed with stderr: %s", errMsg)
			return "", errors.New(errMsg)
		}
		r.logger.Debug("Command failed with error: %v", err)
		return "", err
	}

	// Get the output
	output := strings.TrimSpace(stdout.String())

	r.logger.Debug("Command executed successfully, output length: %d bytes", len(output))
	if stderr.Len() > 0 {
		r.logger.Debug("Command generated stderr (but no error): %s", strings.TrimSpace(stderr.String()))
	}

	// Return the stdout output
	return output, nil
}

// getShell returns the shell to use for command execution,
// using the provided shell, falling back to $SHELL env var,
// and finally using /bin/sh as a last resort.
//
// Parameters:
//   - configShell: The configured shell to use (can be empty)
//
// Returns:
//   - The shell executable path to use
func getShell(configShell string) string {
	if configShell != "" {
		return configShell
	}

	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}

	return "/bin/sh"
}

// CheckImplicitRequirements checks if the runner meets its implicit requirements
// Exec runner has no special requirements
func (r *RunnerExec) CheckImplicitRequirements() error {
	// No special requirements for the basic exec runner
	return nil
}
