package config

import (
	"runtime"
	"testing"
)

func TestCheckToolPrerequisites(t *testing.T) {
	// Create test cases
	tests := []struct {
		name         string
		requirements MCPToolRequirements
		expected     bool
	}{
		{
			name:         "No prerequisites",
			requirements: MCPToolRequirements{},
			expected:     true,
		},
		{
			name: "Matching OS only",
			requirements: MCPToolRequirements{
				OS:          runtime.GOOS,
				Executables: nil,
			},
			expected: true,
		},
		{
			name: "Non-matching OS",
			requirements: MCPToolRequirements{
				OS:          "non-existent-os",
				Executables: nil,
			},
			expected: false,
		},
		{
			name: "Existing executable",
			requirements: MCPToolRequirements{
				Executables: []string{"sh"}, // should exist on most systems
			},
			expected: true,
		},
		{
			name: "Non-existent executable",
			requirements: MCPToolRequirements{
				Executables: []string{"non-existent-executable-12345"},
			},
			expected: false,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := Tool{
				Config: MCPToolConfig{
					Requirements: tt.requirements,
				},
			}
			result := tool.checkToolRequirements()
			if result != tt.expected {
				t.Errorf("checkToolPrerequisites() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCreateTools_Prerequisites(t *testing.T) {
	// Create a simple config with two tools, one with unmet prerequisites
	cfg := &Config{
		MCP: MCPConfig{
			Tools: []MCPToolConfig{
				{
					Name:        "tool1",
					Description: "Tool with met prerequisites",
					Requirements: MCPToolRequirements{
						OS:          runtime.GOOS,
						Executables: []string{"sh"}, // should exist on most systems
					},
					Run: MCPToolRunConfig{
						Command: "echo 'Tool 1'",
					},
				},
				{
					Name:        "tool2",
					Description: "Tool with unmet prerequisites",
					Requirements: MCPToolRequirements{
						OS:          "non-existent-os",
						Executables: []string{"non-existent-executable-12345"},
					},
					Run: MCPToolRunConfig{
						Command: "echo 'Tool 2'",
					},
				},
			},
		},
	}

	// Create tools
	tools := cfg.GetTools()

	// We should have only one tool
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	// Check that the correct tool was created
	if len(tools) > 0 && tools[0].MCPTool.Name != "tool1" {
		t.Errorf("Expected tool named 'tool1', got '%s'", tools[0].MCPTool.Name)
	}
}
