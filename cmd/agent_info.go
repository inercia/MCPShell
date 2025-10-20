package root

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"

	"github.com/inercia/MCPShell/pkg/agent"
	"github.com/inercia/MCPShell/pkg/common"
	toolsConfig "github.com/inercia/MCPShell/pkg/config"
	"github.com/inercia/MCPShell/pkg/utils"
)

var (
	agentInfoJSON           bool
	agentInfoIncludePrompts bool
	agentInfoCheck          bool
)

// agentInfoCommand displays information about the agent configuration
var agentInfoCommand = &cobra.Command{
	Use:   "info",
	Short: "Display agent configuration information",
	Long: `
Display information about the agent configuration including:
- LLM model details
- API configuration
- System prompts (with --include-prompts)
- LLM connectivity status (with --check)

The configuration is loaded from ~/.mcpshell/agent.yaml and merged with
command-line flags (if provided).

The --tools flag is optional for this command. It's only needed if you want
to verify the full agent configuration including tools setup.

Examples:
$ mcpshell agent info
$ mcpshell agent info --json
$ mcpshell agent info --include-prompts
$ mcpshell agent info --check
$ mcpshell agent info --model gpt-4o --json
$ mcpshell agent info --tools examples/config.yaml
`,
	Args: cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logger
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Tools are optional for agent info - we only need them for actual agent execution
		logger.Debug("Agent info command initialized")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Build agent configuration (tools are optional for info command)
		agentConfig, err := buildAgentConfigForInfo()
		if err != nil {
			return fmt.Errorf("failed to build agent config: %w", err)
		}

		// Use the model config that was built - it already has the correct model
		// based on: --model flag > MCPSHELL_AGENT_MODEL env var > default from config
		orchestratorConfig := agentConfig.ModelConfig
		toolRunnerConfig := agentConfig.ModelConfig

		// Check LLM connectivity if requested
		var checkResult *CheckResult
		if agentInfoCheck {
			checkResult = checkLLMConnectivity(orchestratorConfig, logger)
		}

		// Output in JSON format if requested
		if agentInfoJSON {
			err := outputJSON(agentConfig, orchestratorConfig, toolRunnerConfig, checkResult)
			if err != nil {
				return err
			}
			// If check was performed and failed, exit with error
			if checkResult != nil && !checkResult.Success {
				return fmt.Errorf("LLM connectivity check failed: %s", checkResult.Error)
			}
			return nil
		}

		// Output in human-readable format
		return outputHumanReadable(agentConfig, orchestratorConfig, toolRunnerConfig, checkResult)
	},
}

// CheckResult holds the result of an LLM connectivity check
type CheckResult struct {
	Success      bool    `json:"success"`
	ResponseTime float64 `json:"response_time_ms"`
	Error        string  `json:"error,omitempty"`
	Model        string  `json:"model"`
}

// InfoOutput holds the complete info output structure for JSON
type InfoOutput struct {
	ConfigFile   string       `json:"config_file,omitempty"`
	ToolsFile    string       `json:"tools_file,omitempty"`
	Once         bool         `json:"once_mode"`
	Orchestrator ModelInfo    `json:"orchestrator"`
	ToolRunner   ModelInfo    `json:"tool_runner"`
	Check        *CheckResult `json:"check,omitempty"`
	Prompts      *PromptsInfo `json:"prompts,omitempty"`
}

// ModelInfo holds model configuration details for JSON output
type ModelInfo struct {
	Model  string `json:"model"`
	Class  string `json:"class"`
	Name   string `json:"name,omitempty"`
	APIURL string `json:"api_url,omitempty"`
	APIKey string `json:"api_key_masked,omitempty"`
}

// PromptsInfo holds prompt information for JSON output
type PromptsInfo struct {
	System []string `json:"system,omitempty"`
	User   string   `json:"user,omitempty"`
}

