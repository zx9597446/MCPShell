package common

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DownloadTimeout is the timeout for downloading content from a URL
const DownloadTimeout = 10 * time.Second

// List of supported text content types
var SupportedContentTypes = []string{
	"text/",                  // All text/* types
	"application/json",       // JSON
	"application/xml",        // XML
	"application/yaml",       // YAML
	"application/x-yaml",     // Alternative YAML
	"application/javascript", // JavaScript
	"application/ecmascript", // ECMAScript
	"application/markdown",   // Markdown
	"application/x-markdown", // Alternative Markdown
}

// fetchURLContent downloads content from a URL and verifies it's a text format
// Returns the content as a byte slice if successful, or an error if download fails
// or if the content is not a supported text format.
func FetchURLText(url string) ([]byte, error) {
	// Create a new HTTP client with a timeout
	client := &http.Client{
		Timeout: DownloadTimeout,
	}

	// Send a GET request to the URL
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// If we already have an error, don't overwrite it
			if err == nil {
				err = fmt.Errorf("failed to close response body: %w", closeErr)
			}
		}
	}()

	// Check response status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request returned non-success status: %d %s",
			resp.StatusCode, resp.Status)
	}

	// Check content type to ensure it's a text format
	contentType := resp.Header.Get("Content-Type")
	if !isTextContentType(contentType) {
		return nil, fmt.Errorf("unsupported content type: %s - only text formats are supported", contentType)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// isTextContentType checks if a Content-Type header represents a text format
func isTextContentType(contentType string) bool {
	// Remove any parameters (like charset)
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	// Check if content type is in the supported list
	for _, supported := range SupportedContentTypes {
		if supported == contentType || (strings.HasSuffix(supported, "/") && strings.HasPrefix(contentType, supported)) {
			return true
		}
	}

	return false
}
