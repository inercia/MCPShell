# Configuration Standards for MCPShell

## YAML Structure
```yaml
mcp:
  description: "What this tool collection does"
  run:
    shell: bash
  tools:
    - name: "tool_name"
      description: "What the tool does"
      run:
        command: "echo {{ .param }}"
      params:
        param:
          type: string
          description: "Parameter description"
          required: true
```

## Required Fields
- MCP server: `description`
- Each tool: `name`, `description`, `run.command`
- Each parameter: `description`

## Tool Naming
- Lowercase with underscores: `disk_usage`, `file_reader`
- Descriptive and concise

## Parameters
- Types: `string`, `number`, `integer`, `boolean`
- Mark as `required: true` or provide `default` values
- Write detailed descriptions for LLM understanding

## Constraints
- **ALWAYS** include constraints for user input
- Add inline comments explaining each constraint
- Common patterns: command injection prevention, path traversal, length limits, whitelisting

## Templates
- Go template syntax: `{{ .param_name }}`
- Quote variables: `"{{ .param }}"`
- Supports Sprig functions

## Runners
- Order by preference (most restrictive first)
- Include fallback (usually `exec`)
- Disable networking when not needed: `allow_networking: false`
- Specify OS requirements for platform-specific runners

## Environment Variables
- **ONLY** pass explicitly whitelisted variables
- Document why each is needed

## Timeouts
- **ALWAYS** specify timeout for commands that may hang
- Format: `"30s"`, `"5m"`, `"1h30m"`

## Validation
- Use `mcpshell validate --tools <file>`
- Run `make validate-examples` in CI/CD

## Agent Mode

For AI agent functionality (LLM connectivity, RAG support), see the
[Don](https://github.com/inercia/don) project which uses MCPShell's tool configuration.
