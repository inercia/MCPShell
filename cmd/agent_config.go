package root

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/inercia/MCPShell/pkg/agent"
	"github.com/inercia/MCPShell/pkg/utils"
)

var (
	agentConfigShowJSON bool
)

// agentConfigCommand is the parent command for agent configuration subcommands
var agentConfigCommand = &cobra.Command{
	Use:   "config",
	Short: "Manage agent configuration",
	Long: `

The config command provides subcommands to manage agent configuration files.

Available subcommands:
- create: Create a default agent configuration file
- show: Display the current agent configuration
`,
}

// agentConfigCreateCommand creates a default agent configuration file
var agentConfigCreateCommand = &cobra.Command{
	Use:   "create",
	Short: "Create a default agent configuration file",
	Long: `

Creates a default agent configuration file at ~/.mcpshell/agent.yaml.

If the file already exists, it will be overwritten with the default configuration.
The default configuration includes sample models and prompts that you can customize.

Example:
$ mcpshell agent config create
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Create the default config file
		if err := agent.CreateDefaultConfigForce(); err != nil {
			logger.Error("Failed to create default config: %v", err)
			return fmt.Errorf("failed to create default config: %w", err)
		}

		mcpShellHome, err := utils.GetMCPShellHome()
		if err != nil {
			return fmt.Errorf("failed to get MCPShell home directory: %w", err)
		}

		configPath := filepath.Join(mcpShellHome, "agent.yaml")
		fmt.Printf("Default agent configuration created at: %s\n", configPath)
		fmt.Println("You can now edit this file to customize your agent settings.")

		return nil
	},
}

// ConfigShowOutput holds the JSON output structure for config show
type ConfigShowOutput struct {
	ConfigurationFile string                `json:"configuration_file"`
	Models            []ConfigShowModelInfo `json:"models"`
	DefaultModel      *ConfigShowModelInfo  `json:"default_model,omitempty"`
	Orchestrator      *ConfigShowModelInfo  `json:"orchestrator,omitempty"`
	ToolRunner        *ConfigShowModelInfo  `json:"tool_runner,omitempty"`
}

// ConfigShowModelInfo holds model info for JSON output
type ConfigShowModelInfo struct {
	Name          string   `json:"name"`
	Model         string   `json:"model"`
	Class         string   `json:"class"`
	Default       bool     `json:"default"`
	APIKey        string   `json:"api_key_masked,omitempty"`
	APIURL        string   `json:"api_url,omitempty"`
	SystemPrompts []string `json:"system_prompts,omitempty"`
}

// agentConfigShowCommand displays the current agent configuration
var agentConfigShowCommand = &cobra.Command{
	Use:   "show",
	Short: "Display the current agent configuration",
	Long: `

Displays the current agent configuration in a pretty-printed format.

The configuration is loaded from ~/.mcpshell/agent.yaml and parsed to show
the available models, their settings, and which model is set as default.

Use --json flag to output in JSON format for easy parsing by other tools.

Examples:
$ mcpshell agent config show
$ mcpshell agent config show --json
`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Get the config file path
		mcpShellHome, err := utils.GetMCPShellHome()
		if err != nil {
			return fmt.Errorf("failed to get MCPShell home directory: %w", err)
		}
		configPath := filepath.Join(mcpShellHome, "agent.yaml")

		// Load the current configuration
		config, err := agent.GetConfig()
		if err != nil {
			logger.Error("Failed to load config: %v", err)
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check if config is empty
		if len(config.Agent.Models) == 0 {
			if agentConfigShowJSON {
				output := ConfigShowOutput{
					ConfigurationFile: configPath,
					Models:            []ConfigShowModelInfo{},
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(output)
			}

			fmt.Printf("Configuration file: %s\n", configPath)
			fmt.Println()
			fmt.Println("No agent configuration found.")
			fmt.Println("Run 'mcpshell agent config create' to create a default configuration.")
			return nil
		}

		// Output in JSON format if requested
		if agentConfigShowJSON {
			return outputConfigShowJSON(configPath, config)
		}

		// Pretty print the configuration
		fmt.Printf("Configuration file: %s\n", configPath)
		fmt.Println()
		fmt.Println("Agent Configuration:")
		fmt.Println("===================")
		fmt.Println()

		for i, model := range config.Agent.Models {
			fmt.Printf("Model %d:\n", i+1)
			fmt.Printf("  Name: %s\n", model.Name)
			fmt.Printf("  Model: %s\n", model.Model)
			fmt.Printf("  Class: %s\n", model.Class)
			fmt.Printf("  Default: %t\n", model.Default)

			if model.APIKey != "" {
				if model.APIKey == "${OPENAI_API_KEY}" {
					fmt.Printf("  API Key: %s (from environment)\n", model.APIKey)
				} else {
					fmt.Printf("  API Key: %s\n", maskAPIKey(model.APIKey))
				}
			}

			if model.APIURL != "" {
				fmt.Printf("  API URL: %s\n", model.APIURL)
			}

			// Display prompts information
			if model.Prompts.HasSystemPrompts() {
				systemPrompts := model.Prompts.GetSystemPrompts()
				fmt.Printf("  System Prompts: %s\n", truncateString(systemPrompts, 80))
			}

			fmt.Println()
		}

		// Show which model is default
		defaultModel := config.GetDefaultModel()
		if defaultModel != nil {
			fmt.Printf("Default Model: %s (%s)\n", defaultModel.Name, defaultModel.Model)
		} else {
			fmt.Println("No default model configured.")
		}

		return nil
	},
}

// outputConfigShowJSON outputs the configuration in JSON format
func outputConfigShowJSON(configPath string, config *agent.Config) error {
	output := ConfigShowOutput{
		ConfigurationFile: configPath,
		Models:            make([]ConfigShowModelInfo, 0, len(config.Agent.Models)),
	}

	// Add all models
	for _, model := range config.Agent.Models {
		modelInfo := ConfigShowModelInfo{
			Name:    model.Name,
			Model:   model.Model,
			Class:   model.Class,
			Default: model.Default,
			APIURL:  model.APIURL,
		}

		if model.APIKey != "" {
			modelInfo.APIKey = maskAPIKey(model.APIKey)
		}

		if model.Prompts.HasSystemPrompts() {
			modelInfo.SystemPrompts = model.Prompts.System
		}

		output.Models = append(output.Models, modelInfo)
	}

	// Add default model
	if defaultModel := config.GetDefaultModel(); defaultModel != nil {
		modelInfo := ConfigShowModelInfo{
			Name:    defaultModel.Name,
			Model:   defaultModel.Model,
			Class:   defaultModel.Class,
			Default: defaultModel.Default,
			APIURL:  defaultModel.APIURL,
		}
		if defaultModel.APIKey != "" {
			modelInfo.APIKey = maskAPIKey(defaultModel.APIKey)
		}
		if defaultModel.Prompts.HasSystemPrompts() {
			modelInfo.SystemPrompts = defaultModel.Prompts.System
		}
		output.DefaultModel = &modelInfo
	}

	// Add orchestrator model if defined
	if orchestrator := config.GetOrchestratorModel(); orchestrator != nil {
		modelInfo := ConfigShowModelInfo{
			Name:   orchestrator.Name,
			Model:  orchestrator.Model,
			Class:  orchestrator.Class,
			APIURL: orchestrator.APIURL,
		}
		if orchestrator.APIKey != "" {
			modelInfo.APIKey = maskAPIKey(orchestrator.APIKey)
		}
		if orchestrator.Prompts.HasSystemPrompts() {
			modelInfo.SystemPrompts = orchestrator.Prompts.System
		}
		output.Orchestrator = &modelInfo
	}

	// Add tool runner model if defined
	if toolRunner := config.GetToolRunnerModel(); toolRunner != nil {
		modelInfo := ConfigShowModelInfo{
			Name:   toolRunner.Name,
			Model:  toolRunner.Model,
			Class:  toolRunner.Class,
			APIURL: toolRunner.APIURL,
		}
		if toolRunner.APIKey != "" {
			modelInfo.APIKey = maskAPIKey(toolRunner.APIKey)
		}
		if toolRunner.Prompts.HasSystemPrompts() {
			modelInfo.SystemPrompts = toolRunner.Prompts.System
		}
		output.ToolRunner = &modelInfo
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// Helper function to mask API keys for security
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// Helper function to truncate long strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	// Add create and show subcommands to agent config
	agentConfigCommand.AddCommand(agentConfigCreateCommand)
	agentConfigCommand.AddCommand(agentConfigShowCommand)

	// Add flags to show command
	agentConfigShowCommand.Flags().BoolVar(&agentConfigShowJSON, "json", false, "Output in JSON format")
}
