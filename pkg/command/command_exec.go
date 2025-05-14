// Package command provides functions for creating and executing command handlers.
package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/inercia/MCPShell/pkg/common"
)

// executeToolCommand handles the core logic of executing a command with the given parameters.
// This is a common implementation used by both direct execution and MCP handler.
//
// Parameters:
//   - ctx: Context for command execution
//   - params: Map of parameter names to their values
//   - extraRunnerOpts: Additional runner options to apply
//
// Returns:
//   - The command output as a string
//   - An error if command execution fails
//   - A slice of failed constraint messages
func (h *CommandHandler) executeToolCommand(ctx context.Context, params map[string]interface{}, extraRunnerOpts map[string]interface{}) (string, error, []string) {
	// Log the tool execution
	h.logger.Printf("Tool execution requested for '%s'", h.toolName)
	h.logger.Printf("Arguments: %v", params)

	// Validate constraints before executing command
	var failedConstraints []string
	if h.constraintsCompiled != nil {
		h.logger.Printf("Checking %d constraints", len(h.constraints))
		satisfied, failed, err := h.constraintsCompiled.Evaluate(params, h.params)
		if err != nil {
			h.logger.Printf("Error evaluating constraints: %v", err)
			return "", fmt.Errorf("error evaluating constraints: %v", err), nil
		}
		if !satisfied {
			h.logger.Printf("Constraints not satisfied, blocking execution")
			failedConstraints = failed
			errorMsg := "command execution blocked by constraints"

			// Add details about which constraints failed
			if len(failedConstraints) > 0 {
				errorMsg += ":\n"
				for i, fc := range failedConstraints {
					errorMsg += fmt.Sprintf("- Constraint %d: %s", i+1, fc)
					if i < len(failedConstraints)-1 {
						errorMsg += "\n"
					}
				}
			}

			return "", fmt.Errorf(errorMsg), failedConstraints
		}
		h.logger.Printf("All constraints satisfied")
	}

	// Process the command template with the tool arguments
	h.logger.Printf("Processing command template: %s", h.cmd)

	cmd, err := common.ProcessTemplate(h.cmd, params)
	if err != nil {
		h.logger.Printf("Error processing command template: %v", err)
		return "", fmt.Errorf("error processing command template: %v", err), nil
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
		case string(RunnerTypeFirejail):
			runnerType = RunnerTypeFirejail
		default:
			h.logger.Printf("Unknown runner type '%s', falling back to default runner", h.runnerType)
		}
	}

	// Start with the configured runner options from the tool definition
	runnerOptions := RunnerOptions{}
	for k, v := range h.runnerOpts {
		runnerOptions[k] = v
	}

	// Add or override with any options from the parameters if present
	if extraRunnerOpts != nil {
		h.logger.Printf("Found runner options in parameters: %v", extraRunnerOpts)
		for k, v := range extraRunnerOpts {
			runnerOptions[k] = v
		}
	}

	// Create the appropriate runner with options
	runner, err := NewRunner(runnerType, runnerOptions, h.logger)
	if err != nil {
		h.logger.Printf("Error creating runner: %v", err)
		return "", fmt.Errorf("error creating runner: %v", err), nil
	}

	// Execute the command
	commandOutput, err := runner.Run(ctx, h.shell, cmd, []string{}, env, params)
	if err != nil {
		h.logger.Printf("Error executing command: %v", err)
		return "", err, nil
	}

	// Process the output
	finalOutput := commandOutput
	h.logger.Printf("Command output: %s", finalOutput)

	// Apply prefix if provided
	if h.output.Prefix != "" {
		h.logger.Printf("Applying output prefix template: %s", h.output.Prefix)

		// Process the prefix template with the tool arguments
		prefix, err := common.ProcessTemplate(h.output.Prefix, params)
		if err != nil {
			h.logger.Printf("Error processing output prefix template: %v", err)
			return "", fmt.Errorf("error processing output prefix template: %v", err), nil
		}

		// Combine prefix and command output
		finalOutput = strings.TrimSpace(prefix) + "\n\n" + finalOutput
		h.logger.Printf("Final output with prefix: %s", finalOutput)
	}

	h.logger.Printf("Tool execution completed successfully")
	return finalOutput, nil, nil
}

// ExecuteCommand handles the direct execution of a command without going through the MCP server.
// This is used by the "exe" command to execute a tool directly from the command line.
//
// Parameters:
//   - params: Map of parameter names to their values
//
// Returns:
//   - The command output as a string
//   - An error if command execution fails
func (h *CommandHandler) ExecuteCommand(params map[string]interface{}) (string, error) {
	// Extract runner options if present
	var runnerOpts map[string]interface{}
	if opts, ok := params["options"].(map[string]interface{}); ok {
		runnerOpts = opts
		// Remove options from params to avoid processing them as command parameters
		tmpParams := make(map[string]interface{})
		for k, v := range params {
			if k != "options" {
				tmpParams[k] = v
			}
		}
		params = tmpParams
	}

	// Use the common implementation
	output, err, failedConstraints := h.executeToolCommand(context.Background(), params, runnerOpts)

	// If constraints failed, format the error message
	if err != nil && len(failedConstraints) > 0 {
		return "", err
	}

	return output, err
}
