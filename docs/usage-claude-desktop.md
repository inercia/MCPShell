# Configuration File

**Claude Desktop** from Anthropic can talk to MCPShell running in http mode on localhost.

## Prerequisites

None.

## Basic Structure

The configuration file for **Claude Desktop** is typically located in ~/.config/Claude as **claude_desktop_config.json**. Below are the basic elements required in the file:

```
{
  "mcpServers": {
    "mcpshell-http": {
      "command": "npx",
      "args": ["-y", "mcp-remote", "http://localhost:3333/sse"]
    }
  }
}

```
