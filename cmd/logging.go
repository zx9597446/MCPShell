package root

import (
	"fmt"
	"os"

	"github.com/inercia/mcp-cli-adapter/pkg/common"
)

// setupLogger initializes the logger based on command-line flags
func setupLogger() (*common.Logger, error) {
	// Determine log level from flag
	level := common.LogLevelFromString(logLevel)

	// Create logger
	return common.NewLogger("[mcp-cli-adapter] ", logFile, level, true)
}

// GetLogger returns the global application logger.
// If the logger hasn't been initialized yet, it returns a default stderr logger.
func GetLogger() *common.Logger {
	if globalLogger == nil {
		// Create a default stderr logger at info level
		logger, err := common.NewLogger("[mcp-cli-adapter] ", "", common.LogLevelInfo, false)
		if err != nil {
			// If we can't even create a basic logger, just return a minimal one
			fmt.Fprintf(os.Stderr, "Error creating default logger: %v\n", err)
			minimalLogger, _ := common.NewLogger("[MCP-CLI] ", "", common.LogLevelError, false)
			return minimalLogger
		}
		return logger
	}
	return globalLogger
}

// SetLogger sets the global application logger
func SetLogger(logger *common.Logger) {
	globalLogger = logger
}
