// Package command provides functions for creating and executing command handlers.
//
// This package defines the core functionality for translating MCP tool calls
// into shell command executions, providing the bridge between the MCP protocol
// and the underlying operating system commands.
package command

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
)

// CommandHandler encapsulates the configuration and behavior needed to handle tool commands.
type CommandHandler struct {
	cmd                 string                        // the command to execute
	output              common.OutputConfig           // the output configuration
	constraints         []string                      // the constraints to evaluate
	constraintsCompiled *common.CompiledConstraints   // ... and the compiled versions
	params              map[string]common.ParamConfig // the parameter configurations
	envVars             []string                      // the environment variables passed to the command
	shell               string                        // the shell to use
	toolName            string                        // the name of the tool
	runnerType          string                        // the type of runner to use
	runnerOpts          RunnerOptions                 // the options for the runner

	logger *log.Logger
}

// NewCommandHandler creates a new CommandHandler instance.
//
// Parameters:
//   - tool: The tool definition containing command, constraints, and output configuration
//   - params: Map of parameter names to their type configurations
//   - shell: The shell to use for command execution
//   - logger: Logger for detailed execution information (required)
//
// Returns:
//   - A new CommandHandler instance and nil if successful
//   - nil and an error if constraint compilation fails or if a required parameter is missing
func NewCommandHandler(tool config.Tool, params map[string]common.ParamConfig, shell string, logger *log.Logger) (*CommandHandler, error) {
	// Check required parameters
	if logger == nil {
		return nil, fmt.Errorf("logger is required for CommandHandler")
	}

	// Log tool creation
	logger.Printf("Creating handler for tool '%s'", tool.MCPTool.Name)

	// Compile constraints during initialization
	var compiled *common.CompiledConstraints
	var err error

	if len(tool.Config.Constraints) > 0 {
		logger.Printf("Compiling %d constraints for tool '%s'", len(tool.Config.Constraints), tool.MCPTool.Name)

		compiled, err = common.NewCompiledConstraints(tool.Config.Constraints, params, logger)
		if err != nil {
			logger.Printf("Failed to compile constraints for tool %s: %v", tool.MCPTool.Name, err)
			return nil, fmt.Errorf("constraint compilation error: %w", err)
		}

		logger.Printf("Successfully compiled constraints for tool '%s'", tool.MCPTool.Name)
	}

	// Convert the runner options to RunnerOptions
	runnerOpts := RunnerOptions{}
	if tool.Config.Run.Options != nil {
		for k, v := range tool.Config.Run.Options {
			runnerOpts[k] = v
		}
		logger.Printf("Runner options for tool '%s': %v", tool.MCPTool.Name, runnerOpts)
	}

	// Create and return the handler
	return &CommandHandler{
		cmd:                 tool.Config.Run.Command,
		output:              tool.Config.Output,
		constraints:         tool.Config.Constraints,
		params:              params,
		constraintsCompiled: compiled,
		envVars:             tool.Config.Run.Env,
		shell:               shell,
		toolName:            tool.MCPTool.Name,
		runnerType:          tool.Config.Run.Runner,
		runnerOpts:          runnerOpts,
		logger:              logger,
	}, nil
}

// GetMCPHandler returns a function that handles MCP tool calls by executing shell commands.
//
// This is the function that should be registered with the MCP server.
//
// Returns:
//   - A function that handles MCP tool calls
func (h *CommandHandler) GetMCPHandler() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract runner options if present
		var runnerOpts map[string]interface{}
		if opts, ok := request.Params.Arguments["options"].(map[string]interface{}); ok {
			runnerOpts = opts
		}

		// Execute the command using the common implementation
		output, err, _ := h.executeToolCommand(ctx, request.Params.Arguments, runnerOpts)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(output), nil
	}
}

// getEnvironmentVariables gets values for specified environment variables from parent process
// and returns them in KEY=VALUE format for the command
func (h *CommandHandler) getEnvironmentVariables() []string {
	if len(h.envVars) == 0 {
		return nil
	}

	envVars := make([]string, 0, len(h.envVars))
	for _, name := range h.envVars {
		if value, exists := os.LookupEnv(name); exists {
			envVars = append(envVars, name+"="+value)
		} else {
			envVars = append(envVars, name+"=")
		}
	}

	return envVars
}
