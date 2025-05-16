#!/bin/bash

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Test files to run
TEST_FILES=(
    "test_agent.sh"
    "test_exe.sh"
    "test_exe_empty_file.sh"
    "test_exe_constraints.sh"
    "test_runner_docker.sh"
)

echo "==================================="
echo "MCPShell E2E Tests"
echo "==================================="

# Make test scripts executable
chmod +x "$SCRIPT_DIR"/*.sh

# Track overall test status
PASSED=0
FAILED=0

export E2E_LOG_FILE="$SCRIPT_DIR/e2e_output.log"

# Run each test
for test_file in "${TEST_FILES[@]}"; do
    echo
    warning "Running test: $test_file"
    
    # Execute the test script
    "$SCRIPT_DIR/$test_file"
    RESULT=$?
    
    # Check test result
    if [ $RESULT -eq 0 ]; then
        success "Test passed: $test_file"
        ((PASSED++))
    else
        failure "Test failed: $test_file with exit code $RESULT"
        ((FAILED++))
    fi
    
    separator
done

# Print summary
echo
echo "==================================="
echo "Test Summary:"
success "Tests passed: $PASSED"
[ $FAILED -eq 0 ] || failure "Tests failed: $FAILED"
echo "Total tests: $((PASSED + FAILED))"
echo "==================================="

# Return non-zero exit code if any tests failed
[ $FAILED -eq 0 ] || exit 1

exit 0 