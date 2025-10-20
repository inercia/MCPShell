// Package agent provides model-specific client initialization and management functionality
package agent

import (
	"fmt"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/sashabaranov/go-openai"
)

// ModelProvider defines the interface for different model providers
type ModelProvider interface {
	// InitializeClient creates and configures the client for this model provider
	InitializeClient(config ModelConfig, logger *common.Logger) (*openai.Client, error)

	// ValidateConfig validates the configuration for this model provider
	ValidateConfig(config ModelConfig, logger *common.Logger) error

	// GetProviderName returns the human-readable name of the provider
	GetProviderName() string
}

// ModelManager manages different model providers and routes requests to the appropriate one
type ModelManager struct {
	providers map[string]ModelProvider
	logger    *common.Logger
}

// NewModelManager creates a new model manager with all supported providers
func NewModelManager(logger *common.Logger) *ModelManager {
	manager := &ModelManager{
		providers: make(map[string]ModelProvider),
		logger:    logger,
	}

	// Register all supported providers
	manager.RegisterProvider("openai", &OpenAIProvider{})
	manager.RegisterProvider("ollama", &OllamaProvider{})

	return manager
}

// RegisterProvider registers a new model provider
func (mm *ModelManager) RegisterProvider(class string, provider ModelProvider) {
	mm.providers[class] = provider
}

// InitializeClient initializes a client for the given model configuration
func (mm *ModelManager) InitializeClient(config ModelConfig) (*openai.Client, error) {
	provider := mm.getProvider(config.Class)
	return provider.InitializeClient(config, mm.logger)
}

// ValidateConfig validates the configuration for the given model class
func (mm *ModelManager) ValidateConfig(config ModelConfig) error {
	provider := mm.getProvider(config.Class)
	return provider.ValidateConfig(config, mm.logger)
}

// getProvider returns the appropriate provider for the given class
func (mm *ModelManager) getProvider(class string) ModelProvider {
	// Default to OpenAI if class is empty or not found
	if class == "" {
		class = "openai"
	}

	if provider, exists := mm.providers[class]; exists {
		return provider
	}

	// Return a generic provider for unknown classes
	return &GenericProvider{class: class}
}

// OpenAIProvider implements ModelProvider for OpenAI models
type OpenAIProvider struct{}

func (p *OpenAIProvider) InitializeClient(config ModelConfig, logger *common.Logger) (*openai.Client, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		logger.Error("API key is required for OpenAI models")
		return nil, fmt.Errorf("API key is required for OpenAI models")
	}

	clientConfig := openai.DefaultConfig(apiKey)
	if config.APIURL != "" {
		clientConfig.BaseURL = config.APIURL
	}

	client := openai.NewClientWithConfig(clientConfig)
	logger.Info("Initialized OpenAI client with model: %s", config.Model)
	return client, nil
}

func (p *OpenAIProvider) ValidateConfig(config ModelConfig, logger *common.Logger) error {
	if config.Model == "" {
		return fmt.Errorf("model name is required for OpenAI models")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key is required for OpenAI models (set API key environment variable or pass via config/flags)")
	}

	logger.Debug("OpenAI model configuration validated: %s", config.Model)
	return nil
}

func (p *OpenAIProvider) GetProviderName() string {
	return "OpenAI"
}

// OllamaProvider implements ModelProvider for Ollama models
type OllamaProvider struct{}

func (p *OllamaProvider) InitializeClient(config ModelConfig, logger *common.Logger) (*openai.Client, error) {
	// Ollama uses OpenAI-compatible API at localhost:11434
	apiKey := "ollama" // Ollama requires a dummy API key but doesn't use it
	clientConfig := openai.DefaultConfig(apiKey)
	clientConfig.BaseURL = "http://localhost:11434/v1"

	if config.APIURL != "" {
		// Allow override of Ollama URL if specified
		clientConfig.BaseURL = config.APIURL
	}

	client := openai.NewClientWithConfig(clientConfig)
	logger.Info("Initialized Ollama client with model: %s", config.Model)
	return client, nil
}

func (p *OllamaProvider) ValidateConfig(config ModelConfig, logger *common.Logger) error {
	if config.Model == "" {
		return fmt.Errorf("model name is required for Ollama models")
	}

	// API key is not required for Ollama
	logger.Debug("Ollama model configuration validated: %s", config.Model)
	return nil
}

func (p *OllamaProvider) GetProviderName() string {
	return "Ollama"
}

// GenericProvider implements ModelProvider for unknown/generic model types
// This allows for extensibility with other OpenAI-compatible APIs
type GenericProvider struct {
	class string
}

func (p *GenericProvider) InitializeClient(config ModelConfig, logger *common.Logger) (*openai.Client, error) {
	logger.Warn("Unknown model class '%s', treating as OpenAI-compatible", p.class)

	apiKey := config.APIKey
	if apiKey == "" {
		// Use a dummy key if none provided for unknown types
		apiKey = "unknown-model-type"
	}

	clientConfig := openai.DefaultConfig(apiKey)
	if config.APIURL != "" {
		clientConfig.BaseURL = config.APIURL
	}

	client := openai.NewClientWithConfig(clientConfig)
	logger.Info("Initialized OpenAI-compatible (%s) client with model: %s", p.class, config.Model)
	return client, nil
}

func (p *GenericProvider) ValidateConfig(config ModelConfig, logger *common.Logger) error {
	if config.Model == "" {
		return fmt.Errorf("model name is required")
	}

	logger.Info("Unknown model class '%s', performing basic validation", p.class)
	return nil
}

func (p *GenericProvider) GetProviderName() string {
	return fmt.Sprintf("OpenAI-compatible (%s)", p.class)
}

// Convenience functions for backward compatibility and ease of use

// InitializeModelClient creates and configures the appropriate model client based on the model class
func InitializeModelClient(config ModelConfig, logger *common.Logger) (*openai.Client, error) {
	manager := NewModelManager(logger)
	return manager.InitializeClient(config)
}

// ValidateModelConfig validates the model configuration for the specified model class
func ValidateModelConfig(config ModelConfig, logger *common.Logger) error {
	manager := NewModelManager(logger)
	return manager.ValidateConfig(config)
}
