package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
)

func TestServer_New(t *testing.T) {
	// Create a test logger
	logger, err := common.NewLogger("", "", common.LogLevelNone, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test creating a new server instance
	srv := New(Config{
		ConfigFile:  "test-config.yaml",
		Logger:      logger,
		Version:     "test",
		Description: "test description",
	})

	if srv == nil {
		t.Fatal("Failed to create server instance")
	}

	// Check that the fields are set correctly
	if srv.configFile != "test-config.yaml" {
		t.Errorf("Expected configFile to be 'test-config.yaml', got '%s'", srv.configFile)
	}

	if srv.version != "test" {
		t.Errorf("Expected version to be 'test', got '%s'", srv.version)
	}

	if srv.description != "test description" {
		t.Errorf("Expected description to be 'test description', got '%s'", srv.description)
	}

	if srv.mcpServer != nil {
		t.Error("Expected mcpServer to be nil until Start() is called")
	}
}

func TestServer_findToolByName(t *testing.T) {
	// Create a test logger
	logger, err := common.NewLogger("", "", common.LogLevelNone, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a mock server instance
	srv := New(Config{
		Logger: logger,
	})

	// Set up test tools
	tools := []config.ToolConfig{
		{Name: "tool1", Description: "Tool 1"},
		{Name: "tool2", Description: "Tool 2"},
		{Name: "tool3", Description: "Tool 3"},
	}

	tests := []struct {
		name     string
		toolName string
		want     int
	}{
		{
			name:     "First tool",
			toolName: "tool1",
			want:     0,
		},
		{
			name:     "Middle tool",
			toolName: "tool2",
			want:     1,
		},
		{
			name:     "Last tool",
			toolName: "tool3",
			want:     2,
		},
		{
			name:     "Non-existent tool",
			toolName: "tool4",
			want:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := srv.findToolByName(tools, tt.toolName)
			if got != tt.want {
				t.Errorf("findToolByName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServer_loadTools(t *testing.T) {
	// Create a temporary directory and config file
	tempDir, err := os.MkdirTemp("", "server-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Warning: failed to remove temp directory: %v", err)
		}
	}()

	// Create a minimal config file
	testConfigFile := filepath.Join(tempDir, "config.yaml")
	configContent := `mcp:
  tools:
    - name: "test_tool"
      description: "Test tool"
      params:
        param1:
          type: string
          description: "Test parameter"
      run:
        command: "echo 'Test'"
`

	if err := os.WriteFile(testConfigFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Skip actually loading the tools to avoid running commands
	t.Skip("loadTools() is tested in integration tests")
}
