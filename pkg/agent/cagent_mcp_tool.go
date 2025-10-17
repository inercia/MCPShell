// Package agent provides cagent integration for MCP tools
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	cagentTools "github.com/docker/cagent/pkg/tools"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/server"
)

// MCPToolSet wraps MCP server tools for use with cagent
type MCPToolSet struct {
	server *server.Server
	logger *common.Logger
}

// NewMCPToolSet creates a new MCP tool set for cagent
func NewMCPToolSet(srv *server.Server, logger *common.Logger) *MCPToolSet {
	return &MCPToolSet{
		server: srv,
		logger: logger,
	}
}

// GetTools returns all MCP tools as cagent-compatible tools
func (m *MCPToolSet) GetTools() ([]cagentTools.Tool, error) {
	// Get MCP tools from the server
	mcpTools, err := m.server.GetTools()
	if err != nil {
		m.logger.Error("Failed to get MCP tools: %v", err)
		return nil, fmt.Errorf("failed to get MCP tools: %w", err)
	}

	// Convert each MCP tool to a cagent tool
	tools := make([]cagentTools.Tool, 0, len(mcpTools))
	for _, mcpTool := range mcpTools {
		tool := m.convertMCPToolToCagent(mcpTool)
		tools = append(tools, tool)
	}

	m.logger.Info("Wrapped %d MCP tools for cagent", len(tools))
	return tools, nil
}

// convertMCPToolToCagent converts an MCP tool to a cagent Tool struct
func (m *MCPToolSet) convertMCPToolToCagent(mcpTool mcp.Tool) cagentTools.Tool {
	// Convert MCP input schema to JSON schema for cagent
	schemaMap := map[string]interface{}{
		"type":       "object",
		"properties": mcpTool.InputSchema.Properties,
		"required":   mcpTool.InputSchema.Required,
	}

	// Create the handler function that executes the MCP tool
	// ToolHandler signature: func(ctx context.Context, toolCall ToolCall) (*ToolCallResult, error)
	handler := func(ctx context.Context, toolCall cagentTools.ToolCall) (*cagentTools.ToolCallResult, error) {
		// Parse the arguments from JSON string
		var args map[string]interface{}

		// Handle empty arguments (for tools with all optional parameters)
		if toolCall.Function.Arguments == "" {
			args = make(map[string]interface{})
			m.logger.Debug("Tool '%s' called with no arguments, using empty map", mcpTool.Name)
		} else {
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				m.logger.Error("Failed to parse tool arguments for '%s': %v (raw: '%s')",
					mcpTool.Name, err, toolCall.Function.Arguments)

				// Return a helpful error message to the agent
				return &cagentTools.ToolCallResult{
					Output: fmt.Sprintf("Error: Invalid JSON arguments provided. Expected valid JSON object but got: %s\n\nExample valid call: {}",
						toolCall.Function.Arguments),
				}, nil
			}
		}

		m.logger.Info("Executing MCP tool '%s' via cagent with args: %+v", mcpTool.Name, args)

		// Execute the tool through the MCP server
		result, err := m.server.ExecuteTool(ctx, mcpTool.Name, args)
		if err != nil {
			m.logger.Error("Failed to execute MCP tool '%s': %v", mcpTool.Name, err)

			// Return error as output instead of returning error, so agent can see it and retry
			return &cagentTools.ToolCallResult{
				Output: fmt.Sprintf("Error executing tool: %v", err),
			}, nil
		}

		m.logger.Debug("MCP tool '%s' result: %s", mcpTool.Name, result)
		return &cagentTools.ToolCallResult{
			Output: result,
		}, nil
	}

	// Marshal schema to JSON for Parameters field
	schemaJSON, err := json.Marshal(schemaMap)
	if err != nil {
		m.logger.Error("Failed to marshal tool parameters for '%s': %v", mcpTool.Name, err)
		// Return minimal valid schema on error
		schemaJSON = []byte(`{"type":"object","properties":{}}`)
	}

	return cagentTools.Tool{
		Name:        mcpTool.Name,
		Description: mcpTool.Description,
		Parameters:  json.RawMessage(schemaJSON),
		Handler:     handler,
	}
}
