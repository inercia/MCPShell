---
type: "agent_requested"
description: "MCP: What is MCP? Protocol Communication, Server Implementation, Tool Registration, Request Handling"
---

# MCP Protocol Implementation Guidelines for MCPShell

## MCP Protocol Overview

### What is MCP?
- Model Context Protocol (MCP) is a standard protocol for connecting LLMs to external tools and data sources
- MCPShell implements the MCP server side, exposing command-line tools as MCP tools
- MCP clients (Cursor, VSCode, Claude Desktop) connect to MCPShell to access these tools

### Protocol Communication
- MCPShell uses the `github.com/mark3labs/mcp-go` library for MCP protocol implementation
- Supports stdio transport (standard input/output) for communication
- JSON-RPC 2.0 message format for requests and responses

## Server Implementation

### Server Lifecycle
1. **Initialization**: Load configuration and create server instance
2. **Tool Registration**: Register all tools from configuration
3. **Server Start**: Begin listening for MCP requests
4. **Request Processing**: Handle tool calls and return results
5. **Shutdown**: Clean up resources and close connections

### Server Creation
- Use `server.New()` to create a server instance with configuration
- Call `CreateServer()` to initialize the MCP server and register tools
- Call `Start()` to begin processing requests
- Example:
  ```go
  srv := server.New(server.Config{
      ConfigFile: configPath,
      Logger:     logger,
      Version:    version,
  })
  
  if err := srv.CreateServer(); err != nil {
      return fmt.Errorf("failed to create server: %w", err)
  }
  
  if err := srv.Start(); err != nil {
      return fmt.Errorf("failed to start server: %w", err)
  }
  ```

## Tool Registration

### Tool Definition
- Each tool is defined in YAML configuration
- Tools are converted to MCP tool format during registration
- Tool schema is generated from parameter definitions
- Example MCP tool structure:
  ```go
  mcp.Tool{
      Name:        "tool_name",
      Description: "Tool description",
      InputSchema: mcp.ToolInputSchema{
          Type: "object",
          Properties: map[string]interface{}{
              "param_name": map[string]interface{}{
                  "type":        "string",
                  "description": "Parameter description",
              },
          },
          Required: []string{"param_name"},
      },
  }
  ```

### Handler Registration
- Each tool has an associated handler function
- Handlers implement the `mcpserver.ToolHandlerFunc` signature
- Handlers are wrapped with panic recovery
- Example:
  ```go
  type ToolHandlerFunc func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
  ```

### Tool Validation
- Tools are validated during registration
- Constraint compilation happens at registration time
- Invalid tools are rejected with clear error messages
- Prerequisites (OS, executables) are checked before registration

## Request Handling

### Request Flow
1. MCP client sends `tools/call` request
2. Server routes request to appropriate tool handler
3. Handler validates parameters and constraints
4. Handler executes command via runner
5. Handler formats output and returns result
6. Server sends response back to client

### Parameter Handling
- Parameters are extracted from `request.Params.Arguments`
- Type assertions are performed to ensure correct types
- Default values are applied for optional parameters
- Parameters are validated against constraints before execution

### Error Handling
- Errors are returned as `mcp.CallToolResult` with error content
- Use `mcp.NewToolResultError()` for error results
- Use `mcp.NewToolResultText()` for success results
- Example:
  ```go
  if err != nil {
      return mcp.NewToolResultError(err.Error()), nil
  }
  return mcp.NewToolResultText(output), nil
  ```

## Tool Execution

### Execution Flow
1. Extract and validate parameters
2. Apply default values for optional parameters
3. Evaluate constraints
4. Render command template with parameters
5. Select and configure runner
6. Execute command via runner
7. Format output with prefix if configured
8. Return result to client

### Constraint Evaluation
- Constraints are evaluated before command execution
- All constraints must pass for execution to proceed
- Failed constraints block execution and return error
- Constraint failures are logged with details

### Command Execution
- Commands are executed via runner implementations
- Runners provide isolation and security
- Timeouts are enforced via context
- Output is captured and returned to client

## Response Formatting

### Success Responses
- Return command output as text content
- Apply output prefix if configured
- Trim whitespace from output
- Example:
  ```go
  return mcp.NewToolResultText(output), nil
  ```

### Error Responses
- Return error message as error content
- Include context about what failed
- Don't leak sensitive information in errors
- Example:
  ```go
  return mcp.NewToolResultError("command execution failed: invalid parameter"), nil
  ```

## Agent Mode Integration

For AI agent functionality that uses MCPShell tools, see the
[Don](https://github.com/inercia/don) project. Don spawns MCPShell as a
subprocess to execute MCP tools while handling LLM connectivity and
conversation management.

## Protocol Extensions

### Custom Descriptions
- Support for custom server descriptions via flags
- Descriptions can be loaded from files or URLs
- Multiple descriptions can be concatenated
- Descriptions help LLMs understand tool capabilities

### Prompts Configuration
- Support for custom prompts in configuration
- Prompts provide additional context to LLMs
- Prompts are exposed via MCP protocol
- Example:
  ```yaml
  prompts:
    - name: "example_prompt"
      description: "Example prompt description"
      arguments:
        - name: "arg1"
          description: "Argument description"
          required: true
  ```

## Best Practices

### Tool Design
- Keep tools focused on single tasks
- Provide clear, descriptive tool names
- Write comprehensive tool descriptions
- Include examples in descriptions when helpful

### Parameter Design
- Use descriptive parameter names
- Provide detailed parameter descriptions
- Set appropriate default values
- Mark required parameters explicitly

### Error Messages
- Provide actionable error messages
- Include context about what failed
- Suggest how to fix the problem
- Don't leak sensitive information

### Performance
- Compile constraints at registration time
- Parse templates once during handler creation
- Use context for timeouts and cancellation
- Clean up resources properly

## Testing MCP Integration

### Unit Testing
- Test tool registration logic
- Test parameter extraction and validation
- Test constraint evaluation
- Test error handling

### Integration Testing
- Test full request/response cycle
- Test with actual MCP clients when possible
- Test error scenarios
- Test timeout handling

### Manual Testing
- Use `mcpshell exe` command for direct tool testing
- Test with MCP clients (Cursor, VSCode)
- Verify tool descriptions are clear
- Test with various parameter combinations

## Debugging MCP Issues

### Logging
- Enable debug logging with `--log-level debug`
- Log all tool registrations
- Log all tool executions
- Log constraint evaluations

### Common Issues
- **Tool not appearing in client**: Check tool registration logs
- **Parameter validation failing**: Check constraint definitions
- **Command execution failing**: Check runner configuration
- **Timeout errors**: Adjust timeout values in configuration

### Troubleshooting Steps
1. Check server logs for errors
2. Verify configuration file syntax
3. Test tool directly with `mcpshell exe`
4. Verify MCP client configuration
5. Check network connectivity (if using HTTP transport)
