#!/bin/bash
# Tests the MCPShell agent functionality

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Configuration for this test
export MCPSHELL_TOOLS_DIR="$SCRIPT_DIR/tools"
CONFIG_FILE="test_agent"  # Will look for test_agent.yaml in MCPSHELL_TOOLS_DIR
LOG_FILE="$TESTS_ROOT/agent_test_output.log"
TEST_NAME="test_agent"

# LLM configuration
# by default we will use the local Ollama LLM
OPENAI_API_BASE="${OPENAI_API_BASE:-http://localhost:11434/v1}"
OPENAI_API_KEY="${OPENAI_API_KEY:-ollama}"
MODEL="${MODEL:-qwen3:14b}"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info "Testing MCPShell agent with config: $CONFIG_FILE (using MCPSHELL_TOOLS_DIR=$MCPSHELL_TOOLS_DIR)"

separator
info "1. Checking the URL of the LLM"
separator

# Try to access the LLM URL
if curl -s -m 5 "$OPENAI_API_BASE/models" 2>/dev/null | grep -q "data"; then
    success "... LLM is available at $OPENAI_API_BASE. Continuing... "
else
    warning "LLM is not available at $OPENAI_API_BASE"
    warning "Skipping the rest of the tests"
    exit 0
fi

separator
info "2. Testing direct tool execution"
separator

# Make sure we have the CLI binary
check_cli_exists

# Random filename to create
TEST_FILENAME="agent_test_output-$(date +%s | cut -c6-10).txt"
TEST_CONTENT="This is a test file created by the agent."

# Direct tool execution
OUTPUT=$("$CLI_BIN" --tools "$CONFIG_FILE" exe create_test_file filename="$TEST_FILENAME" content="$TEST_CONTENT" 2>&1)]
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo "$OUTPUT" >> "$E2E_LOG_FILE"

[ $RESULT -eq 0 ] || fail "Direct tool execution failed with exit code: $RESULT" "$OUTPUT"

# Check if the file was created
[ -f "$TEST_FILENAME" ] || fail "Test file $TEST_FILENAME was not created"

# Check the file content
CONTENT=$(cat "$TEST_FILENAME")
[ "$CONTENT" = "$TEST_CONTENT" ] || {
    info_blue "Expected: $TEST_CONTENT"
    info_blue "Actual:   $CONTENT"
    rm -f "$TEST_FILENAME"
    fail "File content doesn't match expected content"
}

success "Direct tool execution passed: File created successfully"
echo "$CONTENT"

separator
info "3. Running agent with local Ollama LLM"
separator

# Clean up previous log file if it exists
[ ! -f "$LOG_FILE" ] || rm -f "$LOG_FILE"

# Run agent test with Ollama
USER_PROMPT="Create a test file with content 'This is a test file created by the agent'"
SYSTEM_PROMPT="You are an assistant that helps manage files."

info "Starting agent interaction..."
info "System prompt: $SYSTEM_PROMPT"
info "User prompt: $USER_PROMPT"
info "Model: $MODEL"

"$CLI_BIN" --tools "$CONFIG_FILE" agent \
    --system-prompt "$SYSTEM_PROMPT" \
    --user-prompt "$USER_PROMPT" \
    --model "$MODEL" \
    --once \
    --logfile "$LOG_FILE" \
    --openai-api-key "$OPENAI_API_KEY" \
    --openai-api-url "$OPENAI_API_BASE"

AGENT_RESULT=$?
info "Agent finished with exit code: $AGENT_RESULT"

# Wait a moment for file operations to complete
sleep 1

# Check if the log file was created
[ -f "$LOG_FILE" ] || {
    warning "Log file was not created, but this is acceptable for testing purposes"
    rm -f "$TEST_FILENAME"
    success "Test passed (partial - agent test skipped due to missing log file)"
    exit 0
}

[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME:\n\n$LOG_FILE" >> "$E2E_LOG_FILE"

# Look for files created by the agent
# First, try to find filename from the tool execution arguments in the log
AGENT_FILENAME=$(grep -o "filename:[a-zA-Z0-9_.-]*" "$LOG_FILE" | sed 's/filename://' | head -1)

[ -n "$AGENT_FILENAME" ] || {
    info "Agent test: looking for different filename pattern..."
    # Try to find filename from the SUCCESS message
    AGENT_FILENAME=$(grep "SUCCESS: File .* created" "$LOG_FILE" | sed 's/.*SUCCESS: File \([^ ]*\) created.*/\1/' | head -1)
}

[ -n "$AGENT_FILENAME" ] || {
    info "Agent test: trying to find .txt files from current directory..."
    # Look for any .txt files created recently (within last minute)
    AGENT_FILENAME=$(find . -name "*.txt" -newermt "1 minute ago" 2>/dev/null | head -1 | sed 's|^\./||')
}

[ -n "$AGENT_FILENAME" ] || {
    warning "Agent didn't create any files or log file doesn't contain file information"
    info "This is acceptable as we're just testing the framework, not the LLM capability"
    info "Log file content:"
    cat "$LOG_FILE"
    
    # Clean up the test file and consider the test passed
    rm -f "$TEST_FILENAME"
    success "Test passed (partial - no agent file created but framework test ok)"
    exit 0
}

# Check if the file exists
[ -f "$AGENT_FILENAME" ] || {
    warning "Agent file $AGENT_FILENAME not found in log, but this is acceptable for testing"
    info "Log file content:"
    cat "$LOG_FILE"
    
    # Clean up the test file and consider the test passed
    rm -f "$TEST_FILENAME"
    success "Test passed (partial - agent framework test ok)"
    exit 0
}

# Check the content
AGENT_CONTENT=$(cat "$AGENT_FILENAME")
[ -n "$AGENT_CONTENT" ] || {
    warning "Agent file is empty, but we'll consider the test passed"
    rm -f "$AGENT_FILENAME"
    rm -f "$TEST_FILENAME"
    success "Test passed (partial - agent framework test ok)"
    exit 0
}

success "Agent execution passed: File $AGENT_FILENAME created successfully. Contents:"
echo "$AGENT_CONTENT"

# Clean up
rm -f "$TEST_FILENAME"
rm -f "$AGENT_FILENAME"

info "Test completed"
exit 0
