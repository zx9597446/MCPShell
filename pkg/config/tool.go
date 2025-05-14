package config

import (
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/inercia/MCPShell/pkg/common"
)

// Tool holds an MCP tool and its associated handling information.
type Tool struct {
	// MCPTool is the MCP client-facing tool definition
	MCPTool mcp.Tool

	// Config is the original tool configuration
	Config MCPToolConfig
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
func (t *Tool) checkToolRequirements() bool {
	prerequisites := t.Config.Requirements

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

// CreateMCPTool creates an MCP tool from a tool configuration.
//
// Parameters:
//   - config: The tool configuration from which to create the MCP tool
//
// Returns:
//   - An mcp.Tool object ready to be registered with the MCP server
func CreateMCPTool(config MCPToolConfig) mcp.Tool {
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
