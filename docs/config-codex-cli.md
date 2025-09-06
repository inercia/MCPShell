# Configuration File

The **Codex CLI** from OpenAI can talk to MCPShell running in http mode on localhost. By design, **Codex CLI** wants to speak MCP over stdio (which is a mode fully supported by MCPShell).  However, if you want to use MCPShell in a centralized manner (as a server), you will need to install mcp-proxy.

## Prerequisites

A working install of **mcp-proxy**.  On many systems this can be installed using **npm**.

## Basic Structure

The configuration file for codex is typically located in ~/.codex as **config.toml**.  Below are the basic elements required in the file:

```
# (Optional) general preferences
network_access = true
model_reasoning_effort = "medium"

# MCP servers
[mcp_servers.sse_local]
# Use the proxy as a stdio server that connects to your SSE endpoint
command = "mcp-proxy"
args = ["--transport", "streamablehttp", "http://localhost:3333/sse"]
# If you installed via uv and the command isn't on PATH, use the full path, e.g.:
# command = "/home/<you>/.local/bin/mcp-proxy"
# You can also pass headers if your SSE server needs auth:
# env = { API_ACCESS_TOKEN = "your-token" }
```

