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

	"gopkg.in/yaml.v3"

	"github.com/inercia/MCPShell/pkg/common"
)

// Config represents the top-level configuration structure for the application.
type Config struct {
	// Prompts is a list of prompts that will be provided to clients
	Prompts []Prompts `yaml:"prompts,omitempty"`

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
	Tools []MCPToolConfig `yaml:"tools"`
}

// Prompts is a list of prompts that could be provided to clients
type Prompts struct {
	// System is a list of system prompts
	System []string `yaml:"system,omitempty"`

	// User is a list of user prompts
	User []string `yaml:"user,omitempty"`
}

// MCPRunConfig represents run-specific configuration options.
type MCPRunConfig struct {
	// Shell is the shell to use for executing commands (e.g., bash, sh, zsh)
	Shell string `yaml:"shell,omitempty"`
}

// MCPToolConfig represents a single tool configuration.
type MCPToolConfig struct {
	// Name is the unique identifier for the tool
	Name string `yaml:"name"`

	// Requirements is a list of tool names that must be executed before this tool
	Requirements MCPToolRequirements `yaml:"requirements,omitempty"`

	// Description explains what the tool does (shown to AI clients)
	Description string `yaml:"description"`

	// Params defines the parameters that the tool accepts
	Params map[string]common.ParamConfig `yaml:"params"`

	// Constraints are expressions that limit when the tool can be executed
	Constraints []string `yaml:"constraints,omitempty"`

	// Run specifies how to execute the tool
	Run MCPToolRunConfig `yaml:"run"`

	// Output specifies how to format the tool's output
	Output common.OutputConfig `yaml:"output,omitempty"`
}

// MCPToolRequirements represents a prerequisite tool configuration.
// If these prerequisites are not met, the tool will not even be shown as
// available to the client.
// This allows for tools to be conditionally shown based on the user's system.
type MCPToolRequirements struct {
	// OS is the operating system that the prerequisite tool must be installed on
	OS string `yaml:"os,omitempty"`

	// Executables is a list of executable names that must be present in the system
	Executables []string `yaml:"executables"`
}

// MCPToolRunConfig represents the run configuration for a tool.
type MCPToolRunConfig struct {
	// Runner is the type of runner to use for executing the command
	Runner string `yaml:"runner,omitempty"`

	// Command is a template for the shell command to execute
	Command string `yaml:"command"`

	// Env is a list of environment variable names to pass from the parent process
	Env []string `yaml:"env,omitempty"`

	// Options for the runner
	Options map[string]interface{} `yaml:"options,omitempty"`
}

////////////////////////////////////////////////////////////////////////////////////

// NewConfigFromFile loads the configuration from a YAML file at the specified path.
//
// Parameters:
//   - filepath: Path to the YAML configuration file
//
// Returns:
//   - A pointer to the loaded Config structure
//   - An error if loading or parsing fails
func NewConfigFromFile(filepath string) (*Config, error) {
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

// GetTools converts the configuration's tool definitions into a list of
// executable ToolDefinition objects ready to be registered with the MCP server.
//
// Returns:
//   - A slice of ToolDefinition objects
func (c *Config) GetTools() []Tool {
	var tools []Tool

	for _, toolConfig := range c.MCP.Tools {
		tool := Tool{
			MCPTool: CreateMCPTool(toolConfig),
			Config:  toolConfig,
		}

		// Check prerequisites before creating the tool
		if !tool.checkToolRequirements() {
			fmt.Printf("Skipping tool %s because prerequisites are not met\n", toolConfig.Name)
			continue // Skip this tool if prerequisites are not met
		}

		tools = append(tools, tool)
	}

	return tools
}
