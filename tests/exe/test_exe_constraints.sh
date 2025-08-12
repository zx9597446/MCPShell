#!/bin/bash
# Test script for the mcpshell exe command
# Tests that constraint violations are properly enforced

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

# Test configuration
CONFIG_FILE="$SCRIPT_DIR/test_exe_config.yaml"
TEST_NAME="test_exe_constraints"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info_blue "Configuration file: $CONFIG_FILE"

# Make sure we have the CLI binary
check_cli_exists

# Test a path that would fail constraint checks
INVALID_PATH="/etc/invalid_test_file.txt"
info "Invalid path (expected to fail): $INVALID_PATH"
separator

# Command to test with invalid path
CMD="$CLI_BIN exe --tools $CONFIG_FILE create_file filepath=$INVALID_PATH"

info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

# This should fail, so we expect a non-zero exit code
[ $RESULT -ne 0 ] || fail "Command unexpectedly succeeded with invalid path!" "$OUTPUT"

success "Command failed as expected. Testing constraint violation for path containing injection character."

# Test path with shell injection attempt
INJECTION_PATH="/tmp/test;rm -rf /"
CMD="$CLI_BIN exe --tools $CONFIG_FILE create_file filepath=\"$INJECTION_PATH\""

info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?

# This should fail too
[ $RESULT -ne 0 ] || fail "Command unexpectedly succeeded with injection character!" "$OUTPUT"

success "Test successful! The exe command correctly enforced constraints."
exit 0 