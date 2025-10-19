#!/bin/bash

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

# Configuration file for this test
CONFIG_FILE="$SCRIPT_DIR/test_runner_sandbox_exec.yaml"
TEST_NAME="test_runner_sandbox_exec"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info "Testing sandbox-exec runner with config: $CONFIG_FILE"

separator
info "1. Checking if running on macOS"
separator

# Check if we're on macOS
OS_TYPE=$(uname -s)
if [ "$OS_TYPE" != "Darwin" ]; then
    skip "sandbox-exec is only available on macOS (detected: $OS_TYPE), skipping test"
fi

success "Running on macOS ($OS_TYPE), proceeding with tests"

separator
info "2. Checking if sandbox-exec is installed"
separator

# Check if sandbox-exec is installed
command_exists sandbox-exec || skip "sandbox-exec not found, skipping test"
success "sandbox-exec is available, proceeding with tests"

# Make sure we have the CLI binary
check_cli_exists

separator
info "3. Simple hello world in sandbox-exec"
separator

CMD="$CLI_BIN --tools $CONFIG_FILE exe sandbox_hello"
info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (sandbox_hello):\n\n$OUTPUT" >> "$E2E_LOG_FILE"
[ $RESULT -eq 0 ] || fail "Test failed with exit code $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Hello from sandbox-exec" || fail "Test failed: Expected output not found" "$OUTPUT"

success "Test passed"

separator
info "4. Reading files with proper permissions"
separator

CMD="$CLI_BIN --tools $CONFIG_FILE exe sandbox_read_file"
info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (sandbox_read_file):\n\n$OUTPUT" >> "$E2E_LOG_FILE"

[ $RESULT -eq 0 ] || fail "Test failed with exit code $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Can read /etc files" || fail "Test failed: Expected output not found" "$OUTPUT"

success "Test passed"

separator
info "5. Writing to /tmp with proper permissions"
separator

RANDOM_FILE="sandbox_test_$(date +%s).txt"
CMD="$CLI_BIN --tools $CONFIG_FILE exe sandbox_with_write filename=$RANDOM_FILE"
info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (sandbox_with_write):\n\n$OUTPUT" >> "$E2E_LOG_FILE"
[ $RESULT -eq 0 ] || fail "Test failed with exit code $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Test content" || fail "Test failed: Expected output not found" "$OUTPUT"

success "Test passed"

separator
info "6. Timeout functionality with sandbox-exec"
separator

CMD="$CLI_BIN --tools $CONFIG_FILE exe sandbox_with_timeout"
info "Executing: $CMD"

START_TIME=$(date +%s)
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (sandbox_with_timeout):\n\n$OUTPUT" >> "$E2E_LOG_FILE"

# Command should fail due to timeout
[ $RESULT -ne 0 ] || fail "Command should have timed out but succeeded" "$OUTPUT"

# Should complete in roughly 2-3 seconds (not the full 10 seconds)
if [ $ELAPSED -ge 8 ]; then
    fail "Command took ${ELAPSED}s but should have timed out after ~2s" "$OUTPUT"
fi

success "Timeout test passed (completed in ${ELAPSED}s, expected ~2s)"

separator
info "7. Network access is properly blocked"
separator

# Only run this test if curl is available
if command_exists curl; then
    CMD="$CLI_BIN --tools $CONFIG_FILE exe sandbox_network_blocked"
    info "Executing: $CMD"
    OUTPUT=$(eval "$CMD" 2>&1)
    RESULT=$?
    [ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (sandbox_network_blocked):\n\n$OUTPUT" >> "$E2E_LOG_FILE"
    [ $RESULT -eq 0 ] || fail "Test failed with exit code $RESULT" "$OUTPUT"
    echo "$OUTPUT" | grep -q "Network access blocked as expected" || fail "Test failed: Expected output not found" "$OUTPUT"
    
    success "Test passed"
else
    warning "curl not available, skipping network test"
fi

echo
success "All sandbox-exec runner tests passed!"
exit 0
