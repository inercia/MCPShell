// Package agent provides MCP agent functionality that enables direct interaction
// between Large Language Models and command-line tools. The agent handles LLM
// communication, tool execution, and conversation management.

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/fatih/color"
	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/server"
)

// AgentConfig holds the configuration for the agent including tools file location,
// user prompts, execution mode, and embedded model configuration (API keys, model name, etc.)
type AgentConfig struct {
	ToolsFile   string // Path to the YAML configuration file defining available tools
	UserPrompt  string // Initial user prompt to send to the LLM
	Once        bool   // Whether to run in one-shot mode (exit after first response)
	Version     string // Version information for the agent
	ModelConfig        // Embedded model configuration (Model, APIKey, APIURL, Prompts)
}

// Agent represents an MCP agent
type Agent struct {
	config AgentConfig
	logger *common.Logger
}

// New creates a new agent instance
func New(cfg AgentConfig, logger *common.Logger) *Agent {
	return &Agent{
		config: cfg,
		logger: logger,
	}
}

// Validate checks if the configuration is valid
func (a *Agent) Validate() error {
	// Check if config file is provided
	if a.config.ToolsFile == "" {
		a.logger.Error("Tools configuration file is required")
		return fmt.Errorf("tools configuration file is required")
	}

	// Validate model configuration using the model manager
	if err := ValidateModelConfig(a.config.ModelConfig, a.logger); err != nil {
		a.logger.Error("Model configuration validation failed: %v", err)
		return fmt.Errorf("model configuration validation failed: %w", err)
	}

	return nil
}

// Run executes the agent using cagent multi-agent framework
func (a *Agent) Run(ctx context.Context, userInput chan string, agentOutput chan string) error {
	// Setup panic handler
	defer common.RecoverPanic()
	defer close(agentOutput) // Ensure agentOutput is closed when Run exits

	// Create server instance for MCP tools
	srv, cleanup, err := a.setupServer(ctx)
	if err != nil {
		agentOutput <- fmt.Sprintf("Error: %v", err)
		return err
	}
	defer cleanup() // Ensure cleanup is called

	// Load agent configuration to get orchestrator and tool-runner models
	config, err := GetConfig()
	if err != nil {
		a.logger.Error("Failed to load agent config: %v", err)
		agentOutput <- fmt.Sprintf("Error: Failed to load agent config: %v", err)
		return fmt.Errorf("failed to load agent config: %w", err)
	}

	// Get model configurations for orchestrator and tool-runner
	orchestratorConfig := a.config.ModelConfig
	if cfgOrch := config.GetOrchestratorModel(); cfgOrch != nil {
		// Merge config file settings with command-line overrides
		orchestratorConfig = mergeModelConfig(*cfgOrch, a.config.ModelConfig)
	}

	toolRunnerConfig := orchestratorConfig // Default to same as orchestrator
	if cfgTool := config.GetToolRunnerModel(); cfgTool != nil {
		// Merge config file settings with command-line overrides for tool runner
		toolRunnerConfig = mergeModelConfig(*cfgTool, a.config.ModelConfig)
	}

	a.logger.Info("Orchestrator model: %s (%s)", orchestratorConfig.Model, orchestratorConfig.Class)
	a.logger.Info("Tool-runner model: %s (%s)", toolRunnerConfig.Model, toolRunnerConfig.Class)

	// Create a single-run context if in --once mode
	if a.config.Once {
		// Create a context with a timeout to ensure we don't get stuck in --once mode
		singleRunCtx, singleRunCancel := context.WithTimeout(ctx, 120*time.Second)
		defer singleRunCancel()
		ctx = singleRunCtx
		a.logger.Info("Running in one-shot mode with 120s safety timeout")
	} else {
		a.logger.Info("Running in interactive mode (will wait for user input to continue)")
	}

	// Create cagent runtime with multi-agent system
	cagentRT, err := CreateCagentRuntime(ctx, srv, orchestratorConfig, toolRunnerConfig, a.config.UserPrompt, a.logger)
	if err != nil {
		a.logger.Error("Failed to create cagent runtime: %v", err)
		agentOutput <- fmt.Sprintf("Error: Failed to create cagent runtime: %v", err)
		return fmt.Errorf("failed to create cagent runtime: %w", err)
	}

	// Conversation loop - run until Once mode or context cancellation
	for {
		// Start streaming events from cagent
		a.logger.Debug("Starting cagent event stream")
		events := cagentRT.RunStream(ctx)

		// Process events and send output
		eventCount := 0
		for event := range events {
			eventCount++
			a.logger.Debug("Received event #%d: %T", eventCount, event)

			// Handle tool call confirmations - auto-approve tools
			if _, ok := event.(*runtime.ToolCallConfirmationEvent); ok {
				a.logger.Debug("Auto-approving tool execution")
				cagentRT.Runtime().Resume(ctx, "approve-session")
			}

			if err := a.handleCagentEvent(event, agentOutput); err != nil {
				a.logger.Error("Error handling event: %v", err)
				// Continue processing other events
			}
		}
		a.logger.Debug("Event stream completed, processed %d events", eventCount)

		// In one-shot mode, exit after first response
		if a.config.Once {
			a.logger.Info("One-shot mode: exiting after first response")
			return nil
		}

		// In interactive mode, wait for user input to continue
		a.logger.Debug("Waiting for user input to continue conversation...")
		promptColor := color.New(color.Bold, color.FgHiCyan)
		agentOutput <- fmt.Sprintf("\n%s", promptColor.Sprint("💬 Enter your next question (or Ctrl+C to exit): "))

		select {
		case <-ctx.Done():
			a.logger.Info("Context cancelled, exiting")
			return ctx.Err()
		case nextInput, ok := <-userInput:
			if !ok {
				a.logger.Info("User input channel closed, exiting")
				return nil
			}
			if nextInput == "" {
				continue // Skip empty input
			}

			// Add the new user message to the session to continue the conversation
			a.logger.Debug("Received user input: %s", nextInput)
			if err := cagentRT.ContinueConversation(nextInput); err != nil {
				a.logger.Error("Failed to continue conversation: %v", err)
				agentOutput <- fmt.Sprintf("Error: %v\n", err)
				return fmt.Errorf("failed to continue conversation: %w", err)
			}
			// Loop will continue with the updated session
		}
	}
}

