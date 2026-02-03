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

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

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