// buildAgentConfigForInfo creates an AgentConfig for the info command
// Unlike buildAgentConfig, this doesn't require tools files
func buildAgentConfigForInfo() (agent.AgentConfig, error) {
	// Load configuration from file
	config, err := agent.GetConfig()
	if err != nil {
		return agent.AgentConfig{}, fmt.Errorf("failed to load config: %w", err)
	}

	// Start with default model from config file
	var modelConfig agent.ModelConfig
	if defaultModel := config.GetDefaultModel(); defaultModel != nil {
		modelConfig = *defaultModel
	}

	logger := common.GetLogger()

	// If --model flag not provided, check for environment variable
	if agentModel == "" {
		if envModel := os.Getenv("MCPSHELL_AGENT_MODEL"); envModel != "" {
			agentModel = envModel
			logger.Debug("Using model from MCPSHELL_AGENT_MODEL environment variable: %s", agentModel)
		}
	}

	// Override with command-line flags if provided
	if agentModel != "" {
		logger.Debug("Looking for model '%s' in agent config", agentModel)

		// Check if the specified model exists in config
		if configModel := config.GetModelByName(agentModel); configModel != nil {
			modelConfig = *configModel
			logger.Info("Found model '%s' in config: model=%s, class=%s, name=%s",
				agentModel, configModel.Model, configModel.Class, configModel.Name)
		} else {
			// Use command-line model name if not found in config
			logger.Info("Model '%s' not found in config, using as direct model name", agentModel)
			modelConfig.Model = agentModel
		}
	}

	// Merge system prompts from config file and command-line
	if agentSystemPrompt != "" {
		// Join system prompts from config with command-line system prompt
		var allSystemPrompts []string

		// Add existing system prompts from config
		if modelConfig.Prompts.HasSystemPrompts() {
			allSystemPrompts = append(allSystemPrompts, modelConfig.Prompts.System...)
		}

		// Add command-line system prompt
		allSystemPrompts = append(allSystemPrompts, agentSystemPrompt)

		// Update the model config with merged prompts
		modelConfig.Prompts.System = allSystemPrompts
	}

	// Override API key and URL if provided
	if agentOpenAIApiKey != "" {
		modelConfig.APIKey = agentOpenAIApiKey
	}
	if agentOpenAIApiURL != "" {
		modelConfig.APIURL = agentOpenAIApiURL
	}

	// Handle environment variable substitution for API key
	if strings.HasPrefix(modelConfig.APIKey, "${") && strings.HasSuffix(modelConfig.APIKey, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(modelConfig.APIKey, "${"), "}")
		modelConfig.APIKey = os.Getenv(envVar)
		logger.Debug("Substituted API key from environment variable: %s", envVar)
	}

	// Handle environment variable substitution for API URL
	if strings.HasPrefix(modelConfig.APIURL, "${") && strings.HasSuffix(modelConfig.APIURL, "}") {
		envVar := strings.TrimSuffix(strings.TrimPrefix(modelConfig.APIURL, "${"), "}")
		modelConfig.APIURL = os.Getenv(envVar)
		logger.Debug("Substituted API URL from environment variable: %s = %s", envVar, modelConfig.APIURL)
	}

	// Tools file is optional for info command
	toolsFile := ""
	if len(toolsFiles) > 0 {
		// Resolve tools configuration if provided
		localConfigPath, _, err := toolsConfig.ResolveMultipleConfigPaths(toolsFiles, logger)
		if err != nil {
			return agent.AgentConfig{}, fmt.Errorf("failed to resolve config paths: %w", err)
		}
		toolsFile = localConfigPath
	}

	return agent.AgentConfig{
		ToolsFile:   toolsFile,
		UserPrompt:  agentUserPrompt,
		Once:        agentOnce,
		Version:     version,
		ModelConfig: modelConfig,
	}, nil
}

// checkLLMConnectivity tests if the LLM is responding
func checkLLMConnectivity(modelConfig agent.ModelConfig, logger *common.Logger) *CheckResult {
	result := &CheckResult{
		Model: modelConfig.Model,
	}

	logger.Info("Testing LLM connectivity for model: %s", modelConfig.Model)

	// Initialize the model client
	client, err := agent.InitializeModelClient(modelConfig, logger)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to initialize client: %v", err)
		return result
	}

	// Make a simple test request
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()

	req := openai.ChatCompletionRequest{
		Model: modelConfig.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "Respond with just the word 'OK'",
			},
		},
		MaxTokens: 10,
	}

	_, err = client.CreateChatCompletion(ctx, req)
	elapsed := time.Since(startTime)

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("LLM request failed: %v", err)
		logger.Error("LLM connectivity check failed: %v", err)
		return result
	}

	result.Success = true
	result.ResponseTime = float64(elapsed.Milliseconds())
	logger.Info("LLM connectivity check successful (%.0fms)", result.ResponseTime)

	return result
}

