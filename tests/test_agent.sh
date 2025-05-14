#!/bin/bash

# This script tests the MCPShell agent with a local LLM

# Set up the environment
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="$SCRIPT_DIR/test_agent.yaml"
LOG_FILE="$PROJECT_ROOT/agent_test_output.log"

# LLM
OPENAI_API_BASE="http://localhost:11434/v1"
OPENAI_API_KEY="ollama"
MODEL="qwen3:14b"

#####################################################################################

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RESET='\033[0m'

echo -e "${BLUE}Testing MCPShell agent with config: $CONFIG_FILE${RESET}"

# Navigate to the project root
cd "$PROJECT_ROOT" || exit 1

# The build check is no longer needed since we're using go run
# However, check if the main.go file exists
if [ ! -f "main.go" ]; then
    echo -e "${RED}Error: main.go not found in project root${RESET}"
    exit 1
fi

# Clear previous log
rm -f "$LOG_FILE"

# Test parameters
FILENAME="agent_test_output-${RANDOM}.txt"
CONTENT="This is a test file created by the agent."


echo -e "${BLUE}----------------------------------------${RESET}"
echo -e "${BLUE}1. Testing direct tool execution${RESET}"
echo -e "${BLUE}----------------------------------------${RESET}"
go run main.go exe --config "$CONFIG_FILE" "create_test_file" \
    "filename=$FILENAME" \
    "content=$CONTENT"

# Check if the file was created
if [ -f "$FILENAME" ]; then
    echo -e "${GREEN}✓ Direct tool execution passed: File created successfully${RESET}"
    cat "$FILENAME"
    rm "$FILENAME"
else
    echo -e "${RED}✗ Direct tool execution failed: File not created${RESET}"
    exit 1
fi

rm -f $FILENAME
FILENAME="agent_test_output-${RANDOM}.txt"

# Run the agent...
echo -e "${BLUE}----------------------------------------${RESET}"
echo -e "${BLUE}2. Running agent with OpenAI API${RESET}"
echo -e "${BLUE}----------------------------------------${RESET}"

# Run the agent with the test config
go run main.go agent \
    --config "$CONFIG_FILE" \
    --log-level "debug" \
    --logfile "$LOG_FILE" \
    --model "$MODEL" \
    --openai-api-key "$OPENAI_API_KEY" \
    --openai-api-url "$OPENAI_API_BASE" \
    --once \
    --user-prompt "Please create a file named $FILENAME with the content: $CONTENT"
RES=$?

# Print the log file
echo -e "${BLUE}Log file content:${RESET}"
tail -n 50 "$LOG_FILE"

if [ $RES -ne 0 ]; then
    echo -e "${RED}✗ Agent execution failed with exit code: $RES${RESET}"
    exit 1
fi

# Check if the file was created
if [ -f "$FILENAME" ]; then
    echo -e "${GREEN}✓ Agent execution passed: File $FILENAME created successfully. Contents:${RESET}"
    cat "$FILENAME"
    rm "$FILENAME"
else
    echo -e "${RED}✗ Agent execution failed: File $FILENAME not created${RESET}"
    exit 1
fi

echo -e "${BLUE}Test completed${RESET}"
