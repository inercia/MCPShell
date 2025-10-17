// Package command provides functions for creating and executing command handlers.
package command

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/inercia/MCPShell/pkg/common"
)

// executeToolCommand handles the core logic of executing a command with the given parameters.
// This is a common implementation used by both direct execution and MCP handler.
//
// Parameters:
//   - ctx: Context for command execution
//   - params: Map of parameter names to their values
//   - extraRunnerOpts: Additional runner options to apply
//
// Returns:
//   - The command output as a string
//   - A slice of failed constraint messages
//   - An error if command execution fails
func (h *CommandHandler) executeToolCommand(ctx context.Context, params map[string]interface{}, extraRunnerOpts map[string]interface{}) (string, []string, error) {
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

	// h.logger.Debug("Processed command: %s", cmd)

	// Prepare environment variables
	env := h.getEnvironmentVariables(params)

	h.logger.Debug("Executing command:")
	h.logger.Debug("\n------------------------------------------------------\n%s\n------------------------------------------------------\n", cmd)

	// Determine which runner to use based on the configuration
	runnerType := RunnerTypeExec // default runner
	if h.runnerType != "" {
		h.logger.Debug("Using configured runner type: %s", h.runnerType)
		switch h.runnerType {
		case string(RunnerTypeExec):
			runnerType = RunnerTypeExec
		case string(RunnerTypeSandboxExec):
			runnerType = RunnerTypeSandboxExec
		case string(RunnerTypeFirejail):
			runnerType = RunnerTypeFirejail
		default:
			h.logger.Error("Unknown runner type '%s', falling back to default runner", h.runnerType)
		}
	}

	// Start with the configured runner options from the tool definition
	runnerOptions := RunnerOptions{}
	for k, v := range h.runnerOpts {
		runnerOptions[k] = v
	}

	// Add or override with any options from the parameters if present
	if extraRunnerOpts != nil {
		h.logger.Debug("Found runner options in parameters: %v", extraRunnerOpts)
		for k, v := range extraRunnerOpts {
			runnerOptions[k] = v
		}
	}

	// Create the appropriate runner with options
	h.logger.Debug("Creating runner of type %s and checking implicit requirements", runnerType)
	runner, err := NewRunner(runnerType, runnerOptions, h.logger)
	if err != nil {
		h.logger.Error("Error creating runner: %v", err)
		return "", nil, fmt.Errorf("error creating runner: %v", err)
	}

	// Execute the command
	commandOutput, err := runner.Run(ctx, h.shell, cmd, env, params, true)
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

	h.logger.Info("Tool execution completed successfully")
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
	// Extract runner options if present
	var runnerOpts map[string]interface{}
	if opts, ok := params["options"].(map[string]interface{}); ok {
		runnerOpts = opts
		// Remove options from params to avoid processing them as command parameters
		tmpParams := make(map[string]interface{})
		for k, v := range params {
			if k != "options" {
				tmpParams[k] = v
			}
		}
		params = tmpParams
	}

	// Create context with timeout for command execution
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use the common implementation
	output, failedConstraints, err := h.executeToolCommand(ctx, params, runnerOpts)

	// If constraints failed, format the error message
	if err != nil && len(failedConstraints) > 0 {
		return "", err
	}

	return output, err
}
