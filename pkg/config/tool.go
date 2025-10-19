package config

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/inercia/MCPShell/pkg/common"
)

// Tool holds an MCP tool and its associated handling information.
type Tool struct {
	// MCPTool is the MCP client-facing tool definition
	MCPTool mcp.Tool

	// Config is the original tool configuration
	Config MCPToolConfig

	// SelectedRunner is the runner that will be used to execute the tool command
	// This is set during validation when a suitable runner is found
	SelectedRunner *MCPToolRunner
}

// CheckToolRequirements checks if the tool has at least one runner that meets
// its prerequisites.
//
// Returns:
//   - true if a suitable runner is found, false otherwise
func (t *Tool) CheckToolRequirements() bool {
	// With the removal of deprecated fields, we now only support
	// the runners mechanism
	return t.findSuitableRunner()
}

// findSuitableRunner checks all defined runners and selects the first one
// that meets its requirements.
//
// Returns:
//   - true if a suitable runner was found, false otherwise
func (t *Tool) findSuitableRunner() bool {
	// If no runners are defined, use a default "exec" runner with no requirements
	if len(t.Config.Run.Runners) == 0 {
		defaultRunner := MCPToolRunner{
			Name: "exec",
		}
		t.SelectedRunner = &defaultRunner
		return true
	}

	// Check each defined runner
	for i, runner := range t.Config.Run.Runners {
		// Skip runners with invalid or empty names
		if runner.Name == "" {
			continue
		}

		// Check if OS matches (if specified)
		if runner.Requirements.OS != "" && !common.CheckOSMatches(runner.Requirements.OS) {
			continue
		}

		// Check if all required executables exist
		allExecutablesExist := true
		for _, execName := range runner.Requirements.Executables {
			if !common.CheckExecutableExists(execName) {
				allExecutablesExist = false
				break
			}
		}

		if !allExecutablesExist {
			continue
		}

		// Found a valid runner - store a reference to it
		t.SelectedRunner = &t.Config.Run.Runners[i]
		return true
	}

	// No suitable runner found
	return false
}

// GetEffectiveCommand returns the command template that should be used.
// Since the command is now always defined at the MCPToolRunConfig level,
// we simply return it directly.
func (t *Tool) GetEffectiveCommand() string {
	return t.Config.Run.Command
}

// GetEffectiveRunner returns the runner type that should be used.
func (t *Tool) GetEffectiveRunner() string {
	// Return the selected runner's name if we have one
	if t.SelectedRunner != nil && t.SelectedRunner.Name != "" {
		return t.SelectedRunner.Name
	}

	// Default to "exec" if no runner is selected
	return "exec"
}

// GetEffectiveOptions returns the runner options from the selected runner.
func (t *Tool) GetEffectiveOptions() map[string]interface{} {
	// Return the selected runner's options if we have them
	if t.SelectedRunner != nil && t.SelectedRunner.Options != nil {
		return t.SelectedRunner.Options
	}

	// Default to empty options if no runner is selected
	return nil
}

// CreateMCPTool creates an MCP tool from a tool configuration.
//
// Parameters:
//   - config: The tool configuration from which to create the MCP tool
//
// Returns:
//   - An mcp.Tool object ready to be registered with the MCP server
func CreateMCPTool(config MCPToolConfig) mcp.Tool {
	var options []mcp.ToolOption

	// Add description
	options = append(options, mcp.WithDescription(config.Description))

	// Add parameters
	for name, param := range config.Params {
		// If type is not specified, default to "string"
		paramType := param.Type
		if paramType == "" {
			paramType = "string"
		}

		// Create options for the parameter
		var paramOptions []mcp.PropertyOption

		// Add description
		paramOptions = append(paramOptions, mcp.Description(param.Description))

		// Add required option if needed
		if param.Required {
			paramOptions = append(paramOptions, mcp.Required())
		}

		// Add default value if specified
		if param.Default != nil {
			switch paramType {
			case "string":
				if strVal, ok := param.Default.(string); ok {
					paramOptions = append(paramOptions, mcp.DefaultString(strVal))
				}
			case "number", "integer":
				if numVal, ok := param.Default.(float64); ok {
					paramOptions = append(paramOptions, mcp.DefaultNumber(numVal))
				} else if intVal, ok := param.Default.(int64); ok {
					paramOptions = append(paramOptions, mcp.DefaultNumber(float64(intVal)))
				} else if intVal, ok := param.Default.(int); ok {
					paramOptions = append(paramOptions, mcp.DefaultNumber(float64(intVal)))
				}
			case "boolean":
				if boolVal, ok := param.Default.(bool); ok {
					paramOptions = append(paramOptions, mcp.DefaultBool(boolVal))
				}
			}
		}

		// Create parameter with the appropriate type
		switch paramType {
		case "string":
			options = append(options, mcp.WithString(name, paramOptions...))
		case "number", "integer":
			options = append(options, mcp.WithNumber(name, paramOptions...))
		case "boolean":
			options = append(options, mcp.WithBoolean(name, paramOptions...))
		}
	}

	return mcp.NewTool(config.Name, options...)
}
