#!/bin/bash
# Test script for the mcpshell timeout functionality
# Tests that commands respect the configured timeout values

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Test configuration
TOOLS_FILE="$SCRIPT_DIR/test_exe_timeout.yaml"
TEST_NAME="test_exe_timeout"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info_blue "Configuration file: $TOOLS_FILE"
separator

# Make sure we have the CLI binary
check_cli_exists

#####################################################################################
info "Test 1: Quick command (should complete successfully)"
separator

CMD="$CLI_BIN exe --tools $TOOLS_FILE quick_command"
info "Executing: $CMD"

START_TIME=$(date +%s)
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (quick_command):\n\n$OUTPUT" >> "$E2E_LOG_FILE"

[ $RESULT -eq 0 ] || fail "Quick command failed with exit code: $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Quick command completed successfully" || fail "Expected output not found" "$OUTPUT"

success "Quick command completed in ${ELAPSED}s"

#####################################################################################
separator
info "Test 2: Slow command with short timeout (should timeout after ~2s)"
separator

CMD="$CLI_BIN exe --tools $TOOLS_FILE slow_command_short_timeout"
info "Executing: $CMD"

START_TIME=$(date +%s)
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (slow_command_short_timeout):\n\n$OUTPUT" >> "$E2E_LOG_FILE"

# Command should fail due to timeout
[ $RESULT -ne 0 ] || fail "Slow command should have timed out but succeeded" "$OUTPUT"

# Should complete in roughly 2-3 seconds (not the full 10 seconds)
if [ $ELAPSED -ge 8 ]; then
    fail "Command took ${ELAPSED}s but should have timed out after ~2s" "$OUTPUT"
fi

success "Slow command correctly timed out after ${ELAPSED}s (expected ~2s)"

#####################################################################################
separator
info "Test 3: Command with long timeout (should complete successfully)"
separator

CMD="$CLI_BIN exe --tools $TOOLS_FILE command_with_long_timeout"
info "Executing: $CMD"

START_TIME=$(date +%s)
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (command_with_long_timeout):\n\n$OUTPUT" >> "$E2E_LOG_FILE"

[ $RESULT -eq 0 ] || fail "Command with long timeout failed with exit code: $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Completed successfully" || fail "Expected output not found" "$OUTPUT"

success "Command with long timeout completed in ${ELAPSED}s"

#####################################################################################
separator
info "Test 4: Command without timeout (should use default)"
separator

CMD="$CLI_BIN exe --tools $TOOLS_FILE no_timeout_command"
info "Executing: $CMD"

START_TIME=$(date +%s)
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME (no_timeout_command):\n\n$OUTPUT" >> "$E2E_LOG_FILE"

[ $RESULT -eq 0 ] || fail "Command without timeout failed with exit code: $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Done" || fail "Expected output not found" "$OUTPUT"

success "Command without timeout completed in ${ELAPSED}s"

#####################################################################################
separator
success "All timeout tests passed!"
exit 0
