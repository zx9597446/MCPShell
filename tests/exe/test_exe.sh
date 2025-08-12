#!/bin/bash
# Test script for the mcpshell exe command
# Tests the creation of a file using the create_file tool

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Test configuration
TOOLS_FILE="$SCRIPT_DIR/test_exe_config.yaml"
TEST_NAME="test_exe_command"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info_blue "Configuration file: $TOOLS_FILE"

# Generate a random test file path
TEST_FILE=$(random_tmpfile "mcpshell_test_file")
info_blue "Test file path: $TEST_FILE"
separator

# Make sure the test file doesn't exist yet
[ ! -f "$TEST_FILE" ] || fail "Test file already exists at: $TEST_FILE"

# Make sure we have the CLI binary
check_cli_exists

# Command to test
TEST_CONTENT="This is a test file created by the mcpshell exe command test."
CMD="$CLI_BIN exe --tools $TOOLS_FILE create_file filepath=$TEST_FILE content=\"$TEST_CONTENT\""
info "Executing: $CMD"

# Run the command
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

# Check the result
[ $RESULT -eq 0 ] || {
    failure "Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    cleanup_file "$TEST_FILE"
    exit 1
}

# Check if the file was created
[ -f "$TEST_FILE" ] || fail "Test file was not created at: $TEST_FILE"

# Check file content
CONTENT=$(cat "$TEST_FILE")
[ "$CONTENT" = "$TEST_CONTENT" ] || {
    failure "File content doesn't match expected content"
    info_blue "Expected: $TEST_CONTENT"
    info_blue "Actual: $CONTENT"
    cleanup_file "$TEST_FILE"
    exit 1
}

success "Test successful! The exe command correctly created the file."
info "Content: $CONTENT"

# Cleanup
cleanup_file "$TEST_FILE"

exit 0 