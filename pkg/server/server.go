// Package server implements the MCP server functionality.
//
// It handles loading tool configurations, starting the server,
// and processing requests from AI clients using the MCP protocol.
package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/inercia/mcp-cli-adapter/pkg/command"
	"github.com/inercia/mcp-cli-adapter/pkg/common"
	"github.com/inercia/mcp-cli-adapter/pkg/config"
)

// Server represents the MCP CLI adapter server that handles tool registration
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
	ConfigFile  string         // Path to the YAML configuration file
	Shell       string         // Shell to use for executing commands
	Logger      *common.Logger // Logger for server operations
	Version     string         // Version string for the server
	Description string         // Description shown to AI clients
}

// New creates a new Server instance with the provided configuration
//
// Parameters:
//   - cfg: The server configuration
//
// Returns:
//   - A new Server instance
func New(cfg Config) *Server {
	return &Server{
		configFile:  cfg.ConfigFile,
		shell:       cfg.Shell,
		logger:      cfg.Logger,
		version:     cfg.Version,
		description: cfg.Description,
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
	cfg, err := config.LoadConfig(s.configFile)
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
	toolDefs := cfg.CreateTools()

	// Check if some tools were filtered out due to prerequisites not met
	if len(toolDefs) < len(cfg.MCP.Tools) {
		skippedCount := len(cfg.MCP.Tools) - len(toolDefs)
		s.logger.Info("%d tool(s) would be skipped due to unmet prerequisites", skippedCount)
		fmt.Printf("%d tool(s) would be skipped due to unmet prerequisites\n", skippedCount)

		// Log which tools were skipped
		for _, toolConfig := range cfg.MCP.Tools {
			found := false
			for _, toolDef := range toolDefs {
				if toolDef.Tool.Name == toolConfig.Name {
					found = true
					break
				}
			}

			if !found {
				s.logger.Info("Tool '%s' would be skipped due to unmet prerequisites", toolConfig.Name)
				fmt.Printf("- Tool '%s' would be skipped due to unmet prerequisites\n", toolConfig.Name)
			}
		}
	}

	s.logger.Info("Validating %d tools after checking prerequisites", len(toolDefs))

	// Validate each tool definition
	for _, toolDef := range toolDefs {
		s.logger.Debug("Validating tool '%s'", toolDef.Tool.Name)

		// Find the original tool config
		toolIndex := s.findToolByName(cfg.MCP.Tools, toolDef.Tool.Name)
		if toolIndex == -1 {
			return fmt.Errorf("internal error: tool '%s' not found in configuration after creation", toolDef.Tool.Name)
		}

		// Get parameter types for constraint validation
		paramTypes := cfg.MCP.Tools[toolIndex].Params

		// Validate constraints by attempting to compile them
		if len(toolDef.Constraints) > 0 {
			s.logger.Debug("Compiling %d constraints for tool '%s'", len(toolDef.Constraints), toolDef.Tool.Name)
			_, err := common.NewCompiledConstraints(toolDef.Constraints, paramTypes, s.logger.Logger)
			if err != nil {
				s.logger.Error("Failed to compile constraints for tool '%s': %v", toolDef.Tool.Name, err)
				return fmt.Errorf("constraint compilation error for tool '%s': %w", toolDef.Tool.Name, err)
			}
			s.logger.Debug("All constraints for tool '%s' compiled successfully", toolDef.Tool.Name)
		}

		// Validate command template
		if toolDef.HandlerCmd == "" {
			s.logger.Error("Empty command template for tool '%s'", toolDef.Tool.Name)
			return fmt.Errorf("empty command template for tool '%s'", toolDef.Tool.Name)
		}

		// Format constraint information for display
		var constraintInfo string
		if len(toolDef.Constraints) > 0 {
			constraintInfo = fmt.Sprintf(" (with %d constraints)", len(toolDef.Constraints))
		} else {
			constraintInfo = ""
		}

		fmt.Printf("Validated tool: '%s'%s\n", toolDef.Tool.Name, constraintInfo)
		s.logger.Info("Validated tool: '%s'%s", toolDef.Tool.Name, constraintInfo)
	}

	s.logger.Info("Configuration validation successful")
	fmt.Println("Configuration validation successful")
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
	if err := s.createServer(); err != nil {
		return err
	}

	s.logger.Info("Starting MCP server with stdio handler")
	fmt.Println("Starting MCP server...")

	// Start the stdio server
	if err := mcpserver.ServeStdio(s.mcpServer); err != nil {
		s.logger.Error("Server error: %v", err)
		return fmt.Errorf("server error: %v", err)
	}

	return nil
}

// createServer initializes the MCP server instance
func (s *Server) createServer() error {
	// First create the MCP server
	serverName := "MCP CLI Adapter"
	var options []mcpserver.ServerOption

	// Load server configuration for description, shell, etc.
	cfg, err := config.LoadConfig(s.configFile)
	if err != nil {
		s.logger.Error("Failed to load config: %v", err)
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Use description from config if present and no description is explicitly set
	if s.description == "" && cfg.MCP.Description != "" {
		s.description = cfg.MCP.Description
		s.logger.Info("Using description from config: %s", s.description)
	}

	// Use shell from config if present and no shell is explicitly set
	if s.shell == "" && cfg.MCP.Run.Shell != "" {
		s.shell = cfg.MCP.Run.Shell
		s.logger.Info("Using shell from config: %s", s.shell)
	}

	// Add description if provided
	if s.description != "" {
		s.logger.Info("Using custom description: %s", s.description)
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
func (s *Server) loadTools(cfg *config.Config) error {
	s.logger.Info("Loading configuration from file: %s", s.configFile)

	// Check if there are any tools defined
	if len(cfg.MCP.Tools) == 0 {
		s.logger.Error("No tools defined in the configuration file")
		return fmt.Errorf("no tools defined in the configuration file")
	}

	s.logger.Info("Found %d tools in configuration", len(cfg.MCP.Tools))

	// Create and register tools
	toolDefs := cfg.CreateTools()

	// Check if some tools were filtered out due to prerequisites not met
	if len(toolDefs) < len(cfg.MCP.Tools) {
		skippedCount := len(cfg.MCP.Tools) - len(toolDefs)
		s.logger.Info("Skipped %d tool(s) due to unmet prerequisites", skippedCount)

		// Log which tools were skipped
		for _, toolConfig := range cfg.MCP.Tools {
			found := false
			for _, toolDef := range toolDefs {
				if toolDef.Tool.Name == toolConfig.Name {
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
		s.logger.Debug("Registering tool '%s'", toolDef.Tool.Name)

		// Get the parameter types for this tool
		params := cfg.MCP.Tools[s.findToolByName(cfg.MCP.Tools, toolDef.Tool.Name)].Params

		// Create a new command handler instance
		cmdHandler, err := command.NewCommandHandler(toolDef, params, s.shell, s.logger.Logger)
		if err != nil {
			s.logger.Error("Failed to create handler for tool '%s': %v", toolDef.Tool.Name, err)
			return fmt.Errorf("failed to create handler for tool '%s': %w", toolDef.Tool.Name, err)
		}

		// Get the MCP handler and wrap it with panic recovery
		safeHandler := s.wrapHandlerWithPanicRecovery(cmdHandler.GetMCPHandler())

		// Add the tool to the server
		s.mcpServer.AddTool(toolDef.Tool, safeHandler)

		// Print whether constraints are enabled
		if len(toolDef.Constraints) > 0 {
			msg := fmt.Sprintf("Registered tool: '%s' (with %d constraints)", toolDef.Tool.Name, len(toolDef.Constraints))
			fmt.Println(msg)
			s.logger.Info(msg)
		} else {
			msg := fmt.Sprintf("Registered tool: '%s'", toolDef.Tool.Name)
			fmt.Println(msg)
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
				common.RecoverPanic(s.logger.Logger, "")

				// Return an error instead of crashing
				err = fmt.Errorf("tool execution failed: internal server error")
			}
		}()

		// Call the original handler
		return handler(ctx, request)
	}
}

// findToolByName finds a tool configuration by name
func (s *Server) findToolByName(tools []config.ToolConfig, name string) int {
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
