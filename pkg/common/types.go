// Package common provides shared utilities and types used across the MCPShell.
package common

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
}

// LoggingConfig defines configuration options for application logging.
type LoggingConfig struct {
	// File is the path to the log file
	File string

	// Level sets the logging verbosity (e.g., "info", "debug", "error")
	Level string `yaml:"level,omitempty"`
}
