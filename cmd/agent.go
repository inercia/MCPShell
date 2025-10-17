package root

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/inercia/MCPShell/pkg/agent"
	"github.com/inercia/MCPShell/pkg/common"
	toolsConfig "github.com/inercia/MCPShell/pkg/config"
)

// Cache the agent configuration to avoid duplicate resolution
var cachedAgentConfig agent.AgentConfig

// buildAgentConfig creates an AgentConfig by merging command-line flags with configuration file
func buildAgentConfig() (agent.AgentConfig, error) {
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

	// Resolve multiple config files into a single merged config file
	if len(toolsFiles) == 0 {
		return agent.AgentConfig{}, fmt.Errorf("tools configuration file(s) are required")
	}

	localConfigPath, _, err := toolsConfig.ResolveMultipleConfigPaths(toolsFiles, logger)
	if err != nil {
		return agent.AgentConfig{}, fmt.Errorf("failed to resolve config paths: %w", err)
	}

	return agent.AgentConfig{
		ToolsFile:   localConfigPath,
		UserPrompt:  agentUserPrompt,
		Once:        agentOnce,
		Version:     version,
		ModelConfig: modelConfig,
	}, nil
}

// agentCommand is a command that executes the MCPShell as an agent
var agentCommand = &cobra.Command{
	Use:   "agent",
	Short: "Execute the MCPShell as an agent",
	Long: `

The agent command will execute the MCPShell as an agent, connecting to a remote LLM.

Configuration is loaded from ~/.mcpshell/agent.yaml and can be overridden with command-line flags.
The configuration file should contain model definitions with their API keys and prompts.

For example, you can do:

$ mcpshell agent --tools=examples/config.yaml \
     --model "gpt-4o" \
     --system-prompt "You are a helpful assistant that debugs performance issues" \
     --user-prompt "I am having trouble with my computer. It is slow and I think it is due to the CPU usage."

If a model is configured as default in the agent configuration file, you can omit the --model flag:

$ mcpshell agent --tools=examples/config.yaml \
     --user-prompt "I am having trouble with my computer. It is slow and I think it is due to the CPU usage."

You can also provide the initial user prompt as positional arguments:

$ mcpshell agent I am having trouble with my computer. It is slow and I think it is due to the CPU usage.

The agent will try to debug the issue with the given tools.
`,
	Args: cobra.ArbitraryArgs,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// If --user-prompt is not provided but positional args exist, join them as the user prompt
		if agentUserPrompt == "" && len(args) > 0 {
			agentUserPrompt = strings.Join(args, " ")
		}

		// Initialize logger
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Build agent configuration (this will be cached for RunE)
		cachedAgentConfig, err = buildAgentConfig()
		if err != nil {
			return err
		}

		// Validate agent configuration
		agentInstance := agent.New(cachedAgentConfig, logger)
		if err := agentInstance.Validate(); err != nil {
			return err
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --user-prompt is not provided but positional args exist, join them as the user prompt
		if agentUserPrompt == "" && len(args) > 0 {
			agentUserPrompt = strings.Join(args, " ")
		}

		// Initialize logger
		logger, err := initLogger()
		if err != nil {
			return err
		}

		// Use cached agent configuration (built in PreRunE)
		agentConfig := cachedAgentConfig

		// Create agent instance
		agentInstance := agent.New(agentConfig, logger)

		// Create channels for user input and agent output
		userInput := make(chan string)
		agentOutput := make(chan string)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup signal handling for graceful shutdown
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-signalChan:
				logger.Info("Received interrupt signal, shutting down...")
				cancel()
			case <-ctx.Done():
			}
		}()

		// Start a goroutine to read user input only when not in --once mode
		if !agentConfig.Once {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(userInput) // Always close userInput when this goroutine exits

				scanner := bufio.NewScanner(os.Stdin)
				inputChan := make(chan string)

				// Start a separate goroutine to read from stdin
				go func() {
					for scanner.Scan() {
						inputChan <- scanner.Text()
					}
					close(inputChan)
				}()

				for {
					select {
					case <-ctx.Done():
						return
					case input, ok := <-inputChan:
						if !ok {
							return
						}
						select {
						case userInput <- input:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
		}

		// Start the agent
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := agentInstance.Run(ctx, userInput, agentOutput); err != nil {
				logger.Error(color.HiRedString("Agent encountered an error: %v", err))
				// Cancel context to abort all goroutines on fatal errors
				cancel()
			}
		}()

		// Print agent output (using Print not Println to respect formatting from event handler)
		for output := range agentOutput {
			fmt.Print(output)
		}

		// Wait for all goroutines with a timeout to prevent hanging
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All goroutines finished normally
			logger.Debug("All goroutines completed successfully")
		case <-time.After(5 * time.Second):
			// Force exit after timeout (agent already completed, this is just cleanup)
			logger.Debug("Cleanup timeout reached, forcing shutdown (agent task already completed)")
		}

		return nil
	},
}

// init adds the agent command to the root command
func init() {
	// Add agent command to root
	rootCmd.AddCommand(agentCommand)

	// Add agent-specific flags
	agentCommand.Flags().StringVarP(&agentModel, "model", "m", "", "LLM model to use (required)")
	agentCommand.Flags().StringVarP(&agentSystemPrompt, "system-prompt", "s", "", "System prompt for the LLM (optional, uses model-specific defaults if not provided)")
	agentCommand.Flags().StringVarP(&agentUserPrompt, "user-prompt", "u", "", "Initial user prompt for the LLM")
	agentCommand.Flags().StringVarP(&agentOpenAIApiKey, "openai-api-key", "k", "", "OpenAI API key (or set OPENAI_API_KEY environment variable)")
	agentCommand.Flags().StringVarP(&agentOpenAIApiURL, "openai-api-url", "b", "", "Base URL for the OpenAI API (optional)")
	agentCommand.Flags().BoolVarP(&agentOnce, "once", "o", false, "Exit after receiving a final response from the LLM (one-shot mode)")

	// Add config subcommand
	agentCommand.AddCommand(agentConfigCommand)
}
