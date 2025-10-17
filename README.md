# MCPShell

<p align="center">
  <img src="docs/logo.png" alt="banner" width="300"/>
</p>

The **MCPShell** is a tool that allows LLMs to safely execute **command-line tools**
through the [**Model Context Protocol (MCP)**](https://modelcontextprotocol.io/).
It provides a secure bridge between LLMs and operating system commands.

## Features

- **Flexible command execution**: Run any shell commands as MCP tools,
  with parameter substitution through templates.
- **Configuration-based tool definitions**: Define tools in YAML with parameters,
  constraints, and output formatting.
- **Security through constraints**: Validate tool parameters using CEL expressions
  before execution, as well as optional [**sanboxed environments**](docs/config-runners.md)
  for running commands.
- **Quick proptotyping of MCP tools**: just add some shell code and use it as
  a MCP tool in your LLM.
- **Simple integration**: Works with any LLM client supporting the MCP protocol
  (ie, Cursor, VSCode, Witsy...)

## Quick Start

Imagine you want Cursor (or some other MCP client) help you with your
space problems in your hard disk.

1. Create a configuration file `/my/example.yaml` defining your tools:

   ```yaml
   mcp:
     description: |
       Tool for analyzing disk usage to help identify what's consuming space.
     run:
       shell: bash
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
           - "directory.matches('^[\\w\\s./\\-_]+$')"  # Only allow safe path characters, prevent command injection
         run:
           command: |
             du -h --max-depth={{ .max_depth }} {{ .directory }} | sort -hr | head -20
         output:
           prefix: |
             Disk Usage Analysis (Top 20 largest directories):
   ```

   Take a look at the [examples directory](examples) for more sophisticated and useful examples.
   Maybe you prefer to let the LLM know about your Kubernetes cluster with
   [kubectl](examples/kubectl-ro.yaml)?
   Or let it run some [AWS CLI](examples/aws-networking-ro.yaml) commands?

1. Configure the MCP server in Cursor (or in any other LLM client with support for MCP)

   For example, for Cursor, create `.cursor/mcp.json`:

   ```json
   {
       // you need the "go" command available
       "mcpServers": {
           "mcp-cli-examples": {
               "command": "go",
               "args": [
                  "run", "github.com/inercia/MCPShell@v0.1.5",
                  "mcp", "--tools", "/my/example.yaml",
                  "--logfile", "/some/path/mcpshell/example.log"
               ]
           }
       }
   }
   ```

   You can also use relative paths and omit the `.yaml` extension:

   ```json
   {
       "mcpServers": {
           "mcp-cli-examples": {
               "command": "go",
               "args": [
                  "run", "github.com/inercia/MCPShell@v0.1.5",
                  "mcp", "--tools", "example",
                  "--logfile", "/some/path/mcpshell/example.log"
               ]
           }
       }
   }
   ```

   This will look for `example.yaml` in the tools directory (`~/.mcpshell/tools/` by default).

   See more details on how to configure [Cursor](docs/usage-cursor.md) or
   [Visual Studio Code](docs/usage-vscode.md). Other LLMs with support for MCPs
   should be configured in a similar way.

1. Make sure your MCP client is refreshed (Cursor should recognize it automatically the
   firt time, but any change in the config file will require a refresh).

1. Ask your LLM some questions it should be able to answer with the new tool. For example:
   _"I'm running out of space in my hard disk. Could you help me finding the problem?"_.

## Usage and Configuration

Take a look at all the command in [this document](docs/usage.md).

Configuration files use a YAML format defined [here](docs/config.md).
See the [this directory](examples) for some examples.

## Agent Mode

MCPShell can also be run in agent mode, providing direct connectivity between Large Language Models
(LLMs) and your command-line tools without requiring a separate MCP client. In this mode,
MCPShell connects to an OpenAI-compatible API (including local LLMs like Ollama), makes your
tools available to the model, executes requested tool operations, and manages the conversation flow.
This enables the creation of specialized AI assistants that can autonomously perform system tasks
using the tools you define in your configuration. The agent mode supports both interactive
conversations and one-shot executions, and allows you to define system and user prompts directly
in your configuration files.

For detailed information on using agent mode, see the [Agent Mode documentation](docs/usage-agent.md).

## Security Considerations

So you will probably thing
_"this AI has helped me finding all those big files. What if I create another tool for removing files?"_.
**Don't do that!**.

- Limit the scope of these tools to **read-only actions**, do not give the LLM the power to change things.
- Use **constraints** to limit command execution to safe parameters
- Consider using a [**sanboxed environment**](docs/config-runners.md) for running commands.
- Review all command templates for potential injection vulnerabilities
- Only expose tools that are safe for external use
- All of the above!

Please read the [Security Considerations](docs/security.md) document before using this software.

## Contributing

Contributions are welcome! Take a look at the [development guide](docs/development.md).
Please open an issue or submit a pull request on GitHub.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
