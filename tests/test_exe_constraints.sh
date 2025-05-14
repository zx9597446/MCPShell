#!/bin/bash
# Test script for the mcpshell exe command
# Tests that constraint violations are properly enforced

# Set up
TEST_NAME="test_exe_constraints"
CONFIG_FILE="$(dirname "$0")/test_exe_config.yaml"
INVALID_PATH="/etc/invalid_test_file.txt"  # Not in /tmp, should violate constraint

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
echo -e "${BLUE}Invalid path (expected to fail): $INVALID_PATH${RESET}"
echo -e "${BLUE}----------------------------------------${RESET}"

# Build the exe command with a path that violates the constraint
EXE_CMD="go run main.go exe -c $CONFIG_FILE create_file filepath=$INVALID_PATH"
echo "Executing: $EXE_CMD"

# Run the command - this should fail due to constraint violation
eval "$EXE_CMD"
RESULT=$?

# Check the result - we EXPECT this to fail
if [ $RESULT -eq 0 ]; then
    echo "ERROR: Command execution succeeded, but should have failed due to constraint violation"
    # Check if the file was actually created (it shouldn't have been)
    if [ -f "$INVALID_PATH" ]; then
        echo -e "${RED}CRITICAL ERROR: File was created at $INVALID_PATH despite constraint violation!${RESET}"
        # We don't attempt to delete this file as it would be in a system directory
        echo -e "${RED}Please manually remove this file if you have appropriate permissions.${RESET}"
        exit 2
    fi
    exit 1
fi

echo "Command failed as expected. Testing constraint violation for path containing injection character."

# Try another constraint violation - injection character
INJECTION_PATH="/tmp/test;rm -rf /"
EXE_CMD="go run main.go exe -c $CONFIG_FILE create_file filepath=\"$INJECTION_PATH\""
echo "Executing: $EXE_CMD"

# Run the command - this should fail due to constraint violation
eval "$EXE_CMD"
RESULT=$?

# Check the result - we EXPECT this to fail
if [ $RESULT -eq 0 ]; then
    echo -e "${RED}ERROR: Command execution succeeded, but should have failed due to constraint violation${RESET}"
    exit 1
fi

echo -e "${GREEN}Test successful! The exe command correctly enforced constraints.${RESET}"
exit 0 