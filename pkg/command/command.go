// Package command provides functions for creating and executing command handlers.
//
// This package defines the core functionality for translating MCP tool calls
// into shell command executions, providing the bridge between the MCP protocol
// and the underlying operating system commands.
package command

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
)

// isSingleExecutableCommand checks if the command string is a single word (no spaces or shell metacharacters)
// and if that word is an existing executable (absolute/relative path or in PATH).
func isSingleExecutableCommand(command string) bool {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return false
	}
	// Disallow spaces, shell metacharacters, and redirections
	if strings.ContainsAny(cmd, " \t|&;<>(){}[]$`'\"\n") {
		return false
	}
	// If it's an absolute or relative path
	if strings.HasPrefix(cmd, "/") || strings.HasPrefix(cmd, ".") {
		info, err := os.Stat(cmd)
		if err != nil {
			return false
		}
		mode := info.Mode()
		return !info.IsDir() && mode&0111 != 0 // executable by someone
	}
	// Otherwise, check if it's in PATH
	return common.CheckExecutableExists(cmd)
}

// CommandHandler encapsulates the configuration and behavior needed to handle tool commands.
type CommandHandler struct {
	cmd                 string                        // the command to execute
	output              common.OutputConfig           // the output configuration
	constraints         []string                      // the constraints to evaluate
	constraintsCompiled *common.CompiledConstraints   // ... and the compiled versions
	params              map[string]common.ParamConfig // the parameter configurations
	envVars             []string                      // the environment variables passed to the command
	timeout             string                        // the timeout for command execution (e.g., "30s", "5m")
	shell               string                        // the shell to use
	toolName            string                        // the name of the tool
	runnerType          string                        // the type of runner to use
	runnerOpts          RunnerOptions                 // the options for the runner

	logger *common.Logger
}

// NewCommandHandler creates a new CommandHandler instance.
//
// Parameters:
//   - tool: The tool definition containing command, constraints, and output configuration
//   - params: Map of parameter names to their type configurations
//   - shell: The shell to use for command execution
//   - logger: Logger for detailed execution information (required)
//
// Returns:
//   - A new CommandHandler instance and nil if successful
//   - nil and an error if constraint compilation fails or if a required parameter is missing
func NewCommandHandler(tool config.Tool, params map[string]common.ParamConfig, shell string, logger *common.Logger) (*CommandHandler, error) {
	// Check required parameters
	if logger == nil {
		return nil, fmt.Errorf("logger is required for CommandHandler")
	}

	// Log tool creation
	logger.Debug("Creating handler for tool '%s'", tool.MCPTool.Name)

	// Compile constraints during initialization
	var compiled *common.CompiledConstraints
	var err error

	if len(tool.Config.Constraints) > 0 {
		logger.Info("Compiling %d constraints for tool '%s'", len(tool.Config.Constraints), tool.MCPTool.Name)

		compiled, err = common.NewCompiledConstraints(tool.Config.Constraints, params, logger)
		if err != nil {
			logger.Error("Failed to compile constraints for tool %s: %v", tool.MCPTool.Name, err)
			return nil, fmt.Errorf("constraint compilation error: %w", err)
		}

		logger.Info("Successfully compiled constraints for tool '%s'", tool.MCPTool.Name)
	}

	// Get the effective command, runner type, and options from the tool
	effectiveCommand := tool.GetEffectiveCommand()
	effectiveRunnerType := tool.GetEffectiveRunner()
	effectiveOptions := tool.GetEffectiveOptions()

	logger.Debug("Using command: %s", effectiveCommand)
	logger.Debug("Using runner type: %s", effectiveRunnerType)

	// Convert the runner options to RunnerOptions
	runnerOpts := RunnerOptions{}
	if effectiveOptions != nil {
		for k, v := range effectiveOptions {
			runnerOpts[k] = v
		}
		logger.Debug("Runner options for tool '%s': %v", tool.MCPTool.Name, runnerOpts)
	}

	// Create and return the handler
	return &CommandHandler{
		cmd:                 effectiveCommand,
		output:              tool.Config.Output,
		constraints:         tool.Config.Constraints,
		params:              params,
		constraintsCompiled: compiled,
		envVars:             tool.Config.Run.Env,
		timeout:             tool.Config.Run.Timeout,
		shell:               shell,
		toolName:            tool.MCPTool.Name,
		runnerType:          effectiveRunnerType,
		runnerOpts:          runnerOpts,
		logger:              logger,
	}, nil
}

// GetMCPHandler returns a function that handles MCP tool calls by executing shell commands.
//
// This is the function that should be registered with the MCP server.
//
// Returns:
//   - A function that handles MCP tool calls
func (h *CommandHandler) GetMCPHandler() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Assert Arguments to a map
		var args map[string]interface{}
		if request.Params.Arguments != nil {
			var ok bool
			args, ok = request.Params.Arguments.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: expected a map, got %T", request.Params.Arguments)), nil
			}
		}

		// Extract runner options if present
		var runnerOpts map[string]interface{}
		if args != nil {
			if opts, ok := args["options"].(map[string]interface{}); ok {
				runnerOpts = opts
			}
		}

		// Apply timeout if configured
		executionCtx := ctx
		if h.timeout != "" {
			timeout, err := time.ParseDuration(h.timeout)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid timeout format '%s': %v", h.timeout, err)), nil
			}
			var cancel context.CancelFunc
			executionCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		// Execute the command using the common implementation
		output, _, err := h.executeToolCommand(executionCtx, args, runnerOpts)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(output), nil
	}
}

// getEnvironmentVariables gets the environment variables for the process.
//
// * for single env variables (ie, ENV_VAR), it obtains the value from the parent process
// * for assignments (ie, ENV_VAR=value), it uses the value directly
// * for templated assignments (ie, EBV_VAR={{ .param }}), it processes the template with the given params
//
// It returns all the env vars as a list of KEY=VALUE.
func (h *CommandHandler) getEnvironmentVariables(params map[string]interface{}) []string {
	if len(h.envVars) == 0 {
		return nil
	}

	envVars := make([]string, 0, len(h.envVars))
	for _, name := range h.envVars {
		comps := strings.Split(name, "=")
		if len(comps) == 1 {
			if value, exists := os.LookupEnv(name); exists {
				envVars = append(envVars, name+"="+value)
			} else {
				envVars = append(envVars, name+"=")
			}
		} else {
			p, err := common.ProcessTemplate(comps[1], params)
			if err != nil {
				envVars = append(envVars, name)
			} else {
				envVars = append(envVars, comps[0]+"="+p)
			}
		}
	}

	return envVars
}
