# MCPShell Agent Mode

The MCPShell can be run in "agent mode" to establish a direct connection between Large Language Models (LLMs) and the tools you define in your configuration. This allows for autonomous execution of tasks without requiring a separate MCP client like Cursor or Visual Studio Code.

## Overview

In agent mode, MCPShell:

1. Connects directly to an LLM API (currently OpenAI-compatible APIs)
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

## Creating an Agent Script

You can create a shell script to run MCPShell in agent mode with a specific configuration. This is useful for creating specialized agents for different tasks.

Here's a practical example showing both agent configuration and tools configuration for a disk space analyzer agent:

**Tools Configuration** (`disk-analyzer.yaml`):

```yaml
  description: "Tools for analyzing disk space usage and system performance"
  run:
    shell: "bash"
  tools:
    - name: "disk_usage"
      description: "Check disk usage for a directory"
      params:
        directory:
          type: string
          description: "Directory to analyze"
          required: true
        max_depth:
          type: number
          description: "Maximum depth to analyze (1-3)"
          default: 2
      constraints:
        - "directory.startsWith('/')"  # Must be absolute path
        - "!directory.contains('..')"  # Prevent directory traversal
        - "max_depth >= 1 && max_depth <= 3"  # Limit recursion depth
      run:
        command: |
          du -h --max-depth={{ .max_depth }} {{ .directory }} | sort -hr | head -20
      output:
        format: "text"
        prefix: "Disk Usage Analysis (Top 20 largest directories):"

    - name: "filesystem_info"
      description: "Show filesystem usage information"
      run:
        command: |
          df -h
      output:
        format: "text"
        prefix: "Filesystem Usage Information:"

    - name: "large_files"
      description: "Find large files in a directory"
      params:
        directory:
          type: string
          description: "Directory to search in"
          required: true
        min_size:
          type: string
          description: "Minimum file size (e.g., '100M', '1G')"
          default: "100M"
        file_type:
          type: string
          description: "Filter by file extension (e.g., 'log', 'zip')"
          required: false
      constraints:
        - "directory.startsWith('/')"  # Must be absolute path
        - "!directory.contains('..')"  # Prevent directory traversal
      run:
        command: |
          file_type="{{ .file_type }}"
          if [ -n "$file_type" ]; then
            find {{ .directory }} -type f -name "*.${file_type}" -size +{{ .min_size }} -exec ls -lh {} \; | sort -k5hr | head -20
          else
            find {{ .directory }} -type f -size +{{ .min_size }} -exec ls -lh {} \; | sort -k5hr | head -20
          fi
      output:
        format: "text"
        prefix: "Large Files (minimum size {{ .min_size }}):"
```

## Running the Agent

With the tools configuration saved as `disk-analyzer.yaml`, you can run:

```bash
mcpshell agent \
  --tools disk-analyzer.yaml \
  --user-prompt "My root partition is running low on space. Can you help me find what's taking up space and how I might free some up?"
```

If you have a default model configured in your agent config file,
you don't need to specify the model or API key:

```bash
mcpshell agent --tools disk-analyzer.yaml
```

The agent will:

1. Load model and API settings from `~/.mcpshell/agent.yaml` (see [Agent Configuration Guide](usage-agent-conf.md))
1. Load system prompts from the agent configuration
1. Load tools from `disk-analyzer.yaml`
1. Connect to the configured LLM API
1. Process the LLM's responses and execute tool calls as requested

## Interacting with the Agent

In interactive mode (without the `--once` flag), the agent will:

- Display the LLM's responses
- Execute tool calls as requested by the LLM
- Prompt you for additional input
- Continue the conversation until you exit (Ctrl+C) or the LLM responds with "TERMINATE"

In one-shot mode (with the `--once` flag), the agent will:

- Process the initial prompt
- Execute any requested tools
- Display the final response
- Exit automatically

## Testing and Debugging

When developing agents, you can:

1. Enable debug logging with `--log-level debug`

1. Examine the log file for detailed information

1. Test with the `exe` command to verify individual tools:

   ```bash
   mcpshell exe --tools disk-analyzer.yaml disk_usage directory="/" max_depth=2
   ```

## Conclusion

The agent mode provides a powerful way to create specialized AI assistants that can perform specific tasks on your system using the tools you define. By combining well-defined tools with appropriate system and user prompts, you can create agents that solve real-world problems in a secure and controlled manner.
