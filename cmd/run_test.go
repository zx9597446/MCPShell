package root

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
	"github.com/inercia/mcp-cli-adapter/pkg/server"
)

func TestServerStartup(t *testing.T) {
	// Initialize logger for test
	testLogger, err := common.NewLogger("", "", common.LogLevelNone, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a temporary config file
	tempDir, err := os.MkdirTemp("", "mcp-cli-adapter-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create a basic config file with constraints
	testConfigFile := filepath.Join(tempDir, "config.yaml")
	configContent := `mcp:
  tools:
    - name: "hello_world"
      description: "Say hello to someone"
      params:
        name:
          type: string
          description: "Name of the person to greet"
          required: true
      constraints:
        - "name.size() <= 100"
        - "!name.contains('/')"
      run:
        command: "echo 'Hello, {{ .name }}!'"
    
    - name: "calculator"
      description: "Perform a calculation"
      params:
        expression:
          type: string
          description: "The mathematical expression to evaluate"
          required: true
      constraints:
        - "expression.size() <= 200"
        - "!expression.matches('.*[;&|].*')"
      run:
        command: "echo '{{ .expression }}' | bc -l"
      output:
        prefix: "Result: "
`

	err = os.WriteFile(testConfigFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create a server instance for testing (we won't actually start it)
	srv := server.New(server.Config{
		ConfigFile: testConfigFile,
		Logger:     testLogger,
		Version:    "test",
	})

	// Test loading tools (this won't actually start the server)
	// Just verify the server can be created successfully
	if srv == nil {
		t.Fatal("Failed to create server instance")
	}
}

func TestFindToolName(t *testing.T) {
	// This test is just a placeholder since findToolByName is now private
	// and belongs to the server package
	t.Skip("findToolByName functionality is now tested in the server package")
}
