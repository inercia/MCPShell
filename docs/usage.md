# MCPShell Command-Line Usage

MCPShell is a command-line interface for the MCP (Model Context Protocol) platform that enables AI systems to securely execute commands. This document describes the available commands and their arguments.

## Global Options

The following options are available for all MCPShell commands:

- `--version`, `-v`: Print version information

## Commands

MCPShell provides the following commands:

- [`mcp`](#mcp-command): Run the MCP server for a configuration file
- [`exe`](#exe-command): Execute a specific MCP tool directly
- [`validate`](#validate-command): Validate an MCP configuration file
- [`agent`](#agent-command): Execute MCPShell as an agent connected to a remote LLM

## Common arguments

This is the list of argument that are common to all the commands:

- `--config`, `-c` (required): Path to the YAML configuration file or URL
- `--logfile`, `-l`: Path to the log file (optional)
- `--log-level`: Log level: none, error, info, debug (default: "info")
- `--description-override`: override the description found in the config file.
- `--description`, `-d`: Server description (optional, can be specified multiple times).
  If an existing description is specified in the config file (and `--description-override` is not passed)
  it will append the description to the existing one. If it is specified multiple times, the
  resulting description with be the join of all of them
- `--description-file`: Read the description from files (optional, can be specified multiple times).   
  Shell globbing is supported (e.g., `--description-file *.md`). URLs are also supported (e.g., 
  `--description-file https://example.com/description.txt`). It follows the same behaviour of
  `--description`, where the final description is the result of the concatenation of all of them
  
For example, imagine you want to use the [kubectl](../examples/kubectl-ro.yaml) toolkit,
but you want to add some additional instructions specific to your infrastructure,
you could do:

```console
mcpshell mcp \
    --configfile example/kubectl-ro.yaml \
    --description "Monitoring namespace is called 'monitoring'"
    --description "Envoy is running in namespace 'envoy'"
```

So multiple descriptions can be combined in the order they are provided in the command line,
like:

```console
# Combine multiple descriptions from different sources
mcpshell mcp --config=examples/config.yaml \
  --description "Primary server description" \
  --description "Additional information" \
  --description-file docs/intro.txt \
  --description-file "docs/details.md" \
  --description-file docs/*.md
```

### MCP Command

The `mcp` command starts an MCP server that provides tools to LLM applications.

**Usage**:

```console
mcpshell mcp [flags]
```

**Aliases**: `serve`, `server`, `run`

**Description**:

Runs an MCP server that communicates using the Model Context Protocol and exposes the tools defined in a MCP configuration file. The server loads tool definitions from a YAML configuration file and makes them available to AI applications via the MCP protocol.

**Example**:

```console
mcpshell mcp --config=examples/config.yaml --log-level=debug
```

### EXE Command

The `exe` command executes a specific MCP tool directly.

**Usage**:

```console
mcpshell exe [flags] TOOL_NAME [PARAM1=VALUE1 PARAM2=VALUE2 ...]
```

**Description**:
Directly executes a MCP tool with the specified parameters. This command is useful for debugging tool execution, as it follows the whole process of constraint evaluation, tool selection, and tool execution.

**Example**:

```console
mcpshell exe --config=examples/config.yaml "hello_world" "name=John"
```

### Validate Command

The `validate` command checks an MCP configuration file for errors.

**Usage**:

```console
mcpshell validate [flags]
```

**Description**:

Validates an MCP configuration file without starting the server. It checks for errors including file format and schema validation, tool parameter definitions, constraint expression syntax, and command template syntax.

**Example**:

```console
mcpshell validate --config=examples/config.yaml
```

### Agent Command

The `agent` command executes MCPShell as an agent that connects to a remote LLM.

See [this document](usage-agent.md) for more details.

## Integration with IDEs and Tools

MCPShell can be integrated with various IDEs and tools:

- [Agent Usage Guide](usage-agent.md): Detailed guide for using MCPShell's agent functionality 
- [VSCode Integration](usage-vscode.md): Guide for integrating MCPShell with Visual Studio Code
- [Cursor Integration](usage-cursor.md): Guide for integrating MCPShell with Cursor editor

## Configuration

For information about configuring MCPShell, including defining tools, parameters, and constraints, see:

- [Configuration Guide](config.md): General configuration information
- [Runner Configuration](config-runners.md): Configuration for command runners

## Troubleshooting

For help with common issues, see the [Troubleshooting Guide](troubleshooting.md). 