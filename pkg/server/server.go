// Package server implements the MCP server functionality.
//
// It handles loading tool configurations, starting the server,
// and processing requests from AI clients using the MCP protocol.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/sashabaranov/go-openai"

	"github.com/inercia/MCPShell/pkg/command"
	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
)

// Server represents the MCPShell server that handles tool registration
// and request processing.
type Server struct {
	configFile  string
	shell       string
	version     string
	description string

	mcpServer *mcpserver.MCPServer // MCP server instance

	logger *common.Logger
}

// Config contains the configuration options for creating a new Server
type Config struct {
	ConfigFile          string         // Path to the YAML configuration file
	Shell               string         // Shell to use for executing commands
	Logger              *common.Logger // Logger for server operations
	Version             string         // Version string for the server
	Descriptions        []string       // Descriptions shown to AI clients (can be specified multiple times)
	DescriptionFiles    []string       // Paths to files containing descriptions (can be specified multiple times)
	DescriptionOverride bool           // Whether to override the description in the config file
}

// New creates a new Server instance with the provided configuration
//
// Parameters:
//   - cfg: The server configuration
//
// Returns:
//   - A new Server instance
func New(cfg Config) *Server {
	// Process description based on input flags
	finalDescription, err := GetDescription(cfg)
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Error("Failed to process description flags: %v", err)
		}
	} else if finalDescription != "" {
		if cfg.Logger != nil {
			cfg.Logger.Debug("Using MCP server description: %s", finalDescription)
		}
	} else {
		if cfg.Logger != nil {
			cfg.Logger.Debug("No MCP server description provided")
		}
	}

	return &Server{
		configFile:  cfg.ConfigFile,
		shell:       cfg.Shell,
		logger:      cfg.Logger,
		version:     cfg.Version,
		description: finalDescription,
	}
}

