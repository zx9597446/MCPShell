#!/bin/bash
# Test script for the mcpshell exe command
# Tests the creation of an empty file using the create_file tool

# Set up
TEST_NAME="test_exe_empty_file_command"
CONFIG_FILE="$(dirname "$0")/test_exe_config.yaml"
TEST_FILE="/tmp/mcpshell_empty_test_file_$(date +%s).txt"

# Print test header
echo "=== Running test: $TEST_NAME ==="
echo "Configuration file: $CONFIG_FILE"
echo "Test file path: $TEST_FILE"

# Make sure the test file doesn't exist yet
if [ -f "$TEST_FILE" ]; then
    echo "ERROR: Test file already exists at: $TEST_FILE"
    exit 1
fi

# Build the exe command (only passing the filepath parameter)
EXE_CMD="go run main.go exe -c $CONFIG_FILE create_file filepath=$TEST_FILE"
echo "Executing: $EXE_CMD"

# Run the command
eval "$EXE_CMD"
RESULT=$?

# Check the result
if [ $RESULT -ne 0 ]; then
    echo "ERROR: Command execution failed with exit code: $RESULT"
    exit 1
fi

# Check if the file was created
if [ ! -f "$TEST_FILE" ]; then
    echo "ERROR: Test file was not created at: $TEST_FILE"
    exit 1
fi

# Verify the file is empty
FILE_SIZE=$(stat -f%z "$TEST_FILE")
if [ "$FILE_SIZE" -ne 0 ]; then
    echo "ERROR: Expected an empty file but file has size: $FILE_SIZE bytes"
    exit 1
fi

# Cleanup
rm -f "$TEST_FILE"
if [ -f "$TEST_FILE" ]; then
    echo "WARNING: Could not clean up test file at: $TEST_FILE"
fi

echo "Test successful! The exe command correctly created the empty file."
exit 0 