// Package command provides functions for creating and executing command handlers.
//
// This package defines the core functionality for translating MCP tool calls
// into shell command executions, providing the bridge between the MCP protocol
// and the underlying operating system commands.
package command

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
	"github.com/inercia/mcp-cli-adapter/pkg/config"
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
func NewCommandHandler(tool config.ToolDefinition, params map[string]common.ParamConfig, shell string, logger *log.Logger) (*CommandHandler, error) {
	// Check required parameters
	if logger == nil {
		return nil, fmt.Errorf("logger is required for CommandHandler")
	}

	// Log tool creation
	logger.Printf("Creating handler for tool '%s'", tool.Tool.Name)

	// Compile constraints during initialization
	var compiled *common.CompiledConstraints
	var err error

	if len(tool.Constraints) > 0 {
		logger.Printf("Compiling %d constraints for tool '%s'", len(tool.Constraints), tool.Tool.Name)

		compiled, err = common.NewCompiledConstraints(tool.Constraints, params, logger)
		if err != nil {
			logger.Printf("Failed to compile constraints for tool %s: %v", tool.Tool.Name, err)
			return nil, fmt.Errorf("constraint compilation error: %w", err)
		}

		logger.Printf("Successfully compiled constraints for tool '%s'", tool.Tool.Name)
	}

	// Convert the runner options to RunnerOptions
	runnerOpts := RunnerOptions{}
	if tool.RunnerOpts != nil {
		for k, v := range tool.RunnerOpts {
			runnerOpts[k] = v
		}
		logger.Printf("Runner options for tool '%s': %v", tool.Tool.Name, runnerOpts)
	}

	// Create and return the handler
	return &CommandHandler{
		cmd:                 tool.HandlerCmd,
		output:              tool.Output,
		constraints:         tool.Constraints,
		params:              params,
		constraintsCompiled: compiled,
		envVars:             tool.EnvVars,
		shell:               shell,
		toolName:            tool.Tool.Name,
		runnerType:          tool.Runner,
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
		// Log the tool execution
		h.logger.Printf("Tool execution requested for '%s'", h.toolName)
		h.logger.Printf("Arguments: %v", request.Params.Arguments)

		// Validate constraints before executing command
		if h.constraintsCompiled != nil {
			h.logger.Printf("Evaluating %d constraints", len(h.constraints))

			// Evaluate constraints with logging
			valid, err := h.constraintsCompiled.Evaluate(request.Params.Arguments, h.params)
			if err != nil {
				h.logger.Printf("Constraint evaluation error: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("constraint evaluation error: %v", err)), nil
			}

			if !valid {
				h.logger.Printf("Command execution blocked by constraints")
				return mcp.NewToolResultError("command execution blocked by constraints"), nil
			}

			h.logger.Printf("All constraints passed validation")
		}

		// Process the command template with the tool arguments
		h.logger.Printf("Processing command template: %s", h.cmd)

		cmd, err := h.processTemplate(h.cmd, request.Params.Arguments)
		if err != nil {
			h.logger.Printf("Error processing command template: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("error processing command template: %v", err)), nil
		}

		h.logger.Printf("Processed command: %s", cmd)

		// Prepare environment variables
		env := h.getEnvironmentVariables()

		h.logger.Printf("Executing command: %s", cmd)

		// Determine which runner to use based on the configuration
		runnerType := RunnerTypeExec // default runner
		if h.runnerType != "" {
			h.logger.Printf("Using configured runner type: %s", h.runnerType)
			switch h.runnerType {
			case string(RunnerTypeExec):
				runnerType = RunnerTypeExec
			case string(RunnerTypeSandboxExec):
				runnerType = RunnerTypeSandboxExec
			default:
				h.logger.Printf("Unknown runner type '%s', falling back to default runner", h.runnerType)
			}
		}

		// Start with the configured runner options from the tool definition
		runnerOptions := RunnerOptions{}
		for k, v := range h.runnerOpts {
			runnerOptions[k] = v
		}

		// Add or override with any options from the request parameters if present
		if runnerOpts, ok := request.Params.Arguments["options"].(map[string]interface{}); ok {
			h.logger.Printf("Found runner options in request parameters: %v", runnerOpts)
			for k, v := range runnerOpts {
				runnerOptions[k] = v
			}
		}

		// Create the appropriate runner with options
		runner, err := NewRunner(runnerType, runnerOptions, h.logger)
		if err != nil {
			h.logger.Printf("Error creating runner: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("error creating runner: %v", err)), nil
		}

		// Execute the command
		commandOutput, err := runner.Run(ctx, h.shell, cmd, []string{}, env)
		if err != nil {
			h.logger.Printf("Error executing command: %v", err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Process the output
		finalOutput := commandOutput
		h.logger.Printf("Command output: %s", finalOutput)

		// Apply prefix if provided
		if h.output.Prefix != "" {
			h.logger.Printf("Applying output prefix template: %s", h.output.Prefix)

			// Process the prefix template with the tool arguments
			prefix, err := h.processTemplate(h.output.Prefix, request.Params.Arguments)
			if err != nil {
				h.logger.Printf("Error processing output prefix template: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("error processing output prefix template: %v", err)), nil
			}

			// Combine prefix and command output
			finalOutput = strings.TrimSpace(prefix) + " " + finalOutput
			h.logger.Printf("Final output with prefix: %s", finalOutput)
		}

		h.logger.Printf("Tool execution completed successfully")
		return mcp.NewToolResultText(finalOutput), nil
	}
}

// processTemplate processes a command template with the given arguments.
// It uses Go's template engine to substitute variables in the template.
//
// Parameters:
//   - args: Map of variable names to their values
//
// Returns:
//   - The processed command string with substituted variables
//   - An error if template processing fails
func (h *CommandHandler) processTemplate(text string, args map[string]interface{}) (string, error) {
	// Create a template from the command string
	tmpl, err := template.New("command").Option("missingkey=zero").Parse(text)
	if err != nil {
		return "", err
	}

	// Execute the template with the arguments
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, args); err != nil {
		return "", err
	}

	// fix https://github.com/golang/go/issues/24963
	res := buf.String()
	res = strings.ReplaceAll(res, "<no value>", "")

	return res, nil
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
