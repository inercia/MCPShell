#!/bin/bash

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

# Configuration file for this test
CONFIG_FILE="$SCRIPT_DIR/test_runner_docker.yaml"
TEST_NAME="test_runner_docker"


#####################################################################################
# Start the test

testcase "$TEST_NAME"

info "Testing Docker runner with config: $CONFIG_FILE"

separator
info "1. Checking if Docker is installed and running"
separator

# Check if Docker is installed and running
# And try to run a simple docker command to check if the daemon is running
command_exists docker || skip "Docker not installed, skipping test"
docker ps &> /dev/null || skip "Docker daemon not running, skipping test"
success "Docker is available, proceeding with tests"

# Make sure we have the CLI binary
check_cli_exists

separator
info "2. Simple hello world in Docker container"
separator

CMD="$CLI_BIN --tools $CONFIG_FILE exe docker_hello"
info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME:\n\n$OUTPUT" >> "$E2E_LOG_FILE"
[ $RESULT -eq 0 ] || fail "Test failed with exit code $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Hello from Docker container" || fail "Test failed: Expected output not found" "$OUTPUT"

success "Test passed"

separator
info "3. Environment variable passing"
separator

CMD="$CLI_BIN --tools $CONFIG_FILE exe docker_with_env message=\"Hello from Docker container\""
info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME:\n\n$OUTPUT" >> "$E2E_LOG_FILE"

[ $RESULT -eq 0 ] || fail "Test failed with exit code $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Hello from Docker container" || fail "Test failed: Expected output not found" "$OUTPUT"

success "Test passed"

separator
info "4. Prepare command functionality"
separator

CMD="$CLI_BIN --tools $CONFIG_FILE exe docker_with_prepare"
info "Executing: $CMD"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?
[ -n "$E2E_LOG_FILE" ] && echo -e "\n$TEST_NAME:\n\n$OUTPUT" >> "$E2E_LOG_FILE"
[ $RESULT -eq 0 ] || fail "Test failed with exit code $RESULT" "$OUTPUT"
echo "$OUTPUT" | grep -q "Preparation successful" || fail "Test failed: prepare command output not found" "$OUTPUT"

success "Test passed"

echo
success "All Docker runner tests passed!"
exit 0 