# Environment Variables Reference

MCPShell supports various environment variables to customize its behavior across
different modes (MCP server, exe, daemon). Environment variables provide a
flexible way to configure MCPShell without modifying configuration files or passing
command-line flags.

## Overview

Environment variables in MCPShell are used for:

- **Configuration paths**: Override default locations for config files and directories
- **System integration**: Platform-specific settings (HOME, SHELL, etc.)

**Precedence**: In most cases, environment variables have lower precedence than
command-line flags but higher precedence than default values. See individual variable
descriptions for specific precedence rules.

> **Note**: For agent-related environment variables (LLM API keys, model selection, RAG caching),
> see the [Don](https://github.com/inercia/don) project documentation.

## Configuration Paths

### `MCPSHELL_DIR`

Specifies a custom MCPShell home directory.

- **Default**: `~/.mcpshell` (Unix/Linux/macOS) or `%USERPROFILE%\.mcpshell` (Windows)
- **Used by**: All modes (mcp, agent, exe, daemon)
- **Example**:
  ```bash
  export MCPSHELL_DIR="/custom/mcpshell/dir"
  mcpshell agent --tools=tools.yaml
  ```
- **Use cases**:
  - Testing with isolated configurations
  - Multi-user environments
  - Custom deployment locations

### `MCPSHELL_TOOLS_DIR`

Specifies a custom tools directory.

- **Default**: `~/.mcpshell/tools`
- **Used by**: All modes (mcp, agent, exe, daemon)
- **Example**:
  ```bash
  export MCPSHELL_TOOLS_DIR="/custom/tools/dir"
  mcpshell mcp --tools=my-tools.yaml
  ```
- **Use cases**:
  - Shared tools directory across projects
  - Custom tools organization
  - CI/CD environments

## See Also

- [Configuration Reference](config.md) - Tools configuration reference
- [Security Guide](security.md) - Security best practices
- [Don](https://github.com/inercia/don) - For agent-related environment variables
