# MCPShell Documentation

This directory contains comprehensive documentation for MCPShell.

## Getting Started

- [Usage Guide](usage.md) - Command-line usage and basic concepts
- [Configuration](config.md) - YAML configuration file format
- [Security Considerations](security.md) - Security best practices and guidelines

## Usage Guides

### MCP Client Integration

- [Cursor Integration](usage-cursor.md) - Using MCPShell with Cursor IDE
- [VS Code Integration](usage-vscode.md) - Using MCPShell with Visual Studio Code
- [Claude Desktop Integration](usage-claude-desktop.md) - Using MCPShell with Claude
  Desktop
- [Codex CLI Integration](usage-codex-cli.md) - Using MCPShell with Codex CLI

### Agent Mode

For AI agent functionality (direct LLM connectivity, RAG support), see the
[Don](https://github.com/inercia/don) project which uses MCPShell's tool configuration.

### Deployment

- [Container Deployment](usage-containers.md) - Deploying MCPShell in containers and
  Kubernetes

## Configuration

- [Configuration Reference](config.md) - Complete YAML configuration format
- [Environment Variables](config-env.md) - Environment variables reference for all modes
- [Runners Configuration](config-runners.md) - Sandboxed execution environments
  (firejail, sandbox-exec, docker)

## Development

- [Development Guide](development.md) - Setting up development environment and
  contributing
- [Release Process](release-process.md) - How releases are created and published
- [Troubleshooting](troubleshooting.md) - Common issues and solutions

## Quick Links

### For Users

- **First time?** Start with [Usage Guide](usage.md)
- **Setting up tools?** See [Configuration](config.md)
- **Security concerns?** Read [Security Considerations](security.md)
- **Using with Cursor?** Check [Cursor Integration](usage-cursor.md)
- **Want agent mode?** See [Don](https://github.com/inercia/don)

### For Developers

- **Contributing?** Read [Development Guide](development.md)
- **Releasing?** Follow [Release Process](release-process.md)

## Documentation Structure

```text
docs/
├── README.md                    # This file - documentation index
├── usage.md                     # Main usage guide
├── config.md                    # Configuration reference
├── config-env.md                # Environment variables reference
├── config-runners.md            # Runners configuration
├── security.md                  # Security guidelines
├── troubleshooting.md           # Troubleshooting guide
├── usage-cursor.md              # Cursor integration
├── usage-vscode.md              # VS Code integration
├── usage-claude-desktop.md      # Claude Desktop integration
├── usage-codex-cli.md           # Codex CLI integration
├── usage-containers.md          # Container deployment
├── development.md               # Development guide
└── release-process.md           # Release process
```

## External Resources

- [GitHub Repository](https://github.com/inercia/MCPShell)
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP specification
- [cagent Library](https://github.com/docker/cagent) - Agent framework used by MCPShell
