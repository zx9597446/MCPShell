package command

import (
	"context"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
	"github.com/inercia/mcp-cli-adapter/pkg/config"
)

// Create a test logger that discards output to keep test output clean
var testLogger = log.New(io.Discard, "", 0)

func TestCommandHandler(t *testing.T) {
	tests := []struct {
		name        string
		cmdTemplate string
		output      common.OutputConfig
		constraints []string
		paramTypes  map[string]common.ParamConfig
		args        map[string]interface{}
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "No constraints",
			cmdTemplate: "echo 'Hello, {{ .name }}'",
			output:      common.OutputConfig{},
			constraints: nil,
			paramTypes: map[string]common.ParamConfig{
				"name": {Type: "string", Description: "User name"},
			},
			args:      map[string]interface{}{"name": "Alice"},
			wantError: false,
		},
		{
			name:        "Empty constraints list",
			cmdTemplate: "echo 'Hello, {{ .name }}'",
			output:      common.OutputConfig{},
			constraints: []string{},
			paramTypes: map[string]common.ParamConfig{
				"name": {Type: "string", Description: "User name"},
			},
			args:      map[string]interface{}{"name": "Alice"},
			wantError: false,
		},
		{
			name:        "Valid constraint - passed",
			cmdTemplate: "echo 'Hello, {{ .name }}'",
			output:      common.OutputConfig{},
			constraints: []string{"name.size() <= 10"},
			paramTypes: map[string]common.ParamConfig{
				"name": {Type: "string", Description: "User name"},
			},
			args:      map[string]interface{}{"name": "Alice"},
			wantError: false,
		},
		{
			name:        "Valid constraint - failed",
			cmdTemplate: "echo 'Hello, {{ .name }}'",
			output:      common.OutputConfig{},
			constraints: []string{"name.size() <= 5"},
			paramTypes: map[string]common.ParamConfig{
				"name": {Type: "string", Description: "User name"},
			},
			args:      map[string]interface{}{"name": "Elizabeth"},
			wantError: true,
			errorMsg:  "command execution blocked by constraints",
		},
		{
			name:        "Multiple constraints - all pass",
			cmdTemplate: "echo 'Value: {{ .value }}'",
			output:      common.OutputConfig{},
			constraints: []string{"value > 0.0", "value < 100.0"},
			paramTypes: map[string]common.ParamConfig{
				"value": {Type: "number", Description: "Numeric value"},
			},
			args:      map[string]interface{}{"value": 50.0},
			wantError: false,
		},
		{
			name:        "Multiple constraints - one fails",
			cmdTemplate: "echo 'Value: {{ .value }}'",
			output:      common.OutputConfig{},
			constraints: []string{"value > 0.0", "value < 100.0"},
			paramTypes: map[string]common.ParamConfig{
				"value": {Type: "number", Description: "Numeric value"},
			},
			args:      map[string]interface{}{"value": 150.0},
			wantError: true,
			errorMsg:  "command execution blocked by constraints",
		},
		{
			name:        "Security constraint - passed",
			cmdTemplate: "echo '{{ .text }}'",
			output:      common.OutputConfig{},
			constraints: []string{"!text.contains(';')", "!text.contains('&')", "!text.contains('|')"},
			paramTypes: map[string]common.ParamConfig{
				"text": {Type: "string", Description: "Text to echo"},
			},
			args:      map[string]interface{}{"text": "Hello, world!"},
			wantError: false,
		},
		{
			name:        "Security constraint - failed",
			cmdTemplate: "echo '{{ .text }}'",
			output:      common.OutputConfig{},
			constraints: []string{"!text.contains(';')", "!text.contains('&')", "!text.contains('|')"},
			paramTypes: map[string]common.ParamConfig{
				"text": {Type: "string", Description: "Text to echo"},
			},
			args:      map[string]interface{}{"text": "Hello; rm -rf /"},
			wantError: true,
			errorMsg:  "command execution blocked by constraints",
		},
		{
			name:        "Whitelist constraint - passed",
			cmdTemplate: "{{ .command }}",
			output:      common.OutputConfig{},
			constraints: []string{"['ls', 'ps', 'echo', 'pwd'].exists(c, c == command)"},
			paramTypes: map[string]common.ParamConfig{
				"command": {Type: "string", Description: "Command to run"},
			},
			args:      map[string]interface{}{"command": "echo"},
			wantError: false,
		},
		{
			name:        "Whitelist constraint - failed",
			cmdTemplate: "{{ .command }}",
			output:      common.OutputConfig{},
			constraints: []string{"['ls', 'ps', 'echo', 'pwd'].exists(c, c == command)"},
			paramTypes: map[string]common.ParamConfig{
				"command": {Type: "string", Description: "Command to run"},
			},
			args:      map[string]interface{}{"command": "rm"},
			wantError: true,
			errorMsg:  "command execution blocked by constraints",
		},
		{
			name:        "Invalid constraint syntax",
			cmdTemplate: "echo 'Hello, {{ .name }}'",
			output:      common.OutputConfig{},
			constraints: []string{"name.invalid()"},
			paramTypes: map[string]common.ParamConfig{
				"name": {Type: "string", Description: "User name"},
			},
			args:      map[string]interface{}{"name": "Alice"},
			wantError: true,
			errorMsg:  "constraint compilation error",
		},
		{
			name:        "Output with prefix - passed constraint",
			cmdTemplate: "echo 'World'",
			output:      common.OutputConfig{Prefix: "Hello,"},
			constraints: []string{"true"},
			paramTypes:  map[string]common.ParamConfig{},
			args:        map[string]interface{}{},
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock tool definition
			toolDef := config.ToolDefinition{
				HandlerCmd:  tt.cmdTemplate,
				Output:      tt.output,
				Constraints: tt.constraints,
				EnvVars:     []string{},
				Tool: mcp.Tool{
					Name: "test-tool",
				},
			}

			// Create a new command handler
			cmdHandler, err := NewCommandHandler(toolDef, tt.paramTypes, "", testLogger)

			// For invalid constraint syntax test, we expect an error during creation
			if tt.name == "Invalid constraint syntax" {
				if err == nil {
					t.Errorf("NewCommandHandler() did not return an error when expected")
					return
				}
				if !strings.Contains(err.Error(), "constraint compilation error") {
					t.Errorf("NewCommandHandler() error = %v, want error containing 'constraint compilation error'", err)
				}
				return
			}

			// Otherwise, we don't expect an error during creation
			if err != nil {
				t.Errorf("NewCommandHandler() unexpected error = %v", err)
				return
			}

			// Get the handler function
			handler := cmdHandler.GetMCPHandler()

			// Create a request
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			// Call the handler
			result, err := handler(context.Background(), request)

			// We don't expect any actual Go errors during handler execution
			if err != nil {
				t.Errorf("CommandHandler.GetMCPHandler() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Errorf("CommandHandler.GetMCPHandler() returned nil result")
				return
			}

			// Check error condition in the result object
			if tt.wantError {
				if !result.IsError {
					t.Errorf("CommandHandler.GetMCPHandler() did not return a result with IsError=true when expected")
					return
				}

				// Check if result content contains the expected error message
				if tt.errorMsg != "" {
					hasErrorMsg := false
					for _, content := range result.Content {
						if textContent, ok := content.(mcp.TextContent); ok {
							if strings.Contains(textContent.Text, tt.errorMsg) {
								hasErrorMsg = true
								break
							}
						}
					}

					if !hasErrorMsg {
						t.Errorf("CommandHandler.GetMCPHandler() result does not contain error message %q", tt.errorMsg)
					}
				}
			} else {
				if result.IsError {
					t.Errorf("CommandHandler.GetMCPHandler() returned a result with IsError=true when not expected")
					return
				}
			}
		})
	}
}
