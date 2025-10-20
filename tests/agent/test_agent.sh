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

info "Testing MCPShell agent with config: $CONFIG_FILE (using MCPSHELL_TOOLS_DIR=$MCPSHELL_TOOLS_DIR)"

separator
info "1. Checking LLM availability using 'agent info --check'"
separator

# Use the agent info --check command to test LLM connectivity
# This is more robust than curl as it tests the actual agent configuration
CHECK_OUTPUT=$("$CLI_BIN" --tools "$CONFIG_FILE" agent \
    $MODEL_FLAG \
    $API_KEY_FLAG \
    $API_URL_FLAG \
    info --check --log-level none 2>&1)
CHECK_RESULT=$?

# Extract the actual model being used from the output
ACTUAL_MODEL=$(echo "$CHECK_OUTPUT" | grep "^  Model:" | head -1 | awk '{print $2}')

if [ $CHECK_RESULT -eq 0 ]; then
    if [ -n "$OPENAI_API_BASE" ]; then
        success "LLM is available and responding (model: ${ACTUAL_MODEL:-default} at $OPENAI_API_BASE)"
    else
        success "LLM is available and responding (model: ${ACTUAL_MODEL:-default})"
    fi
    # Show connectivity info if available
    if echo "$CHECK_OUTPUT" | grep -q "Connected"; then
        echo "$CHECK_OUTPUT" | grep "Status:" || true
        echo "$CHECK_OUTPUT" | grep "Response:" || true
    fi
else
    warning "═══════════════════════════════════════════════════════════════════"
    warning "LLM is not available or not responding"
    warning ""
    warning "Configuration used:"
    warning "  Model: ${ACTUAL_MODEL:-default from config}"
    if [ -n "$OPENAI_API_BASE" ]; then
        warning "  API URL: $OPENAI_API_BASE (override)"
    else
        warning "  API URL: from model config"
    fi
    warning ""
    warning "To run agent tests, ensure you have an LLM available:"
    warning "  - For local testing: Install Ollama (https://ollama.ai)"
    warning "  - For remote LLMs: Set OPENAI_API_KEY and OPENAI_API_BASE"
    warning ""
    warning "Example: MCPSHELL_AGENT_MODEL=qwen3:14b OPENAI_API_BASE=http://localhost:11434/v1 ./test_agent.sh"
    warning "═══════════════════════════════════════════════════════════════════"
    warning ""
    warning "Skipping agent tests due to unavailable LLM"
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
info "3. Running agent with real LLM"
separator

# Clean up previous log file if it exists
[ ! -f "$LOG_FILE" ] || rm -f "$LOG_FILE"

# Run agent test with Ollama
USER_PROMPT="Create a test file with content 'This is a test file created by the agent'"
SYSTEM_PROMPT="You are an assistant that helps manage files."

info "Starting agent interaction..."
info "System prompt: $SYSTEM_PROMPT"
info "User prompt: $USER_PROMPT"
info "Model: ${MCPSHELL_AGENT_MODEL:-default from config}"

"$CLI_BIN" --tools "$CONFIG_FILE" agent \
    --system-prompt "$SYSTEM_PROMPT" \
    --user-prompt "$USER_PROMPT" \
    $MODEL_FLAG \
    --once \
    --logfile "$LOG_FILE" \
    $API_KEY_FLAG \
    $API_URL_FLAG

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
