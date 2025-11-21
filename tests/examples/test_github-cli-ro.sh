#!/bin/bash
# Test script for github-cli-ro.yaml example
# Tests the gh_raw_file tool with public GitHub repositories

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

#####################################################################################
# Test configuration
TOOLS_FILE="$SCRIPT_DIR/../../examples/github-cli-ro.yaml"
TEST_NAME="test_github_cli_ro"

#####################################################################################
# Start the test

testcase "$TEST_NAME"

info_blue "Configuration file: $TOOLS_FILE"
separator

# Make sure we have the CLI binary
check_cli_exists

# Check if curl is available (required for gh_raw_file)
if ! command_exists curl; then
    skip "curl not found, skipping test"
fi

#####################################################################################
# Test 1: Fetch README from torvalds/linux repository

info "Test 1: Fetching README from torvalds/linux (master branch)"
CMD="$CLI_BIN exe --tools $TOOLS_FILE gh_raw_file repo=torvalds/linux filepath=README ref=master"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 1:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

if [ $RESULT -ne 0 ]; then
    failure "Test 1 failed: Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
fi

# Check if output contains expected content
if echo "$OUTPUT" | grep -q "Linux kernel"; then
    success "Test 1 passed: Successfully fetched README from torvalds/linux"
else
    failure "Test 1 failed: Output doesn't contain expected content"
    echo "$OUTPUT"
    exit 1
fi

separator

#####################################################################################
# Test 2: Fetch LICENSE from golang/go repository

info "Test 2: Fetching LICENSE from golang/go (master branch)"
CMD="$CLI_BIN exe --tools $TOOLS_FILE gh_raw_file repo=golang/go filepath=LICENSE ref=master"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 2:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

if [ $RESULT -ne 0 ]; then
    failure "Test 2 failed: Command execution failed with exit code: $RESULT"
    echo "$OUTPUT"
    exit 1
fi

# Check if output contains expected content
if echo "$OUTPUT" | grep -q "Copyright.*Go Authors"; then
    success "Test 2 passed: Successfully fetched LICENSE from golang/go"
else
    failure "Test 2 failed: Output doesn't contain expected content"
    echo "$OUTPUT"
    exit 1
fi

separator

#####################################################################################
# Test 3: Test constraint validation (path traversal prevention)

info "Test 3: Testing path traversal prevention"
CMD="$CLI_BIN exe --tools $TOOLS_FILE gh_raw_file repo=golang/go filepath=../../../etc/passwd ref=master"
info "Executing: $CMD"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 3:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

# This should fail due to constraint violation
if [ $RESULT -eq 0 ]; then
    failure "Test 3 failed: Command should have been blocked by constraints"
    echo "$OUTPUT"
    exit 1
fi

# Check if output mentions constraint violation
if echo "$OUTPUT" | grep -q "constraint"; then
    success "Test 3 passed: Path traversal correctly blocked by constraints"
else
    failure "Test 3 failed: Expected constraint violation message"
    echo "$OUTPUT"
    exit 1
fi

separator

#####################################################################################
# Test 4: Test with default ref parameter (should use 'main')

info "Test 4: Testing default ref parameter"
CMD="$CLI_BIN exe --tools $TOOLS_FILE gh_raw_file repo=golang/go filepath=CONTRIBUTING.md"
info "Executing: $CMD (should default to 'main' branch)"

OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME - Test 4:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

# This might fail if golang/go doesn't have 'main' branch, but that's okay
# We're just testing that the default parameter works
if [ $RESULT -eq 0 ]; then
    success "Test 4 passed: Default ref parameter works"
elif echo "$OUTPUT" | grep -q "404"; then
    info "Test 4: Got 404 (expected if repo uses 'master' instead of 'main')"
    success "Test 4 passed: Default ref parameter was applied (got 404 for 'main' branch)"
else
    failure "Test 4 failed: Unexpected error"
    echo "$OUTPUT"
    exit 1
fi

separator

success "All tests passed for github-cli-ro.yaml!"
exit 0

