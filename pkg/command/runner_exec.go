package command

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

	// Check if we should use the direct approach for Windows cmd regardless of isSingleExecutableCommand
	// This helps avoid the temporary script file issue on Windows where cmd shows version info
	configShell := getShell(shell)
	shellLower := strings.ToLower(configShell)
	
	// For Windows shells, use direct execution with appropriate parameter for better output capture
	if runtime.GOOS == "windows" && 
	   (strings.Contains(shellLower, "cmd") || strings.HasSuffix(shellLower, "cmd.exe") ||
	    strings.Contains(shellLower, "powershell") || strings.HasSuffix(shellLower, "powershell.exe") || 
	    strings.HasSuffix(shellLower, "pwsh.exe")) {
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

		// Format the command with proper shell syntax and file extension based on shell and OS
		var scriptContent strings.Builder
		var scriptFileName string
		
		shellLower := strings.ToLower(configShell)
		if runtime.GOOS == "windows" {
			// On Windows, format script content based on shell type
			if strings.Contains(shellLower, "cmd") || strings.HasSuffix(shellLower, "cmd.exe") {
				// For cmd shell, create a batch script that only outputs command result
				scriptContent.WriteString("@echo off\r\n")
				scriptContent.WriteString("chcp 65001 >nul 2>&1\r\n")  // Set UTF-8 encoding to handle international characters
				scriptContent.WriteString("setlocal\r\n")  // Start local environment
				scriptContent.WriteString(command)
				scriptContent.WriteString("\r\nendlocal\r\n")  // End local environment
				scriptContent.WriteString("exit /b %errorlevel%\r\n")
				scriptFileName = "script.bat"
			} else if strings.Contains(shellLower, "powershell") || strings.HasSuffix(shellLower, "powershell.exe") || strings.HasSuffix(shellLower, "pwsh.exe") {
				// For PowerShell, create a PowerShell script
				scriptContent.WriteString(command)
				scriptContent.WriteString("\nexit $LASTEXITCODE")
				scriptFileName = "script.ps1"
			} else {
				// Fallback to Unix-style for other shells
				scriptContent.WriteString("#!/bin/sh\n")
				scriptContent.WriteString(command)
				scriptFileName = "script.sh"
			}
		} else {
			// On Unix-like systems, use Unix-style script
			scriptContent.WriteString("#!/bin/sh\n")
			scriptContent.WriteString(command)
			scriptFileName = "script.sh"
		}

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

// getShell returns the shell to use for command execution,
// using the provided shell, falling back to $SHELL env var,
// and finally using appropriate default based on OS.
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

	// On Windows, default to cmd.exe if SHELL is not set
	if runtime.GOOS == "windows" {
		shell := os.Getenv("COMSPEC") // More reliable on Windows
		if shell != "" {
			return shell
		}
		return "cmd.exe" // Fallback for Windows
	}

	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}

	return "/bin/sh" // Default for Unix-like systems
}



// CheckImplicitRequirements checks if the runner meets its implicit requirements
// Exec runner has no special requirements
func (r *RunnerExec) CheckImplicitRequirements() error {
	// No special requirements for the basic exec runner
	return nil
}
