#!/bin/bash

# Set script directory for relative paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test files to run
TEST_FILES=(
    "test_exe.sh"
    "test_exe_empty_file.sh"
    "test_exe_constraints.sh"
)

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo "==================================="
echo "MCPShell E2E Tests"
echo "==================================="

# Make test scripts executable
chmod +x "$SCRIPT_DIR"/*.sh

# Track overall test status
PASSED=0
FAILED=0

# Run each test
for test_file in "${TEST_FILES[@]}"; do
    echo -e "\n${YELLOW}Running test: $test_file${NC}"
    echo "-----------------------------------"
    
    # Execute the test script
    "$SCRIPT_DIR/$test_file"
    RESULT=$?
    
    # Check test result
    if [ $RESULT -eq 0 ]; then
        echo -e "${GREEN}✓ Test passed: $test_file${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗ Test failed: $test_file with exit code $RESULT${NC}"
        ((FAILED++))
    fi
    
    echo "-----------------------------------"
done

# Print summary
echo -e "\n==================================="
echo "Test Summary:"
echo -e "${GREEN}Tests passed: $PASSED${NC}"
echo -e "${RED}Tests failed: $FAILED${NC}"
echo "Total tests: $((PASSED + FAILED))"
echo "==================================="

# Return non-zero exit code if any tests failed
if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0 