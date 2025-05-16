#!/bin/bash
# Common utilities for MCPShell E2E tests

#####################################################################################

# Set script directory for relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_BIN="$SCRIPT_DIR/../MCPShell"

# ANSI color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print a separator line
separator() {
  echo -e "${YELLOW}--------------------------------------------------------${NC}"
}

# Print test case header
testcase() {
  local name="$1"
  separator
  echo -e "${YELLOW}=== Running test: $name ===${NC}"
  separator
}

# Print informational message
info() {
  echo "${*}"
}

# Print blue informational message
info_blue() {
  echo -e "${BLUE}${*}${NC}"
}

# Print a success message
success() {
  echo -e "${GREEN}✓ ${*}${NC}"
}

# Print a failure message
failure() {
  echo -e "${RED}✗ ${*}${NC}"
}

# Print a warning or skip message
warning() {
  echo -e "${YELLOW}${*}${NC}"
}

# Skip a test with a message
skip() {
  warning "${*}"
  exit 0
}

# Fail a test with an error message
# If a second argument is provided, it will be displayed as additional output
fail() {
  local message="$1"
  local output="$2"
  
  failure "$message"
  [ -n "$output" ] && echo "$output"
  exit 1
}

# Check if a command exists
command_exists() {
  command -v "$1" &> /dev/null
}

# Run a command and capture its output and exit code
run_command() {
  info "Executing: ${*}"
  OUTPUT=$(eval "${*}" 2>&1)
  RESULT=$?
  echo "$OUTPUT"
  return $RESULT
}

# Default cleanup function for test files
cleanup_file() {
  local file="$1"
  if [ -f "$file" ]; then
    rm -f "$file"
    info "Test file removed"
  fi
}

# Generate a random file name in /tmp
random_tmpfile() {
  local prefix="$1"
  echo "/tmp/${prefix}_$(date +%H%M%S%N | cut -c1-10).txt"
}

# Check if the CLI binary exists
check_cli_exists() {
  if [ ! -f "$CLI_BIN" ]; then
    info "Building CLI..."
    (cd "$SCRIPT_DIR/.." && go build)
  fi
  
  [ -f "$CLI_BIN" ] || fail "CLI binary not found at: $CLI_BIN"
} 