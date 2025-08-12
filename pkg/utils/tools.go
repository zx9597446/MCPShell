// Package utils provides utility functions for MCPShell
package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveToolsFile resolves a tools file path with the following logic:
// 1. If the file path is absolute, use it as-is
// 2. If the file path is relative, first check current directory, then tools directory
// 3. If the file doesn't have an extension, append .yaml
// 4. Return an error if the resolved file doesn't exist
func ResolveToolsFile(toolsFile string) (string, error) {
	// Add .yaml extension if no extension is present
	if filepath.Ext(toolsFile) == "" {
		toolsFile = toolsFile + ".yaml"
	}

	// If it's an absolute path, use it directly
	if filepath.IsAbs(toolsFile) {
		if _, err := os.Stat(toolsFile); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("tools file not found: %s", toolsFile)
			}
			return "", fmt.Errorf("failed to access tools file %s: %w", toolsFile, err)
		}
		return toolsFile, nil
	}

	// It's a relative path, check current directory first
	currentDirPath := toolsFile
	if _, err := os.Stat(currentDirPath); err == nil {
		// File exists in current directory
		absPath, err := filepath.Abs(currentDirPath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path for %s: %w", currentDirPath, err)
		}
		return absPath, nil
	}

	// File not found in current directory, try tools directory
	toolsDir, err := GetMCPShellToolsDir()
	if err != nil {
		return "", fmt.Errorf("failed to get tools directory: %w", err)
	}
	toolsDirPath := filepath.Join(toolsDir, toolsFile)

	if _, err := os.Stat(toolsDirPath); err == nil {
		// File exists in tools directory
		return toolsDirPath, nil
	}

	// File not found in either location
	return "", fmt.Errorf("tools file not found. Searched in:\n%s\n%s",
		currentDirPath, toolsDirPath)
}

// EnsureToolsDir creates the tools directory if it doesn't exist
func EnsureToolsDir() error {
	toolsDir, err := GetMCPShellToolsDir()
	if err != nil {
		return fmt.Errorf("failed to get tools directory: %w", err)
	}

	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory %s: %w", toolsDir, err)
	}

	return nil
}
