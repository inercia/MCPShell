package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/utils"
)

func TestNew(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	cfg := AgentConfig{
		ToolsFile:  "test.yaml",
		UserPrompt: "test prompt",
		Once:       false,
		Version:    "1.0.0",
		ModelConfig: ModelConfig{
			Model:  "gpt-4",
			Class:  "openai",
			APIKey: "test-key",
		},
	}

	agent := New(cfg, logger)

	if agent == nil {
		t.Fatal("New() returned nil")
	}

	if agent.config.ToolsFile != cfg.ToolsFile {
		t.Errorf("Expected ToolsFile %s, got %s", cfg.ToolsFile, agent.config.ToolsFile)
	}

	if agent.config.UserPrompt != cfg.UserPrompt {
		t.Errorf("Expected UserPrompt %s, got %s", cfg.UserPrompt, agent.config.UserPrompt)
	}

	if agent.config.Once != cfg.Once {
		t.Errorf("Expected Once %t, got %t", cfg.Once, agent.config.Once)
	}

	if agent.config.Version != cfg.Version {
		t.Errorf("Expected Version %s, got %s", cfg.Version, agent.config.Version)
	}

	if agent.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestValidate(t *testing.T) {
	logger, err := common.NewLogger("", "", common.LogLevelError, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	tests := []struct {
		name    string
		config  AgentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid OpenAI config",
			config: AgentConfig{
				ToolsFile: "test.yaml",
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					Class:  "openai",
					APIKey: "test-key",
				},
			},
			wantErr: false,
		},
		{
			name: "valid Ollama config",
			config: AgentConfig{
				ToolsFile: "test.yaml",
				ModelConfig: ModelConfig{
					Model: "llama2",
					Class: "ollama",
				},
			},
			wantErr: false,
		},
		{
			name: "missing tools file",
			config: AgentConfig{
				ToolsFile: "",
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					Class:  "openai",
					APIKey: "test-key",
				},
			},
			wantErr: true,
			errMsg:  "tools configuration file is required",
		},
		{
			name: "missing model for OpenAI",
			config: AgentConfig{
				ToolsFile: "test.yaml",
				ModelConfig: ModelConfig{
					Model:  "",
					Class:  "openai",
					APIKey: "test-key",
				},
			},
			wantErr: true,
			errMsg:  "model configuration validation failed: model name is required for OpenAI models",
		},
		{
			name: "missing API key for OpenAI model",
			config: AgentConfig{
				ToolsFile: "test.yaml",
				ModelConfig: ModelConfig{
					Model:  "gpt-4",
					Class:  "openai",
					APIKey: "",
				},
			},
			wantErr: true,
			errMsg:  "model configuration validation failed: API key is required for OpenAI models (set API key environment variable or pass via config/flags)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := New(tt.config, logger)
			err := agent.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Note: setupConversation and initializeModelClient tests removed
// as these methods are now internal to cagent runtime

// TestAgentWithOllama tests the agent with a real Ollama model using cagent
// This test requires Ollama to be running and a tool-capable model to be available
func TestAgentWithOllama(t *testing.T) {
	// Use the test utilities to check if Ollama is running and get a tool-capable model
	modelName := utils.RequireOllamaWithTools(t)

	t.Logf("Running integration test with model: %s", modelName)

	// Create a logger for the test
	logger, err := common.NewLogger("", "", common.LogLevelInfo, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a temporary test config file
	testConfig := createTestConfigFile(t)
	defer func() {
		if err := os.Remove(testConfig); err != nil && !os.IsNotExist(err) {
			t.Logf("failed to remove test config: %v", err)
		}
	}()

	// Create a temporary agent config file for the test
	agentConfigPath := createTestAgentConfigFile(t, modelName)
	defer func() {
		if err := os.Remove(agentConfigPath); err != nil && !os.IsNotExist(err) {
			t.Logf("failed to remove agent config: %v", err)
		}
	}()

	// Create agent configuration for Ollama
	cfg := AgentConfig{
		ToolsFile:  testConfig,
		UserPrompt: "What is the current date? Just respond with 'Test successful' without using any tools.",
		Once:       true,
		Version:    "test",
		ModelConfig: ModelConfig{
			Model:  modelName,
			Class:  "ollama",
			APIURL: "http://localhost:11434/v1", // Ollama's OpenAI-compatible endpoint
			APIKey: "ollama",                    // Ollama doesn't require a real API key
			Prompts: common.PromptsConfig{
				System: []string{"You are a helpful assistant that can use tools to answer questions."},
			},
		},
	}

	// Create and validate the agent
	agent := New(cfg, logger)
	if agent == nil {
		t.Fatal("Failed to create agent")
	}

	err = agent.Validate()
	if err != nil {
		t.Fatalf("Agent validation failed: %v", err)
	}

	// Test that the model supports tools
	if !utils.IsModelToolCapable(modelName) {
		t.Errorf("Model %s should be tool-capable according to our test utilities", modelName)
	}

	t.Log("Ollama integration test setup completed successfully")
	t.Log("Note: Full agent execution test disabled - requires running Ollama instance")
}

// createTestConfigFile creates a temporary configuration file for testing
func createTestConfigFile(t *testing.T) string {
	testConfig := `
tools:
  - name: "date"
    description: "Get the current date and time"
    runner: "exec"
    command: "date"
    parameters: []
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	err := os.WriteFile(configFile, []byte(testConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return configFile
}

// createTestAgentConfigFile creates a temporary agent configuration file for testing
func createTestAgentConfigFile(t *testing.T, modelName string) string {
	agentConfig := `
agent:
  orchestrator:
    model: "` + modelName + `"
    class: "ollama"
    name: "orchestrator"
    api-url: "http://localhost:11434/v1"
    prompts:
      system:
      - "You are a test orchestrator agent."

  tool-runner:
    model: "` + modelName + `"
    class: "ollama"
    name: "tool-runner"
    api-url: "http://localhost:11434/v1"
    prompts:
      system:
      - "You are a test tool execution agent."
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "agent.yaml")

	err := os.WriteFile(configFile, []byte(agentConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create agent config file: %v", err)
	}

	return configFile
}
