# Example Configuration Tests

This directory contains integration tests for the example configuration files in `/examples`.

## Overview

These tests verify that the example configurations work correctly by executing specific tools with known inputs and validating the outputs. The tests use the `mcpshell exe` command to directly invoke tools without requiring a full MCP server setup.

## Test Files

### `test_github-cli-ro.sh`
Tests the GitHub CLI read-only tools from `examples/github-cli-ro.yaml`.

**Tests:**
- Fetching raw files from public GitHub repositories (torvalds/linux, golang/go)
- Path traversal prevention (security constraint validation)
- Default parameter handling (ref defaults to 'main')
- Command injection prevention

**Requirements:**
- `curl` command available
- Internet connectivity to github.com

### `test_config.sh`
Tests basic utility tools from `examples/config.yaml`.

**Tests:**
- `hello_world` - Simple greeting tool
- `calculator` - Mathematical expression evaluation
- `number_validator` - Numeric operations (square, double, half)
- Constraint validation (name length limits)
- `secure_shell` - Whitelisted command execution

**Requirements:**
- `bc` command available (for calculator)
- Basic shell commands (echo, pwd)

### `test_disk-diagnostics-ro.sh`
Tests disk diagnostic tools from `examples/disk-diagnostics-ro.yaml`.

**Tests:**
- `storage_overview` - Filesystem usage and mount information

**Requirements:**
- `df` command available
- Basic disk utilities

## Running Tests

### Run all tests (including examples)
```bash
make test-e2e
```

### Run only example tests
```bash
./tests/examples/test_github-cli-ro.sh
./tests/examples/test_config.sh
./tests/examples/test_disk-diagnostics-ro.sh
```

### Run a specific test
```bash
./tests/examples/test_config.sh
```

## Test Structure

Each test follows this pattern:

1. **Setup**: Source common utilities, define configuration
2. **Validation**: Check prerequisites (CLI binary, required commands)
3. **Test Cases**: Execute tools with specific inputs
4. **Verification**: Validate outputs match expected results
5. **Cleanup**: Remove temporary files if needed
6. **Exit**: Return 0 for success, 1 for failure

## Adding New Tests

To add a test for a new example configuration:

1. Create `test_<example-name>.sh` in this directory
2. Follow the existing test pattern (see `test_config.sh` as a template)
3. Add the test to `TEST_FILES` array in `tests/run_tests.sh`
4. Make the script executable: `chmod +x tests/examples/test_<example-name>.sh`
5. Test locally before committing

### Example Test Template

```bash
#!/bin/bash
# Test script for <example-name>.yaml

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTS_ROOT="$(dirname "$SCRIPT_DIR")"
source "$TESTS_ROOT/common/common.sh"

# Test configuration
TOOLS_FILE="$SCRIPT_DIR/../../examples/<example-name>.yaml"
TEST_NAME="test_<example_name>"

# Start the test
testcase "$TEST_NAME"
info_blue "Configuration file: $TOOLS_FILE"
separator

# Make sure we have the CLI binary
check_cli_exists

# Test 1: Your first test
info "Test 1: Description"
CMD="$CLI_BIN exe --tools $TOOLS_FILE <tool_name> <params>"
OUTPUT=$(eval "$CMD" 2>&1)
RESULT=$?

if [ $RESULT -ne 0 ]; then
    failure "Test 1 failed"
    echo "$OUTPUT"
    exit 1
fi

success "Test 1 passed"
separator

success "All tests passed!"
exit 0
```

## CI/CD Considerations

These tests are designed to run in GitHub Actions CI environment:

- **Avoid tests requiring credentials** (AWS, private repos, etc.)
- **Avoid tests requiring external services** (databases, APIs with auth)
- **Use public, stable resources** (well-known GitHub repos, basic system commands)
- **Handle network failures gracefully** (skip tests if resources unavailable)

## Examples NOT Tested

Some examples are intentionally not tested because they require resources unavailable in CI:

- `aws-*.yaml` - Requires AWS credentials and resources
- `kubectl-ro.yaml` - Requires Kubernetes cluster
- `container-diagnostics-ro.yaml` - Requires Docker containers
- `network-diagnostics-ro.yaml` - May have network restrictions in CI

These can be tested manually in appropriate environments.

## Utilities

Tests use utilities from `tests/common/common.sh`:

- `testcase()` - Print test header
- `info()` - Print informational message
- `success()` - Print success message (green checkmark)
- `failure()` - Print failure message (red X)
- `skip()` - Skip test with message
- `fail()` - Fail test and exit
- `check_cli_exists()` - Verify CLI binary exists
- `command_exists()` - Check if command is available

## Debugging

Test output is logged to `tests/e2e_output.log` for debugging failed tests.

To run tests with verbose output:
```bash
./tests/examples/test_config.sh 2>&1 | tee test_output.log
```

