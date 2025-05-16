#!/bin/bash
# Test script for the mcpshell exe command
# Tests the creation of an empty file using the create_file tool

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Test configuration
CONFIG_FILE="$SCRIPT_DIR/test_exe_config.yaml"
TEST_NAME="test_exe_empty_file_command"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info_blue "Configuration file: $CONFIG_FILE"

# Generate a random test file path
TEST_FILE=$(random_tmpfile "mcpshell_empty_test_file")
info_blue "Test file path: $TEST_FILE"
separator

# Make sure we have the CLI binary
check_cli_exists

# Command to test
CMD="$CLI_BIN exe -c $CONFIG_FILE create_file filepath=$TEST_FILE"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

[ $RESULT -eq 0 ] || {
    failure "Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
}

# Verify the file was created
[ -f "$TEST_FILE" ] || fail "Test file was not created at $TEST_FILE"

# Verify the content of the file
CONTENT=$(cat "$TEST_FILE")
DEFAULT_CONTENT="Default content for an empty file."

[ "$CONTENT" = "$DEFAULT_CONTENT" ] || {
    failure "File content does not match expected default content"
    info_blue "Expected: $DEFAULT_CONTENT"
    info_blue "Actual:   $CONTENT"
    cleanup_file "$TEST_FILE"
    exit 1
}

success "Empty file test passed, default content was used"
info "Content: $CONTENT"

# Clean up
cleanup_file "$TEST_FILE"

exit 0
