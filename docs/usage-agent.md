# MCPShell Agent Mode

The MCPShell can be run in "agent mode" to establish a direct connection between Large Language Models (LLMs) and the tools you define in your configuration. This allows for autonomous execution of tasks without requiring a separate MCP client like Cursor or Visual Studio Code.

## Overview

In agent mode, MCPShell:

1. Connects directly to an LLM API
1. Makes your tools available to the LLM
1. Manages the conversation flow.
1. Handles tool execution requests
1. Provides the tool results back to the LLM

This creates a complete AI agent that can perform tasks on your system using your defined tools.

## Command-Line Options

Run the agent with:

```bash
mcpshell agent [flags]
```

### Required Flags

- `--tools`: Path to the tools configuration file (required)
- `--model`, `-m`: LLM model to use (e.g., "gpt-4o", "llama3", etc.) - can be omitted if a default model is configured in your [agent configuration](usage-agent-conf.md)

### Optional Flags

- `--logfile`, `-l`: Path to the log file
- `--log-level`: Logging level (none, error, info, debug)
- `--system-prompt`, `-s`: System prompt for the LLM (merges with system prompts from [agent configuration](usage-agent-conf.md))
- `--user-prompt`, `-u`: Initial user prompt for the LLM
- `--openai-api-key`, `-k`: OpenAI API key (or set OPENAI_API_KEY environment variable, or configure in [agent config](usage-agent-conf.md))
- `--openai-api-url`, `-b`: Base URL for the OpenAI API (for non-OpenAI services, or configure in [agent config](usage-agent-conf.md))
- `--once`, `-o`: Exit after receiving a final response (one-shot mode)

## Configuration File for Agent Mode

MCPShell has agent-specific configuration that includes model definitions with prompts. The agent configuration is separate from the tools configuration and is managed through the `mcpshell agent config` commands.

**📖 For complete details on agent configuration, including:**

- Configuration file structure and syntax
- Model configuration fields
- Environment variable substitution
- Configuration management commands
- Example configurations

**See the [Agent Configuration Guide](usage-agent-conf.md)**

## Running the Agent

With the tools degined in `disk-diagnostics-ro.yaml`, you can run:

```bash
mcpshell agent \
  --tools disk-diagnostics-ro.yaml \
  "My root partition is running low on space. Can you help me find what's taking up space and how I might free some up?"
```

The agent will:

- Load model and API settings from `~/.mcpshell/agent.yaml` (see [Agent Configuration Guide](usage-agent-conf.md)). It will look like this:

```console
$ cat ~/.mcpshell/agent.yaml
agent:
  models:
    - model: "gpt-4o"
      class: "openai"
      name: "gpt-4o"
      default: true
      api-key: "your-openai-api-key"
      api-url: "https://api.openai.com/v1"

    - name: "claude-sonnet-4"
      class: "amazon-bedrock"
      model: "us.anthropic.claude-sonnet-4-5-20250929-v1:0"
      api-url: "https://bedrock-runtime.us-east-2.amazonaws.com"

    - model: "gemma3n"
      class: "ollama"
      name: "gemma3n"

    - model: "llama3.1:8b"
      class: "ollama"
      name: "llama3"
```

- Load system prompts from the agent configuration (if any, or use teh default ones).
- Load tools from `disk-diagnostics-ro.yaml`.
- Connect to the configured LLM API.
- Process the LLM's responses and execute tool calls as requested.

## Interacting with the Agent

In interactive mode (without the `--once` flag), the agent will:

- Display the LLM's responses
- Execute tool calls as requested by the LLM
- Wait for you to provide additional input after the LLM completes its response
- Continue the conversation with the full conversation context preserved
- Loop until you exit (Ctrl+C)

In one-shot mode (with the `--once` flag), the agent will:

- Process the initial prompt
- Execute any requested tools
- Display the final response
- Exit automatically after the LLM completes

## Testing and Debugging

When developing agents, you can:

1. Enable debug logging with `--log-level debug`

1. Examine the log file for detailed information

1. Test with the `exe` command to verify individual tools:

   ```bash
   mcpshell exe --tools disk-diagnostics-ro.yaml disk_usage directory="/" max_depth=2
   ```