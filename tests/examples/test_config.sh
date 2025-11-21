#!/bin/bash
# Test script for config.yaml example
# Tests basic utility tools like hello_world, calculator, and number_validator

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Test configuration
TOOLS_FILE="$SCRIPT_DIR/../../examples/config.yaml"
TEST_NAME="test_config"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info_blue "Configuration file: $TOOLS_FILE"
separator

# Make sure we have the CLI binary
check_cli_exists

#####################################################################################
# Test 1: hello_world tool

info "Test 1: Testing hello_world tool"
CMD="$CLI_BIN exe --tools $TOOLS_FILE hello_world name=World"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 1:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

if [ $RESULT -ne 0 ]; then
    failure "Test 1 failed: Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
fi

# Check if output contains expected greeting
if echo "$OUTPUT" | grep -q "Hello, World!"; then
    success "Test 1 passed: hello_world tool works correctly"
else
    failure "Test 1 failed: Output doesn't contain expected greeting"
    echo "$OUTPUT"
    exit 1
fi

separator

#####################################################################################
# Test 2: calculator tool

info "Test 2: Testing calculator tool"
CMD="$CLI_BIN exe --tools $TOOLS_FILE calculator expression='2+2'"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 2:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

if [ $RESULT -ne 0 ]; then
    failure "Test 2 failed: Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
fi

# Check if output contains the result
if echo "$OUTPUT" | grep -q "4"; then
    success "Test 2 passed: calculator tool works correctly"
else
    failure "Test 2 failed: Output doesn't contain expected result"
    echo "$OUTPUT"
    exit 1
fi

separator

#####################################################################################
# Test 3: number_validator tool with square operation

info "Test 3: Testing number_validator tool (square operation)"
CMD="$CLI_BIN exe --tools $TOOLS_FILE number_validator value=5 operation=square"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 3:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

if [ $RESULT -ne 0 ]; then
    failure "Test 3 failed: Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
fi

# Check if output contains the result (5*5=25)
if echo "$OUTPUT" | grep -q "25"; then
    success "Test 3 passed: number_validator tool works correctly"
else
    failure "Test 3 failed: Output doesn't contain expected result"
    echo "$OUTPUT"
    exit 1
fi

separator

#####################################################################################
# Test 4: Test constraint validation (name length limit)

info "Test 4: Testing constraint validation (name too long)"
LONG_NAME=$(printf 'A%.0s' {1..150})  # Create a 150-character string
CMD="$CLI_BIN exe --tools $TOOLS_FILE hello_world name='$LONG_NAME'"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 4:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

# This should fail due to constraint violation
if [ $RESULT -eq 0 ]; then
    failure "Test 4 failed: Command should have been blocked by constraints"
    echo "$OUTPUT"
    exit 1
fi

# Check if output mentions constraint violation
if echo "$OUTPUT" | grep -q "constraint"; then
    success "Test 4 passed: Name length constraint correctly enforced"
else
    failure "Test 4 failed: Expected constraint violation message"
    echo "$OUTPUT"
    exit 1
fi

separator

#####################################################################################
# Test 5: Test secure_shell with whitelisted command

info "Test 5: Testing secure_shell with whitelisted command (pwd)"
CMD="$CLI_BIN exe --tools $TOOLS_FILE secure_shell command=pwd"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 5:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

if [ $RESULT -ne 0 ]; then
    failure "Test 5 failed: Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
fi

# Check if output contains a path
if echo "$OUTPUT" | grep -q "/"; then
    success "Test 5 passed: secure_shell tool works correctly"
else
    failure "Test 5 failed: Output doesn't contain expected path"
    echo "$OUTPUT"
    exit 1
fi

separator

success "All tests passed for config.yaml!"
exit 0

