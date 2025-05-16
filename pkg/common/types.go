// Package common provides shared utilities and types used across the MCPShell.
package common

import (
	"fmt"
	"strconv"
	"strings"
)

// OutputConfig defines how tool output should be formatted before being returned.
type OutputConfig struct {
	// Prefix is a template string that gets prepended to the command output.
	// It can use the same template variables as the command itself.
	Prefix string `yaml:"prefix,omitempty"`
}

// ParamConfig defines the configuration for a single parameter in a tool.
type ParamConfig struct {
	// Type specifies the parameter data type. Valid values: "string" (default), "number"/"integer", "boolean"
	Type string `yaml:"type,omitempty"`

	// Description provides information about the parameter's purpose
	Description string `yaml:"description"`

	// Required indicates whether the parameter must be provided
	Required bool `yaml:"required,omitempty"`

	// Default specifies a default value to use when the parameter is not provided
	Default interface{} `yaml:"default,omitempty"`
}

// LoggingConfig defines configuration options for application logging.
type LoggingConfig struct {
	// File is the path to the log file
	File string

	// Level sets the logging verbosity (e.g., "info", "debug", "error")
	Level string `yaml:"level,omitempty"`
}

// ConvertStringToType converts a string value to the appropriate type based on the parameter type.
// This is used when parsing command line arguments for direct tool execution.
//
// Parameters:
//   - value: The string value to convert
//   - paramType: The parameter type ("string", "number", "integer", "boolean")
//
// Returns:
//   - The converted value
//   - An error if the conversion fails
func ConvertStringToType(value string, paramType string) (interface{}, error) {
	// Default to string if type is not specified
	if paramType == "" {
		paramType = "string"
	}

	switch paramType {
	case "string":
		return value, nil
	case "number":
		// Try to parse as float64
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse '%s' as number: %w", value, err)
		}
		return floatVal, nil
	case "integer":
		// Try to parse as int64
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse '%s' as integer: %w", value, err)
		}
		return intVal, nil
	case "boolean":
		// Convert to lowercase for consistent comparison
		lowerVal := strings.ToLower(value)

		// Check for various boolean representations
		switch lowerVal {
		case "true", "t", "yes", "y", "1":
			return true, nil
		case "false", "f", "no", "n", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("failed to parse '%s' as boolean", value)
		}
	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", paramType)
	}
}