// handleCagentEvent processes a single cagent event and sends appropriate output
func (a *Agent) handleCagentEvent(event interface{}, agentOutput chan string) error {
	a.logger.Debug("Handling event type: %T", event)

	// Define color schemes for different outputs
	cyan := color.New(color.FgCyan)
	green := color.New(color.FgGreen)     // Agent thinking/responses
	blue := color.New(color.FgBlue)       // Tool results
	yellow := color.New(color.FgYellow)   // Tool calls
	magenta := color.New(color.FgMagenta) // Agent status

	// Use concrete types from cagent runtime package
	switch e := event.(type) {
	case *runtime.AgentChoiceEvent:
		// Agent is thinking/responding with text - stream in green
		if e.Content != "" {
			// Send colored content to distinguish agent text from system messages
			agentOutput <- green.Sprint(e.Content)
		}

	case *runtime.PartialToolCallEvent:
		// Tool call is being built incrementally - accumulate or just log
		a.logger.Debug("Building tool call: %s", e.ToolCall.Function.Name)

	case *runtime.ToolCallEvent:
		// Complete tool call is ready - use yellow for tool calls
		toolName := e.ToolCall.Function.Name
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(e.ToolCall.Function.Arguments), &args); err == nil {
			argsJSON, _ := json.MarshalIndent(args, "", "  ")
			agentOutput <- fmt.Sprintf("\n%s\n%s\n",
				yellow.Sprintf("→ [%s] Calling tool '%s' with args:", e.AgentName, toolName),
				cyan.Sprint(string(argsJSON)))
		} else {
			agentOutput <- fmt.Sprintf("\n%s\n", yellow.Sprintf("→ [%s] Calling tool '%s'", e.AgentName, toolName))
		}

	case *runtime.ToolCallConfirmationEvent:
		// Tool is being confirmed/executed
		// Add newline before logs to separate from agent output
		agentOutput <- "\n"
		a.logger.Debug("Tool call confirmed for agent: %s", e.AgentName)

	case *runtime.ToolCallResponseEvent:
		// Tool execution result - use blue for tool output
		response := e.Response
		if len(response) > 1000 {
			response = response[:1000] + "... (truncated)"
		}
		agentOutput <- fmt.Sprintf("%s\n%s\n%s\n",
			blue.Sprint("--- tool result BEGIN ---"),
			blue.Sprint(response),
			blue.Sprint("--- tool result END ---"))

	case *runtime.StreamStartedEvent:
		// Agent started processing - use magenta for agent status
		agentOutput <- fmt.Sprintf("\n%s\n\n", magenta.Sprintf("[%s started]", e.AgentName))

	case *runtime.StreamStoppedEvent:
		// Agent finished processing - use magenta for agent status
		// Add newlines before the completion message to ensure separation from streamed text
		agentOutput <- fmt.Sprintf("\n\n%s\n\n", magenta.Sprintf("[%s completed]", e.AgentName))
		a.logger.Debug("Agent %s stream stopped", e.AgentName)

	case *runtime.UserMessageEvent:
		// User message being processed
		a.logger.Debug("Processing user message")

	case *runtime.TokenUsageEvent:
		// Token usage info
		if e.Usage != nil {
			a.logger.Debug("Token usage: input=%d, output=%d", e.Usage.InputTokens, e.Usage.OutputTokens)
		}

	default:
		// Unknown event type
		a.logger.Debug("Unhandled event type: %T", event)
	}

	return nil
}

// mergeModelConfig merges a base configuration with override values
// Override values (from command-line) take precedence over base values (from config file)
func mergeModelConfig(base, override ModelConfig) ModelConfig {
	result := base

	// Override specific fields if they were set via command-line
	if override.Model != "" {
		result.Model = override.Model
	}
	if override.Class != "" {
		result.Class = override.Class
	}
	if override.APIKey != "" {
		result.APIKey = override.APIKey
	}
	if override.APIURL != "" {
		result.APIURL = override.APIURL
	}
	// Merge prompts - command-line prompts are added to config file prompts
	if override.Prompts.HasSystemPrompts() {
		if result.Prompts.System == nil {
			result.Prompts.System = override.Prompts.System
		} else {
			result.Prompts.System = append(result.Prompts.System, override.Prompts.System...)
		}
	}

	return result
}

// setupServer initializes and creates the MCP server
func (a *Agent) setupServer(ctx context.Context) (*server.Server, func(), error) {
	// Use the already resolved configuration file path (no need to resolve again)
	localConfigPath := a.config.ToolsFile
	cleanup := func() {} // No cleanup needed since path was already resolved

	// Initialize MCP server to get tools
	a.logger.Info("Initializing MCP server")
	srv := server.New(server.Config{
		ConfigFile: localConfigPath,
		Logger:     a.logger,
		Version:    a.config.Version,
	})

	// Create the server instance (but don't start it)
	if err := srv.CreateServer(); err != nil {
		a.logger.Error("Failed to create MCP server: %v", err)
		return nil, cleanup, fmt.Errorf("failed to create MCP server: %w", err)
	}

	return srv, cleanup, nil
}
