#!/bin/bash
# Test script for the mcpshell exe command
# Tests the creation of a file using the create_file tool

# Set up
TEST_NAME="test_exe_command"
CONFIG_FILE="$(dirname "$0")/test_exe_config.yaml"
TEST_FILE="/tmp/mcpshell_test_file_$(date +%s).txt"
TEST_CONTENT="This is a test file created by the mcpshell exe command test."

#####################################################################################

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RESET='\033[0m'

# Print test header
echo -e "${BLUE}----------------------------------------${RESET}"
echo -e "${BLUE}=== Running test: $TEST_NAME ===${RESET}"
echo -e "${BLUE}----------------------------------------${RESET}"
echo -e "${BLUE}Configuration file: $CONFIG_FILE${RESET}"
echo -e "${BLUE}Test file path: $TEST_FILE${RESET}"
echo -e "${BLUE}----------------------------------------${RESET}"

# Make sure the test file doesn't exist yet
if [ -f "$TEST_FILE" ]; then
    echo "ERROR: Test file already exists at: $TEST_FILE"
    exit 1
fi

# Build the exe command
EXE_CMD="go run main.go exe -c $CONFIG_FILE create_file filepath=$TEST_FILE content=\"$TEST_CONTENT\""
echo "Executing: $EXE_CMD"

# Run the command
eval "$EXE_CMD"
RESULT=$?

# Check the result
if [ $RESULT -ne 0 ]; then
    echo -e "${RED}ERROR: Command execution failed with exit code: $RESULT${RESET}"
    exit 1
fi

# Check if the file was created
if [ ! -f "$TEST_FILE" ]; then
    echo -e "${RED}ERROR: Test file was not created at: $TEST_FILE${RESET}"
    exit 1
fi

# Check file content
FILE_CONTENT=$(cat "$TEST_FILE")
if [ "$FILE_CONTENT" != "$TEST_CONTENT" ]; then
    echo -e "${RED}ERROR: File content doesn't match expected content${RESET}"
    echo -e "${BLUE}Expected: $TEST_CONTENT${RESET}"
    echo -e "${BLUE}Actual: $FILE_CONTENT${RESET}"
    exit 1
fi

# Cleanup
rm -f "$TEST_FILE"
if [ -f "$TEST_FILE" ]; then
    echo -e "${RED}WARNING: Could not clean up test file at: $TEST_FILE${RESET}"
fi

echo -e "${GREEN}Test successful! The exe command correctly created the file.${RESET}"
exit 0 