// outputJSON outputs the configuration in JSON format
func outputJSON(agentConfig agent.AgentConfig, orchestrator, toolRunner agent.ModelConfig, check *CheckResult) error {
	// Get agent config file path
	var configFile string
	if mcpShellHome, err := utils.GetMCPShellHome(); err == nil {
		configFile = filepath.Join(mcpShellHome, "agent.yaml")
	}

	output := InfoOutput{
		ConfigFile: configFile,
		ToolsFile:  agentConfig.ToolsFile,
		Once:       agentConfig.Once,
		Orchestrator: ModelInfo{
			Model:  orchestrator.Model,
			Class:  orchestrator.Class,
			Name:   orchestrator.Name,
			APIURL: orchestrator.APIURL,
			APIKey: maskAPIKey(orchestrator.APIKey),
		},
		ToolRunner: ModelInfo{
			Model:  toolRunner.Model,
			Class:  toolRunner.Class,
			Name:   toolRunner.Name,
			APIURL: toolRunner.APIURL,
			APIKey: maskAPIKey(toolRunner.APIKey),
		},
		Check: check,
	}

	// Include prompts if requested
	if agentInfoIncludePrompts {
		output.Prompts = &PromptsInfo{
			System: orchestrator.Prompts.System,
			User:   agentConfig.UserPrompt,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputHumanReadable outputs the configuration in human-readable format
func outputHumanReadable(agentConfig agent.AgentConfig, orchestrator, toolRunner agent.ModelConfig, check *CheckResult) error {
	fmt.Println(color.HiCyanString("Agent Configuration"))
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	// Show agent config file location
	mcpShellHome, err := utils.GetMCPShellHome()
	if err == nil {
		agentConfigPath := filepath.Join(mcpShellHome, "agent.yaml")
		fmt.Printf("Config File:   %s\n", agentConfigPath)
	}

	// General settings
	if agentConfig.ToolsFile != "" {
		fmt.Printf("Tools File:    %s\n", agentConfig.ToolsFile)
	}
	fmt.Printf("Once Mode:     %t\n", agentConfig.Once)
	fmt.Println()

	// Orchestrator model
	fmt.Println(color.HiYellowString("Orchestrator Model:"))
	fmt.Printf("  Model:       %s\n", orchestrator.Model)
	if orchestrator.Name != "" {
		fmt.Printf("  Name:        %s\n", orchestrator.Name)
	}
	fmt.Printf("  Class:       %s\n", orchestrator.Class)
	if orchestrator.APIURL != "" {
		fmt.Printf("  API URL:     %s\n", orchestrator.APIURL)
	}
	if orchestrator.APIKey != "" {
		fmt.Printf("  API Key:     %s\n", maskAPIKey(orchestrator.APIKey))
	}
	fmt.Println()

	// Tool-runner model (only if different from orchestrator)
	if toolRunner.Model != orchestrator.Model || toolRunner.Class != orchestrator.Class {
		fmt.Println(color.HiYellowString("Tool-Runner Model:"))
		fmt.Printf("  Model:       %s\n", toolRunner.Model)
		if toolRunner.Name != "" {
			fmt.Printf("  Name:        %s\n", toolRunner.Name)
		}
		fmt.Printf("  Class:       %s\n", toolRunner.Class)
		if toolRunner.APIURL != "" {
			fmt.Printf("  API URL:     %s\n", toolRunner.APIURL)
		}
		if toolRunner.APIKey != "" {
			fmt.Printf("  API Key:     %s\n", maskAPIKey(toolRunner.APIKey))
		}
		fmt.Println()
	}

	// Prompts (if requested)
	if agentInfoIncludePrompts {
		fmt.Println(color.HiYellowString("Prompts:"))
		if orchestrator.Prompts.HasSystemPrompts() {
			fmt.Println(color.CyanString("  System Prompts:"))
			for i, prompt := range orchestrator.Prompts.System {
				fmt.Printf("    %d. %s\n", i+1, truncateString(prompt, 120))
			}
		} else {
			fmt.Println("  System Prompts: (none)")
		}
		if agentConfig.UserPrompt != "" {
			fmt.Printf("  User Prompt:   %s\n", truncateString(agentConfig.UserPrompt, 120))
		}
		fmt.Println()
	}

	// Check result (if performed)
	if check != nil {
		fmt.Println(color.HiYellowString("LLM Connectivity Check:"))
		if check.Success {
			fmt.Printf("  Status:      %s\n", color.HiGreenString("✓ Connected"))
			fmt.Printf("  Response:    %.0fms\n", check.ResponseTime)
		} else {
			fmt.Printf("  Status:      %s\n", color.HiRedString("✗ Failed"))
			fmt.Printf("  Error:       %s\n", check.Error)
			return fmt.Errorf("LLM connectivity check failed: %s", check.Error)
		}
		fmt.Println()
	}

	return nil
}

func init() {
	// Add info subcommand to agent command
	agentCommand.AddCommand(agentInfoCommand)

	// Add info-specific flags
	agentInfoCommand.Flags().BoolVar(&agentInfoJSON, "json", false, "Output in JSON format (for easy parsing)")
	agentInfoCommand.Flags().BoolVar(&agentInfoIncludePrompts, "include-prompts", false, "Include full prompts in the output")
	agentInfoCommand.Flags().BoolVar(&agentInfoCheck, "check", false, "Check LLM connectivity (exits with error if LLM is not responding)")
}
