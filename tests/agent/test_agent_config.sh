#!/bin/bash
# Tests the MCPShell agent config functionality

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Configuration for this test
TEST_NAME="test_agent_config"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info "Testing MCPShell agent config commands"

# Make sure we have the CLI binary
check_cli_exists

separator
info "1. Testing 'agent config show' command"
separator

OUTPUT=$("$CLI_BIN" agent config show 2>&1)
RESULT=$?

[ $RESULT -eq 0 ] || fail "Agent config show command failed with exit code: $RESULT" "$OUTPUT"

# Verify the output contains expected information
echo "$OUTPUT" | grep -q "Configuration file:" || fail "Expected 'Configuration file:' in output" "$OUTPUT"

# Check if config exists (either shows models or says no config found)
if echo "$OUTPUT" | grep -q "No agent configuration found"; then
    info "No agent configuration found (this is acceptable)"
    info "Output: $OUTPUT"
else
    # If config exists, verify it shows models
    echo "$OUTPUT" | grep -q "Agent Configuration:" || fail "Expected 'Agent Configuration:' in output" "$OUTPUT"
    echo "$OUTPUT" | grep -q "Model" || fail "Expected 'Model' information in output" "$OUTPUT"
    success "Agent config show displayed existing configuration"
fi

success "Agent config show command passed"

separator
info "2. Verifying agent configuration file location"
separator

# Extract config file path from output
CONFIG_PATH=$(echo "$OUTPUT" | grep "Configuration file:" | sed 's/Configuration file: //')

if [ -f "$CONFIG_PATH" ]; then
    success "Agent configuration file exists at: $CONFIG_PATH"
    
    # Show a sample of the config
    info "Configuration file content (first 10 lines):"
    head -10 "$CONFIG_PATH" | sed 's/^/  /'
else
    info "Agent configuration file not found at: $CONFIG_PATH"
    info "Run 'mcpshell agent config create' to create a default configuration"
fi

separator
info "3. Testing 'agent config show --json' command"
separator

# Only test JSON output if config exists
if [ -f "$CONFIG_PATH" ]; then
    OUTPUT_JSON=$("$CLI_BIN" agent config show --json 2>&1)
    RESULT=$?
    
    [ $RESULT -eq 0 ] || fail "Agent config show --json command failed with exit code: $RESULT" "$OUTPUT_JSON"
    
    # Verify JSON output is valid
    echo "$OUTPUT_JSON" | grep -q '"configuration_file":' || fail "Expected 'configuration_file' in JSON output" "$OUTPUT_JSON"
    echo "$OUTPUT_JSON" | grep -q '"models":' || fail "Expected 'models' in JSON output" "$OUTPUT_JSON"
    
    # Try to parse as JSON (if jq is available)
    if command -v jq &> /dev/null; then
        echo "$OUTPUT_JSON" | jq . > /dev/null 2>&1 || fail "JSON output is not valid JSON" "$OUTPUT_JSON"
        
        # Show formatted JSON sample
        info "JSON output (formatted):"
        echo "$OUTPUT_JSON" | jq . | head -15 | sed 's/^/  /'
        
        success "Agent config show --json produced valid JSON output"
    else
        info "jq not available, skipping JSON validation"
        success "Agent config show --json command passed"
    fi
else
    info "Skipping JSON test - no configuration file found"
fi

separator
success "All agent config tests completed successfully!"
exit 0
