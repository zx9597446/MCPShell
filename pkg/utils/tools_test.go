package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveToolsFile(t *testing.T) {
	// Create a temporary tools directory for testing
	tmpDir := t.TempDir()
	toolsDir := filepath.Join(tmpDir, "tools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Set the tools directory environment variable
	t.Setenv(MCPShellToolsDirEnv, toolsDir)

	// Create test files in tools directory
	toolsTestFile := filepath.Join(toolsDir, "test.yaml")
	if err := os.WriteFile(toolsTestFile, []byte("test content from tools dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a current directory test file
	currentDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalWd) }()
	if err := os.Chdir(currentDir); err != nil {
		t.Fatal(err)
	}

	currentTestFile := filepath.Join(currentDir, "current.yaml")
	if err := os.WriteFile(currentTestFile, []byte("test content from current dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "file in current directory takes precedence",
			input:    "current.yaml",
			expected: currentTestFile,
			wantErr:  false,
		},
		{
			name:     "file in tools directory when not in current",
			input:    "test.yaml",
			expected: toolsTestFile,
			wantErr:  false,
		},
		{
			name:     "relative path without extension found in tools dir",
			input:    "test",
			expected: toolsTestFile,
			wantErr:  false,
		},
		{
			name:    "nonexistent file",
			input:   "nonexistent",
			wantErr: true,
		},
		{
			name:     "absolute path",
			input:    toolsTestFile,
			expected: toolsTestFile,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveToolsFile(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Resolve symlinks for comparison (important on macOS where /var -> /private/var)
			expectedResolved, err := filepath.EvalSymlinks(tt.expected)
			if err != nil {
				expectedResolved = tt.expected
			}
			resultResolved, err := filepath.EvalSymlinks(result)
			if err != nil {
				resultResolved = result
			}
			if resultResolved != expectedResolved {
				t.Errorf("expected %s, got %s", expectedResolved, resultResolved)
			}
		})
	}
}

func TestEnsureToolsDir(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	toolsDir := filepath.Join(tmpDir, "tools")

	// Set the tools directory environment variable
	t.Setenv(MCPShellToolsDirEnv, toolsDir)

	// Ensure the directory doesn't exist initially
	if _, err := os.Stat(toolsDir); !os.IsNotExist(err) {
		t.Fatal("Tools directory should not exist initially")
	}

	// Call EnsureToolsDir
	err := EnsureToolsDir()
	if err != nil {
		t.Fatalf("EnsureToolsDir failed: %v", err)
	}

	// Check that the directory was created
	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		t.Fatal("Tools directory was not created")
	}

	// Ensure calling it again doesn't cause an error
	err = EnsureToolsDir()
	if err != nil {
		t.Fatalf("EnsureToolsDir failed on second call: %v", err)
	}
}

func TestGetMCPShellToolsDir(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantErr bool
	}{
		{
			name:    "default directory",
			envVar:  "",
			wantErr: false,
		},
		{
			name:    "custom directory from env",
			envVar:  "/custom/tools/dir",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				t.Setenv(MCPShellToolsDirEnv, tt.envVar)
			} else {
				if err := os.Unsetenv(MCPShellToolsDirEnv); err != nil {
					t.Fatalf("failed to unset env var: %v", err)
				}
			}

			result, err := GetMCPShellToolsDir()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.envVar != "" {
				if result != tt.envVar {
					t.Errorf("expected %s, got %s", tt.envVar, result)
				}
			} else {
				// Should contain the default tools directory
				if !filepath.IsAbs(result) {
					t.Errorf("expected absolute path, got %s", result)
				}
				if filepath.Base(result) != MCPShellToolsDir {
					t.Errorf("expected path to end with %s, got %s", MCPShellToolsDir, result)
				}
			}
		})
	}
}
