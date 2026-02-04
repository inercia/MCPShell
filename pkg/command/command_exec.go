// Package command provides functions for creating and executing command handlers.
package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/inercia/MCPShell/pkg/common"
	runnercommon "github.com/inercia/go-restricted-runner/pkg/common"
	"github.com/inercia/go-restricted-runner/pkg/runner"
)

// executeToolCommand handles the core logic of executing a command with the given parameters.
// This is a common implementation used by both direct execution and MCP handler.
//
// Parameters:
//   - ctx: Context for command execution
//   - params: Map of parameter names to their values
//
// Returns:
//   - The command output as a string
//   - A slice of failed constraint messages
//   - An error if command execution fails
//
// Security note: Runner options are only taken from the server-side tool configuration.
// External callers (MCP clients, CLI users) cannot override runner options to prevent
// privilege escalation attacks (e.g., specifying a different Docker image or user).
func (h *CommandHandler) executeToolCommand(ctx context.Context, params map[string]interface{}) (string, []string, error) {
	// Log the tool execution
	h.logger.Debug("Tool execution requested for '%s'", h.toolName)
	h.logger.Debug("Arguments: %v", params)

	// Apply default values for parameters that aren't provided but have defaults
	for paramName, paramConfig := range h.params {
		if _, exists := params[paramName]; !exists && paramConfig.Default != nil {
			h.logger.Debug("Using default value for parameter '%s': %v", paramName, paramConfig.Default)
			params[paramName] = paramConfig.Default
		}
	}

	// Check for required parameters that weren't provided and don't have defaults
	for paramName, paramConfig := range h.params {
		if paramConfig.Required {
			if _, exists := params[paramName]; !exists {
				h.logger.Error("Required parameter missing: %s", paramName)
				return "", nil, fmt.Errorf("required parameter missing: %s", paramName)
			}
		}
	}

	// Validate constraints before executing command
	var failedConstraints []string
	if h.constraintsCompiled != nil {
		h.logger.Debug("Checking %d constraints", len(h.constraints))
		satisfied, failed, err := h.constraintsCompiled.Evaluate(params, h.params)
		if err != nil {
			h.logger.Error("Error evaluating constraints: %v", err)
			return "", nil, fmt.Errorf("error evaluating constraints: %v", err)
		}
		if !satisfied {
			h.logger.Info("Constraints not satisfied, blocking execution")
			failedConstraints = failed
			errorMsg := "command execution blocked by constraints"

			// Add details about which constraints failed
			if len(failedConstraints) > 0 {
				errorMsg += ":\n"
				for i, fc := range failedConstraints {
					errorMsg += fmt.Sprintf("- Constraint %d: %s", i+1, fc)
					if i < len(failedConstraints)-1 {
						errorMsg += "\n"
					}
				}
			}

			return "", failedConstraints, fmt.Errorf("%s", errorMsg)
		}
		h.logger.Debug("All constraints satisfied")
	}

	// Process the command template with the tool arguments
	// h.logger.Debug("Processing command template:\n%s", h.cmd)

	cmd, err := common.ProcessTemplate(h.cmd, params)
	if err != nil {
		h.logger.Error("Error processing command template: %v", err)
		return "", nil, fmt.Errorf("error processing command template: %v", err)
	}

	// Wrap command with timeout if configured and timeout command is available
	if h.timeout != "" {
		timeoutDuration, err := time.ParseDuration(h.timeout)
		if err != nil {
			h.logger.Error("Invalid timeout format '%s': %v", h.timeout, err)
			return "", nil, fmt.Errorf("invalid timeout format '%s': %v", h.timeout, err)
		}

		// Convert to seconds for the timeout command
		timeoutSeconds := int(timeoutDuration.Seconds())
		if timeoutSeconds < 1 {
			timeoutSeconds = 1 // Minimum 1 second
		}

		// Escape single quotes in the command for shell
		escapedCmd := strings.ReplaceAll(cmd, "'", "'\"'\"'")

		// On Unix systems, try to use the 'timeout' command if available, otherwise use context-based timeout
		// On Windows, always use context-based timeout as 'timeout' command doesn't limit execution time
		if runner.ShouldUseUnixTimeoutCommand() {
			// On Unix/Linux/macOS systems, use timeout command with Unix syntax
			cmd = fmt.Sprintf("timeout --kill-after=5s %ds sh -c '%s'", timeoutSeconds, escapedCmd)
			h.logger.Debug("Wrapped command with Unix timeout: %ds", timeoutSeconds)
		} else {
			// timeout command not available on this platform or this is Windows
			// Fall back to context-based timeout (less reliable for child processes)
			h.logger.Debug("Timeout command not available, using context-based timeout: %s", h.timeout)
		}
	}

	// h.logger.Debug("Processed command: %s", cmd)

	// Prepare environment variables
	env := h.getEnvironmentVariables(params)

	h.logger.Debug("Executing command:")
	h.logger.Debug("\n------------------------------------------------------\n%s\n------------------------------------------------------\n", cmd)

	// Determine which runner to use based on the configuration
	runnerType := runner.TypeExec // default runner
	if h.runnerType != "" {
		h.logger.Debug("Using configured runner type: %s", h.runnerType)
		switch h.runnerType {
		case string(runner.TypeExec):
			runnerType = runner.TypeExec
		case string(runner.TypeSandboxExec):
			runnerType = runner.TypeSandboxExec
		case string(runner.TypeFirejail):
			runnerType = runner.TypeFirejail
		case string(runner.TypeDocker):
			runnerType = runner.TypeDocker
		default:
			h.logger.Error("Unknown runner type '%s', falling back to default runner", h.runnerType)
		}
	}

	// Use the configured runner options from the tool definition only
	// (external callers cannot override these for security reasons)
	runnerOptions := runner.Options{}
	for k, v := range h.runnerOpts {
		runnerOptions[k] = v
	}

	// Create the appropriate runner with options
	h.logger.Debug("Creating runner of type %s and checking implicit requirements", runnerType)

	// Create a runner-compatible logger
	runnerLogger, err := runnercommon.NewLogger("", "", runnercommon.LogLevel(h.logger.Level()), false)
	if err != nil {
		h.logger.Error("Error creating runner logger: %v", err)
		return "", nil, fmt.Errorf("error creating runner logger: %v", err)
	}

	r, err := runner.New(runnerType, runnerOptions, runnerLogger)
	if err != nil {
		h.logger.Error("Error creating runner: %v", err)
		return "", nil, fmt.Errorf("error creating runner: %v", err)
	}

	// Execute the command (timeout is handled by the context passed in from caller)
	commandOutput, err := r.Run(ctx, h.shell, cmd, env, params, true)
	if err != nil {
		h.logger.Error("Error executing command: %v", err)
		return "", nil, err
	}

	// Process the output
	finalOutput := commandOutput

	// Apply prefix if provided
	if h.output.Prefix != "" {
		h.logger.Debug("Applying output prefix template: %s", h.output.Prefix)

		// Process the prefix template with the tool arguments
		prefix, err := common.ProcessTemplate(h.output.Prefix, params)
		if err != nil {
			h.logger.Error("Error processing output prefix template: %v", err)
			return "", nil, fmt.Errorf("error processing output prefix template: %v", err)
		}

		// Combine prefix and command output
		finalOutput = strings.TrimSpace(prefix) + "\n\n" + finalOutput
		h.logger.Debug("Final output with prefix:\n--------------------------------\n%s\n--------------------------------", finalOutput)
	}

	h.logger.Debug("Tool execution completed successfully")
	return finalOutput, nil, nil
}

// ExecuteCommand handles the direct execution of a command without going through the MCP server.
// This is used by the "exe" command to execute a tool directly from the command line.
//
// Parameters:
//   - params: Map of parameter names to their values
//
// Returns:
//   - The command output as a string
//   - An error if command execution fails
func (h *CommandHandler) ExecuteCommand(params map[string]interface{}) (string, error) {
	// NOTE: Runner options are NOT extracted from params for security reasons.
	// Runner options must be defined in the tool configuration only.
	// This prevents users from overriding security-sensitive settings like
	// Docker image, user, or network configuration through command-line parameters.

	// Create context with timeout for command execution
	// Use configured timeout if available, otherwise use a default of 60 seconds
	timeout := 60 * time.Second
	if h.timeout != "" {
		parsedTimeout, err := time.ParseDuration(h.timeout)
		if err != nil {
			return "", fmt.Errorf("invalid timeout format '%s': %v", h.timeout, err)
		}
		timeout = parsedTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use the common implementation
	output, failedConstraints, err := h.executeToolCommand(ctx, params)

	// If constraints failed, format the error message
	if err != nil && len(failedConstraints) > 0 {
		return "", err
	}

	return output, err
}
