#!/bin/bash
# Test script for disk-diagnostics-ro.yaml example
# Tests basic disk diagnostic tools

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Test configuration
TOOLS_FILE="$SCRIPT_DIR/../../examples/disk-diagnostics-ro.yaml"
TEST_NAME="test_disk_diagnostics_ro"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info_blue "Configuration file: $TOOLS_FILE"
separator

# Make sure we have the CLI binary
check_cli_exists

#####################################################################################
# Test 1: storage_overview tool

info "Test 1: Testing storage_overview tool"
CMD="$CLI_BIN exe --tools $TOOLS_FILE storage_overview"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 1:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

if [ $RESULT -ne 0 ]; then
    failure "Test 1 failed: Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
fi

# Check if output contains expected filesystem information
if echo "$OUTPUT" | grep -q "Filesystem"; then
    success "Test 1 passed: storage_overview tool works correctly"
else
    failure "Test 1 failed: Output doesn't contain expected filesystem information"
    echo "$OUTPUT"
    exit 1
fi

separator

success "All tests passed for disk-diagnostics-ro.yaml!"
exit 0

