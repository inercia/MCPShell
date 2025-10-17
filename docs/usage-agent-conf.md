# Agent Configuration

The MCPShell agent supports configuration through both configuration files and command-line flags. Configuration files provide a convenient way to manage multiple model configurations and default settings.

## Configuration File

The agent looks for configuration in `~/.mcpshell/agent.yaml`. This file defines model configurations, API keys, and default prompts.

### Configuration Structure

```yaml
agent:
  models:
    - model: "gpt-4o"
      class: "openai"
      name: "GPT-4o Agent"
      default: true
      api-key: "${OPENAI_API_KEY}"
      api-url: "https://api.openai.com/v1"
      prompts:
        system:
          - "You are a helpful assistant specialized in system administration."
          - "Always provide clear, step-by-step instructions."

    - model: "gpt-3.5-turbo"
      class: "openai"
      name: "GPT-3.5 Agent"
      default: false
      api-key: "${OPENAI_API_KEY}"
      api-url: "https://api.openai.com/v1"
      prompts:
        system: "You are a helpful assistant."

    - model: "llama3"
      class: "ollama"
      name: "Llama3 Local"
      default: false
      prompts:
        system:
          - "You are a helpful assistant running locally."
          - "Be concise in your responses."
```

### Model Configuration Fields

- `model`: The model identifier (e.g., "gpt-4o", "gpt-3.5-turbo")
- `class`: The model provider class ("openai", "ollama", etc.)
- `name`: A human-readable name for the model configuration
- `default`: Boolean indicating if this is the default model
- `api-key`: API key for the model provider (supports environment variable substitution)
- `api-url`: Base URL for the API endpoint
- `prompts.system`: Default system prompt for this model (can be a single string or array of strings)

### Environment Variable Substitution

API keys support environment variable substitution using the `${VARIABLE_NAME}` syntax:

```yaml
api-key: "${OPENAI_API_KEY}"
```

### Prompt Configuration

The `prompts.system` field in the configuration accepts either a single string or an array of strings:

```yaml
# Single system prompt
prompts:
  system: "You are a helpful assistant."

# Multiple system prompts (array format)
prompts:
  system:
    - "You are a helpful assistant."
    - "You specialize in system administration."
    - "Always explain your reasoning."
```

**Important:** Only system prompts are supported in the configuration file. User prompts should be provided via the `--user-prompt` command-line flag and are not stored in the configuration.

**System Prompt Merging:** When you use the `--system-prompt` command-line flag, it will be **appended** to any system prompts defined in the configuration file. This allows you to have base prompts in your config and add context-specific prompts via the command line.

## Command-Line Usage

### Using Default Model

If you have a default model configured, you can run the agent without specifying a model:

```bash
mcpshell agent --tools=examples/config.yaml \
    --user-prompt "Help me debug a performance issue"
```

### Overriding Model

You can override the default model by specifying a different one:

```bash
mcpshell agent --tools=examples/config.yaml \
    --model "gpt-3.5-turbo" \
    --user-prompt "Help me debug a performance issue"
```

### Overriding Configuration

Command-line flags take precedence over configuration file settings:

```bash
mcpshell agent --tools=examples/config.yaml \
    --model "gpt-4o" \
    --system-prompt "You are an expert system administrator" \
    --openai-api-key "your-api-key" \
    --user-prompt "Help me debug a performance issue"
```

**Note:** When you provide a `--system-prompt` via command line, it will be **merged** with any system prompts from the configuration file. The system prompts from the config are used first, followed by the command-line system prompt.

## Configuration Precedence

Settings are resolved in the following order (highest to lowest precedence):

1. Command-line flags
1. Configuration file settings
1. Environment variables
1. Default values

## Configuration Management Commands

MCPShell provides commands to manage your agent configuration:

### Create Default Configuration

```bash
mcpshell agent config create
```

Creates a default configuration file at `~/.mcpshell/agent.yaml` with sample models and settings. If the file already exists, it will be overwritten with the default configuration template.

The default configuration includes:

- GPT-4o model (set as default) with OpenAI API settings
- Gemma3n model with Ollama configuration
- Environment variable placeholders for API keys
- Basic system prompts

### Show Current Configuration

```bash
mcpshell agent config show
```

Displays the current agent configuration in a human-readable format, including:

- All configured models with their settings
- API keys (masked for security)
- Which model is set as default
- System prompts for each model

Example output:

```text
Agent Configuration:
===================

Model 1:
  Name: GPT-4o Agent
  Model: gpt-4o
  Class: openai
  Default: true
  API Key: your****-key
  API URL: https://api.openai.com/v1
  System Prompt: You are a helpful assistant.

Default Model: GPT-4o Agent (gpt-4o)
```
