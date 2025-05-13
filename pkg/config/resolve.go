package config

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/inercia/MCPShell/pkg/common"
)

// ResolveConfigPath tries to resolve the configuration file path.
// If the path is a URL, it downloads the file to a temporary location.
// The function returns the local path to the configuration file and a cleanup function
// that should be deferred to remove any temporary files.
func ResolveConfigPath(configPath string, logger *common.Logger) (string, func(), error) {
	// Default no-op cleanup function
	noopCleanup := func() {}

	// Return early if config path is empty
	if configPath == "" {
		return "", noopCleanup, fmt.Errorf("configuration file path is empty")
	}

	// Check if the configPath is a URL
	parsedURL, err := url.Parse(configPath)
	if err != nil {
		return "", noopCleanup, fmt.Errorf("invalid configuration path: %w", err)
	}

	// If it's not a URL, just verify the file exists locally
	if parsedURL.Scheme == "" || parsedURL.Scheme == "file" {
		localPath := configPath
		if parsedURL.Scheme == "file" {
			localPath = parsedURL.Path
		}

		// Check if the file exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return "", noopCleanup, fmt.Errorf("configuration file does not exist: %s", localPath)
		}

		logger.Info("Using local configuration file: %s", localPath)
		return localPath, noopCleanup, nil
	}

	// If it's a remote URL, download it
	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		logger.Info("Downloading configuration from URL: %s", configPath)

		// Create a temporary file
		tmpDir := os.TempDir()
		tmpFile, err := os.CreateTemp(tmpDir, "mcp-config-*.yaml")
		if err != nil {
			return "", noopCleanup, fmt.Errorf("failed to create temporary file: %w", err)
		}
		tmpFilePath := tmpFile.Name()

		// Create cleanup function for the temporary file
		cleanup := func() {
			tmpFile.Close()
			os.Remove(tmpFilePath)
			logger.Debug("Cleaned up temporary configuration file: %s", tmpFilePath)
		}

		// Download the file
		resp, err := http.Get(configPath)
		if err != nil {
			cleanup()
			return "", noopCleanup, fmt.Errorf("failed to download configuration: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			cleanup()
			return "", noopCleanup, fmt.Errorf("failed to download configuration, status code: %d", resp.StatusCode)
		}

		// Copy the response body to the temporary file
		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			cleanup()
			return "", noopCleanup, fmt.Errorf("failed to write configuration to temporary file: %w", err)
		}

		// Close the file after writing
		if err = tmpFile.Close(); err != nil {
			cleanup()
			return "", noopCleanup, fmt.Errorf("failed to close temporary file: %w", err)
		}

		logger.Info("Downloaded configuration to temporary file: %s", tmpFilePath)
		return tmpFilePath, cleanup, nil
	}

	return "", noopCleanup, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
}
