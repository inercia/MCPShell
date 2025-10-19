package root

import (
	"fmt"
	"strings"

	"github.com/inercia/MCPShell/pkg/command"
	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
	"github.com/spf13/cobra"
)

// exeCommand is a command that executes a MCP tool
var exeCommand = &cobra.Command{
	Use:   "exe",
	Short: "Execute a MCP tool",
	Long: `
Direct execution of a MCP tool.

This command will just execute a MCP tool, with the given parameters.
Sometimes it is difficult to debug the execution of a MCP tool.
This command will help you to debug the tool by executing it with
the given parameters, following the whole process of constraint
evaluation, tool selection and tool execution.

For example, you can run:

$ mcpshell exe --tools examples/config.yaml "hello_world" "name=John"

and it will run the "hello_world" tool with the parameter "name" set to "John".
You can also specify multiple tools files:

$ mcpshell exe --tools file1.yaml --tools file2.yaml "hello_world" "name=John"
$ mcpshell exe --tools file1.yaml,file2.yaml "hello_world" "name=John"

Any error in the constraint evaluation, tool selection or tool execution
will be reported.

`,
	Args: cobra.MinimumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Setup panic handler
		defer common.RecoverPanic()

		logger.Info("Executing MCP tool directly")

		// Check if config file is provided
		if len(toolsFiles) == 0 {
			logger.Error("Tools configuration file(s) are required")
			return fmt.Errorf("tools configuration file(s) are required. Use --tools flag to specify the path(s)")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the logger
		logger := common.GetLogger()

		// Setup panic handler
		defer common.RecoverPanic()

		// Get the tool name
		toolName := args[0]
		logger.Info("Executing tool: %s", toolName)

		// Load the configuration file(s) (local or remote)
		localConfigPath, cleanup, err := config.ResolveMultipleConfigPaths(toolsFiles, logger)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Ensure temporary files are cleaned up
		defer cleanup()

		// Load the configuration
		cfg, err := config.NewConfigFromFile(localConfigPath)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Find the requested tool in the configuration
		var targetTool *config.MCPToolConfig
		for _, toolConfig := range cfg.MCP.Tools {
			if toolConfig.Name == toolName {
				targetTool = &toolConfig
				break
			}
		}

		if targetTool == nil {
			logger.Error("Tool not found: %s", toolName)
			return fmt.Errorf("tool not found: %s", toolName)
		}

		// Parse parameters from the remaining arguments
		params := make(map[string]interface{})
		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) != 2 {
				logger.Error("Invalid parameter format: %s (expected name=value)", arg)
				return fmt.Errorf("invalid parameter format: %s (expected name=value)", arg)
			}
			paramName := parts[0]
			paramValue := parts[1]

			// Check if parameter is defined in the tool
			paramConfig, exists := targetTool.Params[paramName]
			if !exists {
				logger.Error("Parameter not defined in tool: %s", paramName)
				return fmt.Errorf("parameter not defined in tool: %s", paramName)
			}

			// Convert parameter value to appropriate type based on parameter config
			typedValue, err := common.ConvertStringToType(paramValue, paramConfig.Type)
			if err != nil {
				logger.Error("Failed to convert parameter value: %v", err)
				return fmt.Errorf("failed to convert parameter value: %w", err)
			}

			params[paramName] = typedValue
		}

		// Apply default values for parameters that aren't provided but have defaults
		for paramName, paramConfig := range targetTool.Params {
			if _, exists := params[paramName]; !exists && paramConfig.Default != nil {
				logger.Info("Using default value for parameter '%s': %v", paramName, paramConfig.Default)
				params[paramName] = paramConfig.Default
			}
		}

		// Check required parameters
		for paramName, paramConfig := range targetTool.Params {
			if paramConfig.Required {
				if _, exists := params[paramName]; !exists {
					logger.Error("Required parameter missing: %s", paramName)
					return fmt.Errorf("required parameter missing: %s", paramName)
				}
			}
		}

		// Use shell from config if present
		shell := cfg.MCP.Run.Shell
		if shell == "" {
			shell = "sh"
		}

		// Create a Tool and check requirements to select the appropriate runner
		tool := config.Tool{
			MCPTool: config.CreateMCPTool(*targetTool),
			Config:  *targetTool,
		}

		// Check tool requirements and select runner
		if !tool.CheckToolRequirements() {
			logger.Error("Tool requirements not met - no suitable runner found")
			return fmt.Errorf("tool requirements not met - no suitable runner found")
		}

		// Create a command handler
		handler, err := command.NewCommandHandler(tool, targetTool.Params, shell, logger)
		if err != nil {
			logger.Error("Failed to create command handler: %v", err)
			return fmt.Errorf("failed to create command handler: %w", err)
		}

		// Execute the command directly
		result, err := handler.ExecuteCommand(params)
		if err != nil {
			logger.Error("Command execution failed: %v", err)
			return fmt.Errorf("command execution failed: %w", err)
		}

		// Print the result
		fmt.Println(result)
		return nil
	},
}

// init adds the exe command to the root command
func init() {
	// Add exe command to root
	rootCmd.AddCommand(exeCommand)

	// Mark required flags
	_ = exeCommand.MarkFlagRequired("tools")
}
