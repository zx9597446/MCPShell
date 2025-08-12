package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetHome(t *testing.T) {
	home, err := GetHome()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	if home == "" {
		t.Error("Expected non-empty home directory")
	}

	// Verify the directory exists
	if _, err := os.Stat(home); os.IsNotExist(err) {
		t.Errorf("Home directory does not exist: %s", home)
	}
}

func TestGetMCPShellHome(t *testing.T) {
	mcpShellHome, err := GetMCPShellHome()
	if err != nil {
		t.Fatalf("Failed to get MCPShell home directory: %v", err)
	}

	if mcpShellHome == "" {
		t.Error("Expected non-empty MCPShell home directory")
	}

	// Verify it ends with .mcpshell
	expectedSuffix := ".mcpshell"
	if filepath.Base(mcpShellHome) != expectedSuffix {
		t.Errorf("Expected MCPShell home to end with %s, got %s", expectedSuffix, mcpShellHome)
	}

	// Verify it's under the user's home directory
	home, err := GetHome()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expectedPath := filepath.Join(home, ".mcpshell")
	if mcpShellHome != expectedPath {
		t.Errorf("Expected MCPShell home to be %s, got %s", expectedPath, mcpShellHome)
	}
}