// Validate verifies the configuration file without starting the server.
// It loads the configuration, attempts to compile all constraints, and checks for errors.
//
// Returns:
//   - nil if the configuration is valid
//   - An error describing validation failures
func (s *Server) Validate() error {
	s.logger.Info("Validating configuration file: %s", s.configFile)

	// Load configuration
	cfg, err := config.NewConfigFromFile(s.configFile)
	if err != nil {
		s.logger.Error("Failed to load config: %v", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if there are any tools defined
	if len(cfg.MCP.Tools) == 0 {
		s.logger.Error("No tools defined in the configuration file")
		return fmt.Errorf("no tools defined in the configuration file")
	}

	s.logger.Info("Found %d tools in configuration", len(cfg.MCP.Tools))

	// Use shell from config if present and no shell is explicitly set
	shell := s.shell
	if shell == "" && cfg.MCP.Run.Shell != "" {
		s.logger.Debug("Using shell from config: %s", cfg.MCP.Run.Shell)
	}

	// Get filtered tool definitions based on prerequisites
	toolDefs := cfg.GetTools()

	// Check if some tools were filtered out due to prerequisites not met
	if len(toolDefs) < len(cfg.MCP.Tools) {
		skippedCount := len(cfg.MCP.Tools) - len(toolDefs)
		s.logger.Info("%d tool(s) would be skipped due to unmet prerequisites", skippedCount)

		// Log which tools were skipped
		for _, toolConfig := range cfg.MCP.Tools {
			found := false
			for _, toolDef := range toolDefs {
				if toolDef.MCPTool.Name == toolConfig.Name {
					found = true
					break
				}
			}

			if !found {
				s.logger.Info("Tool '%s' would be skipped due to unmet prerequisites", toolConfig.Name)
			}
		}
	}

	s.logger.Info("Validating %d tools after checking prerequisites", len(toolDefs))

	// Validate each tool definition
	for _, toolDef := range toolDefs {
		s.logger.Debug("Validating tool '%s'", toolDef.MCPTool.Name)

		// Find the original tool config
		toolIndex := s.findToolByName(cfg.MCP.Tools, toolDef.MCPTool.Name)
		if toolIndex == -1 {
			return fmt.Errorf("internal error: tool '%s' not found in configuration after creation", toolDef.MCPTool.Name)
		}

		// Get parameter types for constraint validation
		paramTypes := cfg.MCP.Tools[toolIndex].Params

		// Validate constraints by attempting to compile them
		if len(toolDef.Config.Constraints) > 0 {
			s.logger.Debug("Compiling %d constraints for tool '%s'", len(toolDef.Config.Constraints), toolDef.MCPTool.Name)
			_, err := common.NewCompiledConstraints(toolDef.Config.Constraints, paramTypes, s.logger)
			if err != nil {
				s.logger.Error("Failed to compile constraints for tool '%s': %v", toolDef.MCPTool.Name, err)
				return fmt.Errorf("constraint compilation error for tool '%s': %w", toolDef.MCPTool.Name, err)
			}
			s.logger.Debug("All constraints for tool '%s' compiled successfully", toolDef.MCPTool.Name)
		}

		// Validate command template
		if toolDef.Config.Run.Command == "" {
			s.logger.Error("Empty command template for tool '%s'", toolDef.MCPTool.Name)
			return fmt.Errorf("empty command template for tool '%s'", toolDef.MCPTool.Name)
		}

		// Format constraint information for display
		var constraintInfo string
		if len(toolDef.Config.Constraints) > 0 {
			constraintInfo = fmt.Sprintf(" (with %d constraints)", len(toolDef.Config.Constraints))
		} else {
			constraintInfo = ""
		}

		s.logger.Info("Validated tool: '%s'%s", toolDef.MCPTool.Name, constraintInfo)
	}

	s.logger.Info("Configuration validation successful")
	return nil
}

// Start initializes the MCP server, loads tools from the configuration file,
// and starts listening for client connections.
//
// Returns:
//   - An error if server initialization or startup fails
func (s *Server) Start() error {
	s.logger.Info("Initializing MCP server")

	// Create and configure MCP server
	if err := s.CreateServer(); err != nil {
		return err
	}

	s.logger.Info("Starting MCP server with stdio handler")

	// Start the stdio server
	if err := mcpserver.ServeStdio(s.mcpServer); err != nil {
		s.logger.Error("Server error: %v", err)
		return fmt.Errorf("server error: %v", err)
	}

	return nil
}

// CreateServer initializes the MCP server instance
func (s *Server) CreateServer() error {
	// First create the MCP server
	serverName := "MCPShell"
	var options []mcpserver.ServerOption

	// Load server configuration for description, shell, etc.
	cfg, err := config.NewConfigFromFile(s.configFile)
	if err != nil {
		s.logger.Error("Failed to load config: %v", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Use shell from config if present and no shell is explicitly set
	if s.shell == "" && cfg.MCP.Run.Shell != "" {
		s.shell = cfg.MCP.Run.Shell
		s.logger.Debug("Using shell from config: %s", s.shell)
	}

	// Add description if provided
	if s.description != "" {
		s.logger.Debug("Using description for MCP server: %s", s.description)
		options = append(options, mcpserver.WithInstructions(s.description))
	}

	// Initialize the MCP server BEFORE loading tools
	s.mcpServer = mcpserver.NewMCPServer(serverName, s.version, options...)

	// Now load tools after the server is initialized
	if err := s.loadTools(cfg); err != nil {
		s.logger.Error("Failed to load tools: %v", err)
		return err
	}

	return nil
}

// loadTools loads tools from the configuration and registers them with the server
func (s *Server) loadTools(cfg *config.ToolsConfig) error {
	// Check if there are any tools defined
	if len(cfg.MCP.Tools) == 0 {
		s.logger.Error("No tools defined in the configuration file")
		return fmt.Errorf("no tools defined in the configuration file")
	}

	s.logger.Info("Found %d tools in configuration", len(cfg.MCP.Tools))

	// Create and register tools
	toolDefs := cfg.GetTools()

	// Check if some tools were filtered out due to prerequisites not met
	if len(toolDefs) < len(cfg.MCP.Tools) {
		skippedCount := len(cfg.MCP.Tools) - len(toolDefs)
		s.logger.Info("Skipped %d tool(s) due to unmet prerequisites", skippedCount)

		// Log which tools were skipped
		for _, toolConfig := range cfg.MCP.Tools {
			found := false
			for _, toolDef := range toolDefs {
				if toolDef.MCPTool.Name == toolConfig.Name {
					found = true
					break
				}
			}

			if !found {
				s.logger.Info("Tool '%s' was skipped due to unmet prerequisites", toolConfig.Name)
			}
		}
	}

	s.logger.Info("Registering %d tools after checking prerequisites", len(toolDefs))

	for _, toolDef := range toolDefs {
		s.logger.Debug("Registering tool '%s'", toolDef.MCPTool.Name)

		// Get the parameter types for this tool
		params := cfg.MCP.Tools[s.findToolByName(cfg.MCP.Tools, toolDef.MCPTool.Name)].Params

		// Create a new command handler instance
		cmdHandler, err := command.NewCommandHandler(toolDef, params, s.shell, s.logger)
		if err != nil {
			s.logger.Error("Failed to create handler for tool '%s': %v", toolDef.MCPTool.Name, err)
			return fmt.Errorf("failed to create handler for tool '%s': %w", toolDef.MCPTool.Name, err)
		}

		// Get the MCP handler and wrap it with panic recovery
		safeHandler := s.wrapHandlerWithPanicRecovery(cmdHandler.GetMCPHandler())

		// Add the tool to the server
		s.mcpServer.AddTool(toolDef.MCPTool, safeHandler)

		// Print whether constraints are enabled
		if len(toolDef.Config.Constraints) > 0 {
			msg := fmt.Sprintf("Registered tool: '%s' (with %d constraints)", toolDef.MCPTool.Name, len(toolDef.Config.Constraints))
			s.logger.Info(msg)
		} else {
			msg := fmt.Sprintf("Registered tool: '%s'", toolDef.MCPTool.Name)
			s.logger.Info(msg)
		}
	}

	return nil
}

// wrapHandlerWithPanicRecovery adds panic recovery to a tool handler
func (s *Server) wrapHandlerWithPanicRecovery(handler mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
		// Set up panic recovery
		defer func() {
			if r := recover(); r != nil {
				// Use the common panic recovery logic but don't exit
				common.RecoverPanic()

				// Return an error instead of crashing
				err = fmt.Errorf("tool execution failed: internal server error")
			}
		}()

		// Call the original handler
		return handler(ctx, request)
	}
}

// findToolByName finds a tool configuration by name
func (s *Server) findToolByName(tools []config.MCPToolConfig, name string) int {
	s.logger.Debug("Looking for tool with name '%s'", name)

	for i, tool := range tools {
		if tool.Name == name {
			s.logger.Debug("Found tool '%s' at index %d", name, i)
			return i
		}
	}

	s.logger.Debug("Tool '%s' not found", name)
	return -1
}

// GetTools returns all available MCP tools from the server
// Used by the agent to get tools for the LLM
func (s *Server) GetTools() ([]mcp.Tool, error) {
	// Ensure the server is initialized
	if s.mcpServer == nil {
		return nil, fmt.Errorf("server not initialized")
	}

	// Create a slice to store the tools
	// Since we don't have direct access to all tools, we'll need to extract them
	// from the original configuration
	cfg, err := config.NewConfigFromFile(s.configFile)
	if err != nil {
		s.logger.Error("Failed to load config: %v", err)
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	toolDefs := cfg.GetTools()
	tools := make([]mcp.Tool, 0, len(toolDefs))

	for _, toolDef := range toolDefs {
		tools = append(tools, toolDef.MCPTool)
	}

	return tools, nil
}

// convertMCPToolsToOpenAI converts MCP tools to OpenAI tool format
func (s *Server) GetOpenAITools() ([]openai.Tool, error) {
	mcpTools, err := s.GetTools()
	if err != nil {
		return nil, err
	}

	openaiTools := make([]openai.Tool, 0, len(mcpTools))

	for _, tool := range mcpTools {
		// Create schema map for parameters
		schemaMap := map[string]interface{}{
			"type":       "object",
			"properties": make(map[string]interface{}),
			"required":   []string{},
		}

		// Get properties from the MCP tool
		props := tool.InputSchema.Properties
		propMap := schemaMap["properties"].(map[string]interface{})

		// Convert all properties
		for name, propInterface := range props {
			// Default property structure
			prop := map[string]interface{}{
				"type":        "string",
				"description": "",
			}

			// Try to extract type and description from the property
			if propMap, ok := propInterface.(map[string]interface{}); ok {
				if propType, exists := propMap["type"]; exists {
					prop["type"] = propType
				}
				if propDesc, exists := propMap["description"]; exists {
					prop["description"] = propDesc
				}
			}

			// Add the property to our schema
			propMap[name] = prop
		}

		// Add required properties
		if len(tool.InputSchema.Required) > 0 {
			schemaMap["required"] = tool.InputSchema.Required
		}

		// Create the OpenAI tool
		openaiTool := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  schemaMap,
			},
		}

		openaiTools = append(openaiTools, openaiTool)
	}

	return openaiTools, nil
}

// ExecuteTool executes a specific tool with the given parameters
// Used by the agent to execute tools requested by the LLM
func (s *Server) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	// Ensure the server is initialized
	if s.mcpServer == nil {
		return "", fmt.Errorf("server not initialized")
	}

	// Log the arguments being passed to help debug
	s.logger.Debug("Executing tool '%s' with arguments: %+v", toolName, args)

	// Create a properly formatted JSON-RPC request manually
	jsonRpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	// Debug output to trace the exact JSON being sent
	jsonBytes, _ := json.MarshalIndent(jsonRpcRequest, "", "  ")
	s.logger.Debug("Sending JSON-RPC request: %s", string(jsonBytes))

	// Execute the tool through the MCP server
	s.logger.Debug("Executing tool: %s", toolName)

	// We need to handle the request manually since we don't have direct access to tool handlers
	jsonMsg := s.mcpServer.HandleMessage(ctx, mustMarshalJSON(jsonRpcRequest))

	// Convert the response to JSON bytes - handle different possible types
	var responseBytes []byte
	switch msg := jsonMsg.(type) {
	case json.RawMessage:
		responseBytes = []byte(msg)
	case string:
		responseBytes = []byte(msg)
	case []byte:
		responseBytes = msg
	case mcp.JSONRPCError:
		// If it's already an error type, return it directly
		s.logger.Error("Error executing tool '%s': %v", toolName, msg.Error.Message)
		return "", fmt.Errorf("error executing tool '%s': %s", toolName, msg.Error.Message)
	default:
		// For any other type, try to marshal it
		var err error
		responseBytes, err = json.Marshal(jsonMsg)
		if err != nil {
			s.logger.Error("Failed to marshal JSON-RPC response: %v", err)
			return "", fmt.Errorf("failed to marshal JSON-RPC response: %w", err)
		}
	}

	// Debug output to trace the exact response
	s.logger.Debug("Received JSON-RPC response: %s", string(responseBytes))

	// Check if the response is a JSON-RPC error
	var errResp mcp.JSONRPCError
	if err := json.Unmarshal(responseBytes, &errResp); err == nil && errResp.Error.Code != 0 {
		s.logger.Error("Error executing tool '%s': %v", toolName, errResp.Error.Message)
		return "", fmt.Errorf("error executing tool '%s': %s", toolName, errResp.Error.Message)
	}

	// Parse the result from the response
	var resp mcp.JSONRPCResponse
	if err := json.Unmarshal(responseBytes, &resp); err != nil {
		s.logger.Error("Failed to parse JSON-RPC response: %v", err)
		return "", fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	// Convert result to CallToolResult
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		s.logger.Error("Failed to marshal tool result: %v", err)
		return "", fmt.Errorf("failed to marshal tool result: %w", err)
	}

	// Log the result for debugging
	s.logger.Debug("Tool result (raw): %s", string(resultBytes))

	// Try to parse as a map first to handle different possible response structures
	var resultMap map[string]interface{}
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		s.logger.Error("Failed to parse tool result as map: %v", err)
		return "", fmt.Errorf("failed to parse tool result: %w", err)
	}

	// Extract text content from the result, handling different possible structures
	var resultText string

	// Try to get content from the map
	if contentVal, ok := resultMap["content"]; ok {
		// Handle content as array or single object
		switch content := contentVal.(type) {
		case []interface{}:
			// It's an array of content items
			for _, item := range content {
				if contentObj, ok := item.(map[string]interface{}); ok {
					if textVal, ok := contentObj["text"]; ok {
						if text, ok := textVal.(string); ok {
							resultText += text
						}
					}
				}
			}
		case map[string]interface{}:
			// It's a single content object
			if textVal, ok := content["text"]; ok {
				if text, ok := textVal.(string); ok {
					resultText = text
				}
			}
		case string:
			// It's a direct string content
			resultText = content
		}
	} else {
		// If there's no "content" field, look for a direct "text" field
		if textVal, ok := resultMap["text"]; ok {
			if text, ok := textVal.(string); ok {
				resultText = text
			}
		}
	}

	// If we couldn't extract text content, try using the original text template from the tool config
	if resultText == "" {
		// Try to get the original tool config to access the output template
		cfg, err := config.NewConfigFromFile(s.configFile)
		if err == nil {
			toolIndex := s.findToolByName(cfg.MCP.Tools, toolName)
			if toolIndex >= 0 {
				// Get the template from the YAML file structure directly
				// This is a workaround for accessing custom fields that might not be in our structs
				var toolConfig map[string]interface{}
				toolBytes, err := json.Marshal(cfg.MCP.Tools[toolIndex])
				if err == nil {
					if err := json.Unmarshal(toolBytes, &toolConfig); err == nil {
						if outputMap, ok := toolConfig["output"].(map[string]interface{}); ok {
							if template, ok := outputMap["template"].(string); ok && template != "" {
								// Simple variable substitution for ${variable} format
								for argName, argValue := range args {
									if strValue, ok := argValue.(string); ok {
										template = strings.ReplaceAll(template, "${"+argName+"}", strValue)
									}
								}
								resultText = template
							}
						}
					}
				}
			}
		}
	}

	return resultText, nil
}

