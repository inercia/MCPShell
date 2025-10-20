# MCPShell Tests

This directory contains end-to-end tests for MCPShell, organized by functionality.

## Directory Structure

```text
tests/
├── agent/                        # Agent functionality tests
│   ├── test_agent.sh             # Main agent test script
│   └── tools/                    # Agent tools configurations
│       └── test_agent.yaml       # Agent test configuration
├── exe/                          # Direct tool execution tests
│   ├── test_exe.sh               # Basic exe command test
│   ├── test_exe_empty_file.sh    # Empty file creation test
│   ├── test_exe_constraints.sh   # Constraint validation test
│   └── test_exe_config.yaml      # Tool configuration for exe tests
├── runners/                      # Runner-specific tests
│   ├── test_runner_docker.sh     # Docker runner tests
│   └── test_runner_docker.yaml   # Docker runner configuration
├── common/                       # Shared utilities and fixtures
│   ├── common.sh                 # Common test utilities
│   ├── test_prompt.json          # Test prompt fixtures
│   └── test_response.json        # Test response fixtures
└── run_tests.sh                  # Main test runner script
```

## Running Tests

### Run All Tests

```bash
cd tests
./run_tests.sh
```

### Run Specific Test Categories

```bash
# Agent tests
cd tests/agent
./test_agent.sh

# Exe command tests
cd tests/exe
./test_exe.sh
./test_exe_empty_file.sh
./test_exe_constraints.sh

# Runner tests
cd tests/runners
./test_runner_docker.sh
```

## Test Categories

### Agent Tests (`agent/`)

Tests the interactive agent functionality that uses LLMs to interact with tools.

- **test_agent_config.sh**: Tests agent configuration management commands (`config show`)
- **test_agent_info.sh**: Tests agent info command with various flags
- **test_agent.sh**: Tests agent initialization, tool calling, and file creation
- **tools/test_agent.yaml**: Agent test configuration (uses MCPSHELL_TOOLS_DIR)

**Note**: The agent test uses the `MCPSHELL_TOOLS_DIR` environment variable to specify
the tools directory, demonstrating how MCPShell can load configurations from custom
directories.

**LLM Availability**: The agent tests use `mcpshell agent info --check` to verify LLM
connectivity before running. If no LLM is available, the tests will skip gracefully
with a clear message explaining how to set up an LLM for testing.

### Exe Tests (`exe/`)

Tests direct tool execution without the agent (using the `exe` command).

- **test_exe.sh**: Basic tool execution and file creation
- **test_exe_empty_file.sh**: Tests default content handling for empty files
- **test_exe_constraints.sh**: Tests that constraints are properly enforced
- **test_exe_timeout.sh**: Tests that command timeouts work correctly

### Runner Tests (`runners/`)

Tests different execution environments for tools.

- **test_runner_docker.sh**: Tests Docker-based tool execution
- **test_runner_sandbox_exec.sh**: Tests sandbox-exec runner (macOS only)

### Common Utilities (`common/`)

Shared utilities and test fixtures used across all tests.

- **common.sh**: Common functions for test setup, assertions, and utilities
- **test_prompt.json**: Sample prompt data for testing
- **test_response.json**: Sample response data for testing

## Adding New Tests

When adding new tests:

1. **Determine the category**: agent, exe, runners, or create a new category
1. **Create test files in the appropriate subdirectory**
1. **Update `run_tests.sh`** to include the new test in the TEST_FILES array
1. **Use common utilities** by sourcing `../common/common.sh` (or appropriate path)
1. **Follow naming conventions**: `test_<functionality>.sh` for scripts

## Test Dependencies

- All test scripts depend on the built `mcpshell` binary in `../build/mcpshell`
- Agent tests require an LLM endpoint (default: local Ollama at `http://localhost:11434/v1`)
  - If no LLM is available, agent tests will be skipped gracefully
  - The tests use `mcpshell agent info --check` to verify LLM connectivity
  - Configure via environment variables: `MODEL`, `OPENAI_API_BASE`, `OPENAI_API_KEY`
- Docker tests require Docker to be installed and running
- All tests use the shared utilities in `common/common.sh`
