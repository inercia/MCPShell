// Package agent provides cagent runtime configuration and setup
package agent

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	cagentAgent "github.com/docker/cagent/pkg/agent"
	cagentConfig "github.com/docker/cagent/pkg/config/v2"
	"github.com/docker/cagent/pkg/environment"
	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/team"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/server"
)

//go:embed prompts/orchestrator.md
var defaultOrchestratorPrompt string

// CagentRuntime wraps the cagent runtime and session
type CagentRuntime struct {
	runtime runtime.Runtime
	session *session.Session
	logger  *common.Logger
}

// CreateCagentRuntime creates and configures a cagent runtime
// Uses a single agent approach for better tool execution continuity
func CreateCagentRuntime(
	ctx context.Context,
	srv *server.Server,
	orchestratorConfig ModelConfig,
	toolRunnerConfig ModelConfig,
	userPrompt string,
	logger *common.Logger,
) (*CagentRuntime, error) {
	logger.Debug("Creating cagent single-agent runtime")

	// Use orchestrator config for the single agent
	agentLLM, err := initializeCagentModel(ctx, orchestratorConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent model: %w", err)
	}

	// Create MCP tool set
	mcpToolSet := NewMCPToolSet(srv, logger)
	tools, err := mcpToolSet.GetTools()
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP tools: %w", err)
	}

	logger.Debug("Creating single agent with %d MCP tools", len(tools))

	// Get system prompts - use tool-runner prompt since this agent will execute tools
	// Use config prompts if provided, otherwise use embedded default
	agentSysPrompt := orchestratorConfig.Prompts.GetSystemPrompts()
	if agentSysPrompt == "" {
		logger.Debug("Using default embedded prompt for agent")
		agentSysPrompt = defaultOrchestratorPrompt
	} else {
		logger.Debug("Using custom prompt from config for agent")
	}
	logger.Debug("Agent prompt (first 200 chars): %s", func() string {
		if len(agentSysPrompt) > 200 {
			return agentSysPrompt[:200] + "..."
		}
		return agentSysPrompt
	}())

	// Create a single agent with all tools
	agent := cagentAgent.New(
		"root",
		agentSysPrompt,
		cagentAgent.WithModel(agentLLM),
		cagentAgent.WithDescription("An agent that executes tools to accomplish user tasks"),
		cagentAgent.WithTools(tools...),
		cagentAgent.WithMaxIterations(50), // Allow up to 50 tool calls
	)

	// Create the team with just the one agent
	agentTeam := team.New(team.WithAgents(agent))

	// Create the runtime with session compaction enabled
	rt, err := runtime.New(
		agentTeam,
		runtime.WithSessionCompaction(true), // Auto-summarize when approaching context limit
	)
	if err != nil {
		logger.Error("Failed to create cagent runtime: %v", err)
		return nil, fmt.Errorf("failed to create cagent runtime: %w", err)
	}

	// Create the session with the user prompt
	// Enhance prompt to emphasize iterative workflow
	enhancedPrompt := userPrompt + `

Remember: This is a multi-step investigation. Keep calling tools iteratively until you have ALL the information needed to fully answer the question. Don't stop after just one tool call.`

	sess := session.New(session.WithUserMessage("", enhancedPrompt))

	logger.Debug("Cagent single-agent runtime created successfully")

	return &CagentRuntime{
		runtime: rt,
		session: sess,
		logger:  logger,
	}, nil
}

// RunStream starts the streaming runtime and returns the event channel
func (cr *CagentRuntime) RunStream(ctx context.Context) <-chan runtime.Event {
	cr.logger.Debug("Starting cagent runtime stream")
	return cr.runtime.RunStream(ctx, cr.session)
}

// Runtime returns the underlying cagent runtime for advanced operations like Resume
func (cr *CagentRuntime) Runtime() runtime.Runtime {
	return cr.runtime
}

// ContinueConversation adds a new user message to the session and continues the conversation
func (cr *CagentRuntime) ContinueConversation(userMessage string) error {
	cr.logger.Debug("Adding user message to continue conversation")

	// Add the user message to the existing session
	msg := session.UserMessage("", userMessage)
	cr.session.AddMessage(msg)

	cr.logger.Debug("User message added to session, ready for next stream")
	return nil
}

// initializeCagentModel creates a cagent-compatible model provider from our ModelConfig
func initializeCagentModel(ctx context.Context, config ModelConfig, logger *common.Logger) (provider.Provider, error) {
	// Create cagent model configuration
	cagentModelConfig := &cagentConfig.ModelConfig{
		Provider: config.Class,
		Model:    config.Model,
	}

	// Handle provider name mapping
	// Default to openai if no class specified
	if cagentModelConfig.Provider == "" {
		cagentModelConfig.Provider = "openai"
		logger.Debug("No provider specified, defaulting to openai")
	}

	// Map "ollama" to "openai" since Ollama uses OpenAI-compatible API
	if cagentModelConfig.Provider == "ollama" {
		cagentModelConfig.Provider = "openai"
		logger.Debug("Mapping ollama provider to openai (OpenAI-compatible)")

		// Set default Ollama URL if not specified
		if config.APIURL == "" {
			config.APIURL = "http://localhost:11434/v1"
			logger.Debug("Using default Ollama URL: http://localhost:11434/v1")
		} else {
			logger.Debug("Using custom Ollama URL: %s", config.APIURL)
		}

		// Ollama doesn't require an API key, set a dummy one if not provided
		if config.APIKey == "" {
			config.APIKey = "ollama-no-key-required"
			logger.Debug("Setting dummy API key for Ollama (no key required)")
		}
	}

	// Set BaseURL if provided in config
	if config.APIURL != "" {
		cagentModelConfig.BaseURL = config.APIURL
		logger.Debug("Setting base URL: %s", config.APIURL)
	}

	logger.Debug("Initializing cagent model: provider=%s, model=%s",
		cagentModelConfig.Provider, cagentModelConfig.Model)

	// Create environment provider for API keys
	// Set API key from config into environment if provided
	if config.APIKey != "" {
		_ = os.Setenv("OPENAI_API_KEY", config.APIKey)
		logger.Debug("Setting API key from config into environment")
	}

	envProvider := environment.NewDefaultProvider()

	client, err := provider.New(ctx, cagentModelConfig, envProvider)
	if err != nil {
		logger.Error("Failed to create model provider '%s': %v", cagentModelConfig.Provider, err)
		return nil, fmt.Errorf("failed to create model provider '%s': %w", cagentModelConfig.Provider, err)
	}

	logger.Debug("Successfully initialized %s provider for model %s",
		cagentModelConfig.Provider, cagentModelConfig.Model)
	return client, nil
}
