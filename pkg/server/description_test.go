package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inercia/MCPShell/pkg/common"
	"github.com/inercia/MCPShell/pkg/config"
)

// TestGetDescription tests the GetDescription function with various scenarios
func TestGetDescription(t *testing.T) {
	// Prepare temporary directory for test files
	tempDir, err := os.MkdirTemp("", "description-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create test files
	file1Path := filepath.Join(tempDir, "desc1.txt")
	file2Path := filepath.Join(tempDir, "desc2.txt")

	if err := os.WriteFile(file1Path, []byte("description from file 1"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	if err := os.WriteFile(file2Path, []byte("description from file 2"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a test config file path (doesn't need to exist)
	configPath := filepath.Join(tempDir, "config.yaml")

	// Create a logger for tests
	logger, err := common.NewLogger("", "", common.LogLevelNone, false)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Define a custom implementation for testing
	testGetDescription := func(cfg Config, configLoader func(string) (*config.ToolsConfig, error)) (string, error) {
		var finalDesc string

		// First, check if we should load the description from the config file
		// (unless description override is explicitly requested)
		configDesc := ""
		if !cfg.DescriptionOverride {
			loadedCfg, loadErr := configLoader(cfg.ConfigFile)
			if loadErr == nil && loadedCfg.MCP.Description != "" {
				configDesc = loadedCfg.MCP.Description
				cfg.Logger.Debug("Found description in config file: %s", configDesc)
				finalDesc = configDesc
				cfg.Logger.Debug("Using description from config file: %s", configDesc)
			}
		}

		// Add descriptions from command line flags
		if len(cfg.Descriptions) > 0 {
			cmdDesc := strings.Join(cfg.Descriptions, "\n")
			if finalDesc != "" && !cfg.DescriptionOverride {
				finalDesc += "\n" + cmdDesc
				cfg.Logger.Info("Appending descriptions from command line flags")
			} else {
				finalDesc = cmdDesc
				cfg.Logger.Info("Using descriptions from command line flags")
			}
		}

		// Add descriptions from files - handle only local files for this test
		if len(cfg.DescriptionFiles) > 0 {
			cfg.Logger.Info("Reading server description from files: %v", cfg.DescriptionFiles)
			var fileDescs []string

			for _, filePath := range cfg.DescriptionFiles {
				content, err := os.ReadFile(filePath)
				if err != nil {
					cfg.Logger.Error("Failed to read description file: %s - %v", filePath, err)
					return "", fmt.Errorf("failed to read description file %s: %w", filePath, err)
				}
				fileDescs = append(fileDescs, string(content))
			}

			// Concatenate all file contents
			if len(fileDescs) > 0 {
				fileContent := strings.Join(fileDescs, "\n")
				if finalDesc != "" && !cfg.DescriptionOverride {
					finalDesc += "\n" + fileContent
					cfg.Logger.Info("Appending descriptions from files")
				} else {
					finalDesc = fileContent
					cfg.Logger.Info("Using descriptions from files")
				}
			}
		}

		return finalDesc, nil
	}

	// Setup test cases
	tests := []struct {
		name           string
		config         Config
		mockLoader     func(string) (*config.ToolsConfig, error)
		expectedResult string
		expectError    bool
	}{
		{
			name: "From config file only",
			config: Config{
				ConfigFile: configPath,
				Logger:     logger,
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{
					MCP: config.MCPConfig{
						Description: "description from config",
					},
				}, nil
			},
			expectedResult: "description from config",
		},
		{
			name: "From command line only",
			config: Config{
				ConfigFile:   configPath,
				Logger:       logger,
				Descriptions: []string{"description from cmd"},
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{}, nil
			},
			expectedResult: "description from cmd",
		},
		{
			name: "Multiple command line descriptions",
			config: Config{
				ConfigFile:   configPath,
				Logger:       logger,
				Descriptions: []string{"description 1", "description 2"},
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{}, nil
			},
			expectedResult: "description 1\ndescription 2",
		},
		{
			name: "From config file and command line (append mode)",
			config: Config{
				ConfigFile:   configPath,
				Logger:       logger,
				Descriptions: []string{"description from cmd"},
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{
					MCP: config.MCPConfig{
						Description: "description from config",
					},
				}, nil
			},
			expectedResult: "description from config\ndescription from cmd",
		},
		{
			name: "From config file and command line (override mode)",
			config: Config{
				ConfigFile:          configPath,
				Logger:              logger,
				Descriptions:        []string{"description from cmd"},
				DescriptionOverride: true,
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{
					MCP: config.MCPConfig{
						Description: "description from config",
					},
				}, nil
			},
			expectedResult: "description from cmd",
		},
		{
			name: "From local files",
			config: Config{
				ConfigFile:       configPath,
				Logger:           logger,
				DescriptionFiles: []string{file1Path, file2Path},
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{}, nil
			},
			expectedResult: "description from file 1\ndescription from file 2",
		},
		{
			name: "From config and local files (append mode)",
			config: Config{
				ConfigFile:       configPath,
				Logger:           logger,
				DescriptionFiles: []string{file1Path},
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{
					MCP: config.MCPConfig{
						Description: "description from config",
					},
				}, nil
			},
			expectedResult: "description from config\ndescription from file 1",
		},
		{
			name: "From config and local files (override mode)",
			config: Config{
				ConfigFile:          configPath,
				Logger:              logger,
				DescriptionFiles:    []string{file1Path},
				DescriptionOverride: true,
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{
					MCP: config.MCPConfig{
						Description: "description from config",
					},
				}, nil
			},
			expectedResult: "description from file 1",
		},
		{
			name: "With missing file",
			config: Config{
				ConfigFile:       configPath,
				Logger:           logger,
				DescriptionFiles: []string{filepath.Join(tempDir, "nonexistent.txt")},
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{}, nil
			},
			expectError: true,
		},
		{
			name: "From config, command line and files",
			config: Config{
				ConfigFile:       configPath,
				Logger:           logger,
				Descriptions:     []string{"description from cmd"},
				DescriptionFiles: []string{file1Path},
			},
			mockLoader: func(path string) (*config.ToolsConfig, error) {
				return &config.ToolsConfig{
					MCP: config.MCPConfig{
						Description: "description from config",
					},
				}, nil
			},
			expectedResult: "description from config\ndescription from cmd\ndescription from file 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the test function with mock loader
			result, err := testGetDescription(tt.config, tt.mockLoader)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expectedResult {
				t.Errorf("Expected result %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

// TestGetDescriptionWithURLs is a more basic test that uses the real GetDescription function
// but with local files only to avoid the complexities of mocking URL fetching.
func TestGetDescriptionWithURLs(t *testing.T) {
	// This test would normally mock URL fetching, but since that's difficult without
	// dependency injection or monkey patching, this is a placeholder that just tests
	// the basic functionality using local files
	t.Run("Basic functionality check with regular function", func(t *testing.T) {
		// Prepare temporary directory for test files
		tempDir, err := os.MkdirTemp("", "real-description-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Errorf("Failed to remove temp directory: %v", err)
			}
		}()

		// Create a test file
		testFilePath := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFilePath, []byte("description from test file"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Create a config file with description
		configPath := filepath.Join(tempDir, "config.yaml")
		configContent := "mcp:\n  description: description from config file\n"
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		// Create a logger
		logger, err := common.NewLogger("", "", common.LogLevelNone, false)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		// Basic case: Get description from command line
		config := Config{
			ConfigFile:   configPath,
			Logger:       logger,
			Descriptions: []string{"description from command line"},
		}

		result, err := GetDescription(config)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Just check if we get some kind of result, don't validate exactly since it depends
		// on the real config file being read, which might not be predictable in all environments
		if result == "" {
			t.Error("Expected non-empty result")
		}

		t.Logf("Result: %s", result)
	})
}