// mustMarshalJSON marshals an object to JSON and panics on error
func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return data
}

// StartHTTP initializes the MCP server and starts an HTTP server for MCP protocol over HTTP/SSE
func (s *Server) StartHTTP(port int) error {
	s.logger.Info("Initializing MCP HTTP server on port %d", port)
	if err := s.CreateServer(); err != nil {
		return err
	}
	http.HandleFunc("/sse", s.handleMCPHTTP)
	addr := fmt.Sprintf(":%d", port)
	s.logger.Info("MCP HTTP server listening on http://localhost%s/sse", addr)
	return http.ListenAndServe(addr, nil)
}

// handleMCPHTTP handles HTTP POST requests for MCP protocol
func (s *Server) handleMCPHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("New HTTP connection from %s %s %s", r.RemoteAddr, r.Method, r.URL.Path)
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		s.logger.Error("Failed to read request body from %s: %v", r.RemoteAddr, err)
		return
	}

	s.logger.Info("Request body: %s", string(body))

	// Parse the JSON-RPC request
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		s.logger.Error("Invalid JSON from %s: %v", r.RemoteAddr, err)
		return
	}

	// Intercept "initialize" method
	if method, ok := req["method"].(string); ok && method == "initialize" {
		id := req["id"]
		s.logger.Info("Received 'initialize' from %s (id=%v)", r.RemoteAddr, id)

		// Extract protocolVersion from params
		protocolVersion := ""
		if params, ok := req["params"].(map[string]interface{}); ok {
			if pv, ok := params["protocolVersion"].(string); ok {
				protocolVersion = pv
			}
		}
		if protocolVersion == "" {
			protocolVersion = "2025-03-26" // fallback, should always be present
		}

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result": map[string]interface{}{
				"serverInfo": map[string]interface{}{
					"name":    "MCPShell",
					"version": s.version,
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"allowedTools": s.getAllowedToolNames(),
					},
				},
				"sessionId":       "local",
				"protocolVersion": protocolVersion,
			},
		}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			s.logger.Error("Failed to marshal response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		s.logger.Info("Response: %s", string(respBytes))
		if _, err := w.Write(respBytes); err != nil {
			s.logger.Error("Failed to write response: %v", err)
		}
		return
	}

	// Fallback to normal MCP handling
	s.logger.Info("Received MCP request from %s: method=%v id=%v", r.RemoteAddr, req["method"], req["id"])
	ctx := r.Context()
	resp := s.mcpServer.HandleMessage(ctx, body)
	var respBytes []byte
	switch v := resp.(type) {
	case []byte:
		respBytes = v
	case string:
		respBytes = []byte(v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			s.logger.Error("Failed to marshal MCP response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		respBytes = b
	}
	s.logger.Info("Response: %s", string(respBytes))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(respBytes); err != nil {
		s.logger.Error("Failed to write response: %v", err)
	}
}

// Helper to get tool names
func (s *Server) getAllowedToolNames() []string {
	tools, err := s.GetTools()
	if err != nil {
		return []string{}
	}
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	return names
}
