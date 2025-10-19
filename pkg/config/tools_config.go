// Package config provides configuration loading and handling functionality.
//
// It defines the data structures for representing the application's configuration,
// which is loaded from YAML files, and includes utilities for parsing and
// processing these configurations.
package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/inercia/MCPShell/pkg/common"
)

// ToolsConfig represents the top-level configuration structure for the application.
type ToolsConfig struct {
	// Prompts is a prompt configuration that will be provided to clients
	Prompts common.PromptsConfig `yaml:"prompts,omitempty"`

	// MCP contains the configuration specific to the MCP server and tools
	MCP MCPConfig `yaml:"mcp"`
}

// MCPConfig represents the MCP server configuration section.
type MCPConfig struct {
	// Description is a text shown to AI clients that explains what this server does
	Description string `yaml:"description,omitempty"`

	// Run contains runtime configuration
	Run MCPRunConfig `yaml:"run,omitempty"`

	// Tools is a list of tool definitions that will be provided to clients
	Tools []MCPToolConfig `yaml:"tools"`
}

// MCPRunConfig represents run-specific configuration options.
type MCPRunConfig struct {
	// Shell is the shell to use for executing commands (e.g., bash, sh, zsh)
	Shell string `yaml:"shell,omitempty"`
}

// MCPToolConfig represents a single tool configuration.
type MCPToolConfig struct {
	// Name is the unique identifier for the tool
	Name string `yaml:"name"`

	// Description explains what the tool does (shown to AI clients)
	Description string `yaml:"description"`

	// Params defines the parameters that the tool accepts
	Params map[string]common.ParamConfig `yaml:"params"`

	// Constraints are expressions that limit when the tool can be executed
	Constraints []string `yaml:"constraints,omitempty"`

	// Run specifies how to execute the tool
	Run MCPToolRunConfig `yaml:"run"`

	// Output specifies how to format the tool's output
	Output common.OutputConfig `yaml:"output,omitempty"`
}

// MCPToolRequirements represents a prerequisite tool configuration.
// If these prerequisites are not met, the tool will not even be shown as
// available to the client.
// This allows for tools to be conditionally shown based on the user's system.
type MCPToolRequirements struct {
	// OS is the operating system that the prerequisite tool must be installed on
	OS string `yaml:"os,omitempty"`

	// Executables is a list of executable names that must be present in the system
	Executables []string `yaml:"executables"`
}

// MCPToolRunner represents a specific execution environment for a tool.
type MCPToolRunner struct {
	// Name is the identifier for this runner (e.g., "osx", "linux")
	Name string `yaml:"name"`

	// Requirements are the prerequisites for this runner to be used
	Requirements MCPToolRequirements `yaml:"requirements,omitempty"`

	// Options for the runner
	Options map[string]interface{} `yaml:"options,omitempty"`
}

// MCPToolRunConfig represents the run configuration for a tool.
type MCPToolRunConfig struct {
	// Command is a template for the shell command to execute
	Command string `yaml:"command"`

	// Env is a list of environment variable names to pass from the parent process
	Env []string `yaml:"env,omitempty"`

	// Timeout is the maximum duration for command execution (e.g., "30s", "5m")
	// If not specified, no timeout is applied
	Timeout string `yaml:"timeout,omitempty"`

	// Runners is a list of possible runner configurations
	Runners []MCPToolRunner `yaml:"runners,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////////

// NewConfigFromFile loads the configuration from a YAML file at the specified path.
// The file path should already be resolved (use ResolveConfigPath for URL/directory resolution).
//
// Parameters:
//   - filepath: Path to the YAML configuration file (should be absolute and resolved)
//
// Returns:
//   - A pointer to the loaded Config structure
//   - An error if loading or parsing fails
func NewConfigFromFile(filepath string) (*ToolsConfig, error) {
	// Open the configuration file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", filepath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Read the file content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filepath, err)
	}

	// Parse the YAML content
	var config ToolsConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filepath, err)
	}

	return &config, nil
}

// GetTools converts the configuration's tool definitions into a list of
// executable ToolDefinition objects ready to be registered with the MCP server.
//
// Returns:
//   - A slice of ToolDefinition objects
func (c *ToolsConfig) GetTools() []Tool {
	var tools []Tool

	for _, toolConfig := range c.MCP.Tools {
		tool := Tool{
			MCPTool: CreateMCPTool(toolConfig),
			Config:  toolConfig,
		}

		// Check prerequisites before creating the tool
		if !tool.CheckToolRequirements() {
			continue // Skip this tool if prerequisites are not met
		}

		tools = append(tools, tool)
	}

	return tools
}

// ToYAML serializes the configuration back to YAML format.
//
// Returns:
//   - YAML data as bytes
//   - An error if serialization fails
func (c *ToolsConfig) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}

// LoadAndMergeConfigs loads multiple configuration files and merges them into a single configuration.
// The merging strategy is:
// - Prompts are concatenated from all files
// - MCP description from the first file is used (others are ignored)
// - MCP run config from the first file is used (others are ignored)
// - Tools from all files are combined
//
// Parameters:
//   - filepaths: List of paths to YAML configuration files
//
// Returns:
//   - A pointer to the merged Config structure
//   - An error if loading or merging fails
func LoadAndMergeConfigs(filepaths []string) (*ToolsConfig, error) {
	if len(filepaths) == 0 {
		return nil, fmt.Errorf("no configuration files provided")
	}

	var mergedConfig ToolsConfig
	var isFirstFile = true

	for _, filepath := range filepaths {
		config, err := NewConfigFromFile(filepath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", filepath, err)
		}

		// Merge prompts (concatenate system and user prompts)
		mergedConfig.Prompts.System = append(mergedConfig.Prompts.System, config.Prompts.System...)
		mergedConfig.Prompts.User = append(mergedConfig.Prompts.User, config.Prompts.User...)

		// For MCP config, use the first file's description and run config
		if isFirstFile {
			mergedConfig.MCP.Description = config.MCP.Description
			mergedConfig.MCP.Run = config.MCP.Run
			isFirstFile = false
		}

		// Merge tools (combine from all files)
		mergedConfig.MCP.Tools = append(mergedConfig.MCP.Tools, config.MCP.Tools...)
	}

	return &mergedConfig, nil
}
