// Package config provides configuration loading and handling functionality.
//
// It defines the data structures for representing the application's configuration,
// which is loaded from YAML files, and includes utilities for parsing and
// processing these configurations.
package config

import (
	"fmt"
	"io"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
)

// Config represents the top-level configuration structure for the application.
type Config struct {
	// MCP contains the configuration specific to the MCP server and tools
	MCP MCPConfig `yaml:"mcp"`
}

// MCPConfig represents the MCP server configuration section.
type MCPConfig struct {
	// Description is a text shown to AI clients that explains what this server does
	Description string `yaml:"description,omitempty"`

	// Run contains runtime configuration
	Run MCPRunConfig `yaml:"run,omitempty"`

	// Tools is a list of tool definitions that will be provided to clients
	Tools []ToolConfig `yaml:"tools"`
}

// MCPRunConfig represents run-specific configuration options.
type MCPRunConfig struct {
	// Shell is the shell to use for executing commands (e.g., bash, sh, zsh)
	Shell string `yaml:"shell,omitempty"`
}

// ToolConfig represents a single tool configuration.
type ToolConfig struct {
	// Name is the unique identifier for the tool
	Name string `yaml:"name"`

	// Requirements is a list of tool names that must be executed before this tool
	Requirements ToolRequirements `yaml:"prerequisites,omitempty"`

	// Description explains what the tool does (shown to AI clients)
	Description string `yaml:"description"`

	// Params defines the parameters that the tool accepts
	Params map[string]common.ParamConfig `yaml:"params"`

	// Constraints are expressions that limit when the tool can be executed
	Constraints []string `yaml:"constraints,omitempty"`

	// Run specifies how to execute the tool
	Run RunConfig `yaml:"run"`

	// Output specifies how to format the tool's output
	Output common.OutputConfig `yaml:"output,omitempty"`
}

// ToolRequirements represents a prerequisite tool configuration.
// If these prerequisites are not met, the tool will not even be shown as
// available to the client.
// This allows for tools to be conditionally shown based on the user's system.
type ToolRequirements struct {
	// OS is the operating system that the prerequisite tool must be installed on
	OS string `yaml:"os,omitempty"`

	// Executables is a list of executable names that must be present in the system
	Executables []string `yaml:"executables"`
}

// RunConfig represents the run configuration for a tool.
type RunConfig struct {
	// Runner is the type of runner to use for executing the command
	Runner string `yaml:"runner,omitempty"`

	// Command is a template for the shell command to execute
	Command string `yaml:"command"`

	// Env is a list of environment variable names to pass from the parent process
	Env []string `yaml:"env,omitempty"`

	// Options for the runner
	Options map[string]interface{} `yaml:"options,omitempty"`
}

// Type aliases for common structs to simplify imports
type (
	OutputConfig = common.OutputConfig
	ParamConfig  = common.ParamConfig
)

// LoadConfig loads the configuration from a YAML file at the specified path.
//
// Parameters:
//   - filepath: Path to the YAML configuration file
//
// Returns:
//   - A pointer to the loaded Config structure
//   - An error if loading or parsing fails
func LoadConfig(filepath string) (*Config, error) {
	// Open the configuration file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Read the file content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the YAML content
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// CreateTools converts the configuration's tool definitions into a list of
// executable ToolDefinition objects ready to be registered with the MCP server.
//
// Returns:
//   - A slice of ToolDefinition objects
func (c *Config) CreateTools() []ToolDefinition {
	var tools []ToolDefinition

	for _, toolConfig := range c.MCP.Tools {
		// Check prerequisites before creating the tool
		if !checkToolRequirements(toolConfig.Requirements) {
			continue // Skip this tool if prerequisites are not met
		}

		tool := ToolDefinition{
			Tool:        createMCPTool(toolConfig),
			HandlerCmd:  toolConfig.Run.Command,
			Runner:      toolConfig.Run.Runner,
			Output:      toolConfig.Output,
			Constraints: toolConfig.Constraints,
			EnvVars:     toolConfig.Run.Env,
			RunnerOpts:  toolConfig.Run.Options,
		}
		tools = append(tools, tool)
	}

	return tools
}

// checkToolRequirements checks if all prerequisites for a tool are met.
// If no prerequisites are specified, it returns true.
// If any prerequisites are specified, all of them must be satisfied.
//
// Parameters:
//   - prerequisites: A slice of ToolPrerequisites to check
//
// Returns:
//   - true if all prerequisites are met or if there are no prerequisites,
//     false otherwise
func checkToolRequirements(prerequisites ToolRequirements) bool {
	// Check if any of the prerequisite sets are met
	// Check if OS matches (if specified)
	if !common.CheckOSMatches(prerequisites.OS) {
		return false
	}

	// Check if all required executables exist
	for _, execName := range prerequisites.Executables {
		if !common.CheckExecutableExists(execName) {
			return false
		}
	}

	return true
}

// ToolDefinition holds an MCP tool and its associated handling information.
type ToolDefinition struct {
	// Tool is the MCP client-facing tool definition
	Tool mcp.Tool

	// HandlerCmd is the command template to execute
	HandlerCmd string

	// Runner is the type of runner to use for command execution
	Runner string

	// Output defines how to format the tool's output
	Output OutputConfig

	// Constraints are expressions that must be satisfied to allow execution
	Constraints []string

	// EnvVars is a list of environment variable names to pass from the parent process
	EnvVars []string

	// RunnerOpts is the options for the runner
	RunnerOpts map[string]interface{}
}

// createMCPTool creates an MCP tool from a tool configuration.
//
// Parameters:
//   - config: The tool configuration from which to create the MCP tool
//
// Returns:
//   - An mcp.Tool object ready to be registered with the MCP server
func createMCPTool(config ToolConfig) mcp.Tool {
	var options []mcp.ToolOption

	// Add description
	options = append(options, mcp.WithDescription(config.Description))

	// Add parameters
	for name, param := range config.Params {
		// If type is not specified, default to "string"
		paramType := param.Type
		if paramType == "" {
			paramType = "string"
		}

		switch paramType {
		case "string":
			if param.Required {
				options = append(options, mcp.WithString(name, mcp.Required(), mcp.Description(param.Description)))
			} else {
				options = append(options, mcp.WithString(name, mcp.Description(param.Description)))
			}
		case "number", "integer":
			if param.Required {
				options = append(options, mcp.WithNumber(name, mcp.Required(), mcp.Description(param.Description)))
			} else {
				options = append(options, mcp.WithNumber(name, mcp.Description(param.Description)))
			}
		case "boolean":
			if param.Required {
				options = append(options, mcp.WithBoolean(name, mcp.Required(), mcp.Description(param.Description)))
			} else {
				options = append(options, mcp.WithBoolean(name, mcp.Description(param.Description)))
			}
		}
	}

	return mcp.NewTool(config.Name, options...)
}
