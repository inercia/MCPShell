package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/inercia/MCPShell/pkg/common"
)

// Exec implements the Runner interface for direct command execution
type Exec struct {
	logger  *common.Logger
	options ExecOptions
}

// ExecOptions is the options for the Exec runner
type ExecOptions struct {
	Shell string `json:"shell"`
}

// NewExecOptions creates a new ExecOptions from Options
func NewExecOptions(options Options) (ExecOptions, error) {
	var opts ExecOptions
	jsonStr, err := options.ToJSON()
	if err != nil {
		return ExecOptions{}, err
	}
	err = json.Unmarshal([]byte(jsonStr), &opts)
	return opts, err
}

// NewExec creates a new Exec runner with the provided logger.
// If logger is nil, a default logger is created.
func NewExec(options Options, logger *common.Logger) (*Exec, error) {
	if logger == nil {
		logger = common.GetLogger()
	}

	execOptions, err := NewExecOptions(options)
	if err != nil {
		return nil, err
	}

	return &Exec{
		logger:  logger,
		options: execOptions,
	}, nil
}

// Run executes a command with the given shell and returns the output.
// It implements the Runner interface.
//
// Note: For Windows native shells (cmd, powershell), the 'tmpfile' parameter is ignored
// and commands are executed directly to avoid issues with output capturing.
func (r *Exec) Run(ctx context.Context, shell string,
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

	// Check if we should use the direct approach for Windows cmd regardless of isSingleExecutableCommand
	// This helps avoid the temporary script file issue on Windows where cmd shows version info
	configShell := getShell(shell)
	shellLower := strings.ToLower(configShell)

	// For Windows shells, use direct execution with appropriate parameter for better output capture
	if runtime.GOOS == "windows" && isWindowsShell(shellLower) {
		// Use direct execution for Windows shells to avoid temp file issues
		shellPath, args := getShellCommandArgs(configShell, command)
		execCmd = exec.CommandContext(ctx, shellPath, args...)
		r.logger.Debug("Created direct command for Windows: %s with args %v", shellPath, args)
	} else if isSingleExecutableCommand(command) {
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

		// Format the command with proper shell syntax and file extension
		var scriptContent strings.Builder

		// On Unix-like systems, use Unix-style script
		scriptContent.WriteString("#!/bin/sh\n")
		scriptContent.WriteString(command)
		scriptFileName := "script.sh"

		tmpFile := filepath.Join(tmpDir, scriptFileName)
		err = os.WriteFile(tmpFile, []byte(scriptContent.String()), 0o700)
		if err != nil {
			r.logger.Debug("Failed to write temporary file: %v", err)
			return "", err
		}

		r.logger.Debug("Created temporary script file at: %s", tmpFile)

		// Set up the command
		r.logger.Debug("Using shell: %s", configShell)

		// Create the command to execute the script file
		execCmd = exec.CommandContext(ctx, configShell, tmpFile)
		r.logger.Debug("Created command: %s %s", configShell, tmpFile)
	} else {
		// Execute the command directly without a temporary file (Unix-style)
		r.logger.Debug("Using shell: %s", configShell)

		// Get the appropriate command arguments for this shell
		shellPath, args := getShellCommandArgs(configShell, command)
		execCmd = exec.CommandContext(ctx, shellPath, args...)
		r.logger.Debug("Created command: %s with args %v", shellPath, args)
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

	// Get the combined output in case stdout doesn't capture everything
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// For Windows, we might need to handle output differently
	// Some Windows commands output to stderr instead of stdout
	output := stdoutStr
	if runtime.GOOS == "windows" && strings.TrimSpace(stdoutStr) == "" && strings.TrimSpace(stderrStr) != "" {
		// If stdout is empty but stderr has content, use stderr
		output = stderrStr
	} else if runtime.GOOS == "windows" && strings.Contains(output, "Microsoft Windows [版本") {
		// If the output contains Windows version info, the command might not have executed properly
		// This indicates the batch file might not have been set up properly to capture command output
		r.logger.Debug("Detected Windows command prompt output, checking for real command output")
		// We'll still return what we captured, but this suggests the command didn't execute as expected
	}

	// Trim the output but preserve meaningful content
	output = strings.TrimSpace(output)

	r.logger.Debug("Command executed successfully, output length: %d bytes", len(output))
	if stderr.Len() > 0 {
		r.logger.Debug("Command generated stderr (but no error): '%s'", strings.TrimSpace(stderrStr))
	}
	r.logger.Debug("Full output captured: '%s'", output)

	// Return the output
	return output, nil
}

// CheckImplicitRequirements checks if the runner meets its implicit requirements.
// Exec runner has no special requirements.
func (r *Exec) CheckImplicitRequirements() error {
	// No special requirements for the basic exec runner
	return nil
}
