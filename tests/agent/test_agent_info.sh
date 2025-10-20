#!/bin/bash
# Tests the MCPShell agent info functionality

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Configuration for this test
export MCPSHELL_TOOLS_DIR="$SCRIPT_DIR/tools"
CONFIG_FILE="test_agent"  # Will look for test_agent.yaml in MCPSHELL_TOOLS_DIR
TEST_NAME="test_agent_info"

# Model resolution:
# 1. Use MCPSHELL_AGENT_MODEL if set
# 2. Otherwise, let agent use default from config file
MODEL_FLAG=""
if [ -n "$MCPSHELL_AGENT_MODEL" ]; then
    MODEL_FLAG="--model $MCPSHELL_AGENT_MODEL"
fi

# API configuration flags - only set if explicitly provided
# This allows the model config to provide its own API URL and key
API_KEY_FLAG=""
API_URL_FLAG=""
if [ -n "$OPENAI_API_KEY" ]; then
    API_KEY_FLAG="--openai-api-key $OPENAI_API_KEY"
fi
if [ -n "$OPENAI_API_BASE" ]; then
    API_URL_FLAG="--openai-api-url $OPENAI_API_BASE"
fi

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info "Testing MCPShell agent info command"

# Make sure we have the CLI binary
check_cli_exists

separator
info "1. Testing basic agent info command (without --tools)"
separator

OUTPUT=$("$CLI_BIN" agent \
    $MODEL_FLAG \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    info --log-level none 2>&1)
RESULT=$?

[ $RESULT -eq 0 ] || fail "Agent info command failed with exit code: $RESULT" "$OUTPUT"

echo "$OUTPUT" | grep -q "Agent Configuration" || fail "Expected 'Agent Configuration' in output" "$OUTPUT"
echo "$OUTPUT" | grep -q "Orchestrator Model:" || fail "Expected 'Orchestrator Model:' in output" "$OUTPUT"

success "Basic agent info command passed (--tools is optional!)"

separator
info "2. Testing agent info with JSON output (without --tools)"
separator

OUTPUT=$("$CLI_BIN" agent \
    $MODEL_FLAG \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    info --json --log-level none 2>&1)
RESULT=$?

[ $RESULT -eq 0 ] || fail "Agent info --json command failed with exit code: $RESULT" "$OUTPUT"

# Verify JSON output is valid (tools_file should not be present when --tools is not used)
echo "$OUTPUT" | grep -q '"orchestrator":' || fail "Expected 'orchestrator' in JSON output" "$OUTPUT"
echo "$OUTPUT" | grep -q '"tool_runner":' || fail "Expected 'tool_runner' in JSON output" "$OUTPUT"

success "Agent info --json command passed"

separator
info "3. Testing agent info with --include-prompts"
separator

OUTPUT=$("$CLI_BIN" agent \
    $MODEL_FLAG \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    --system-prompt "Test system prompt" \
    info --include-prompts --log-level none 2>&1)
RESULT=$?

[ $RESULT -eq 0 ] || fail "Agent info --include-prompts command failed with exit code: $RESULT" "$OUTPUT"

echo "$OUTPUT" | grep -q "Prompts:" || fail "Expected 'Prompts:' in output" "$OUTPUT"
echo "$OUTPUT" | grep -q "Test system prompt" || fail "Expected custom system prompt in output" "$OUTPUT"

success "Agent info --include-prompts command passed"

separator
info "4. Testing agent info with --include-prompts and --json"
separator

OUTPUT=$("$CLI_BIN" agent \
    $MODEL_FLAG \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    --system-prompt "Test system prompt" \
    info --include-prompts --json --log-level none 2>&1)
RESULT=$?

[ $RESULT -eq 0 ] || fail "Agent info --include-prompts --json command failed with exit code: $RESULT" "$OUTPUT"

echo "$OUTPUT" | grep -q '"prompts":' || fail "Expected 'prompts' in JSON output" "$OUTPUT"
echo "$OUTPUT" | grep -q '"system":' || fail "Expected 'system' prompts in JSON output" "$OUTPUT"

success "Agent info --include-prompts --json command passed"

separator
info "5. Testing agent info --check (LLM connectivity test)"
separator

info "Checking if LLM is available..."
OUTPUT=$("$CLI_BIN" agent \
    $MODEL_FLAG \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    info --check --log-level none 2>&1)
RESULT=$?

if [ $RESULT -eq 0 ]; then
    success "LLM connectivity check passed - LLM is available and responding"
    echo "$OUTPUT" | grep -q "Connected" || warning "Expected 'Connected' in output"
    echo "$OUTPUT" | grep -q "Response:" || warning "Expected 'Response:' time in output"
else
    warning "LLM connectivity check failed - this is acceptable if no LLM is running"
    warning "Output: $OUTPUT"
fi

separator
info "6. Testing agent info --check with --json"
separator

OUTPUT=$("$CLI_BIN" agent \
    $MODEL_FLAG \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    info --check --json --log-level none 2>&1)
RESULT=$?

# JSON output should always be valid even if check fails
echo "$OUTPUT" | grep -q '"check":' || fail "Expected 'check' in JSON output" "$OUTPUT"
echo "$OUTPUT" | grep -q '"success":' || fail "Expected 'success' in check JSON output" "$OUTPUT"

if [ $RESULT -eq 0 ]; then
    echo "$OUTPUT" | grep -q '"success": true' || fail "Expected 'success: true' in JSON"
    success "LLM connectivity check with JSON passed - LLM is available"
else
    echo "$OUTPUT" | grep -q '"success": false' || fail "Expected 'success: false' in JSON"
    warning "LLM connectivity check with JSON failed (no LLM available) - but JSON output is valid"
fi

separator
info "7. Testing agent info with model override"
separator

# Test with a different model name
OUTPUT=$("$CLI_BIN" agent \
    --model "different-model" \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    info --json --log-level none 2>&1)
RESULT=$?

[ $RESULT -eq 0 ] || fail "Agent info with model override failed with exit code: $RESULT" "$OUTPUT"

echo "$OUTPUT" | grep -q '"model": "different-model"' || fail "Expected overridden model in output" "$OUTPUT"

success "Agent info with model override passed"

separator
success "All agent info tests completed successfully!"
exit 0
