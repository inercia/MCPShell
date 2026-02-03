---
type: "always_apply"
---

# Architecture and Design Patterns for MCPShell

## Project Structure

### Directory Organization
```
mcpshell/
├── cmd/                    # Command-line interface implementation
│   ├── root.go            # Root command and global flags
│   ├── mcp.go             # MCP server command
│   ├── exe.go             # Direct tool execution command
│   ├── validate.go        # Configuration validation command
│   └── daemon.go          # Daemon mode command
├── pkg/                    # Core application packages
│   ├── server/            # MCP server implementation
│   ├── command/           # Command execution and runners
│   ├── config/            # Configuration loading and validation
│   ├── common/            # Shared utilities and types
│   └── utils/             # Helper functions
├── docs/                   # Documentation
├── examples/              # Example configurations
├── tests/                 # Integration and E2E tests
├── build/                 # Build output directory
└── main.go                # Application entry point
```

### Package Responsibilities

#### `cmd/` Package
- Command-line interface implementation using Cobra
- Command definitions and flag parsing
- User interaction and output formatting
- Delegates business logic to `pkg/` packages

#### `pkg/server/` Package
- MCP server lifecycle management
- Tool registration and discovery
- Request handling and routing
- Integration with MCP protocol library

#### `pkg/command/` Package
- Command handler creation and execution
- Runner implementations (exec, firejail, sandbox-exec, docker)
- Template processing and parameter substitution
- Constraint evaluation and validation

#### `pkg/config/` Package
- YAML configuration loading and parsing
- Configuration validation
- Tool definition structures
- Configuration merging for multiple files

#### `pkg/common/` Package
- Shared types and interfaces
- Logging infrastructure
- Constraint compilation and evaluation (CEL)
- Template utilities
- Panic recovery
- Prerequisite checking

#### `pkg/utils/` Package
- Helper functions for file operations
- Path resolution and normalization
- Home directory detection
- Tool file discovery

## Design Patterns

### Dependency Injection
- Pass dependencies (logger, config) as parameters to constructors
- Use constructor functions (New*) for complex types
- Avoid global state except for the global logger
- Example:
  ```go
  func New(cfg Config, logger *common.Logger) *Server {
      return &Server{
          config: cfg,
          logger: logger,
      }
  }
  ```

### Interface-Based Design
- Define interfaces for pluggable components (Runner, ModelProvider)
- Use interfaces to enable testing with mocks
- Keep interfaces small and focused (Interface Segregation Principle)
- Example:
  ```go
  type Runner interface {
      Run(ctx context.Context, shell string, command string, env []string, params map[string]interface{}, tmpfile bool) (string, error)
      CheckImplicitRequirements() error
  }
  ```

### Factory Pattern
- Use factory functions for creating handlers and runners
- Factory functions handle initialization and validation
- Example:
  ```go
  func NewCommandHandler(tool config.Tool, shell string, logger *common.Logger) (*CommandHandler, error)
  ```

### Strategy Pattern
- Multiple runner implementations (ExecRunner, FirejailRunner, SandboxRunner, DockerRunner)
- Runner selection based on requirements and availability
- Fallback to default runner when specific runner unavailable

### Builder Pattern
- Configuration structs with optional fields
- Use functional options for complex initialization when needed
- Example:
  ```go
  type Config struct {
      ConfigFile          string
      Shell               string
      Logger              *common.Logger
      Version             string
      Descriptions        []string
      DescriptionFiles    []string
      DescriptionOverride bool
  }
  ```

## Architectural Principles

### Separation of Concerns
- Clear separation between CLI, business logic, and infrastructure
- Each package has a single, well-defined responsibility
- Avoid circular dependencies between packages

### Error Handling
- Errors are wrapped with context at each layer
- Use `fmt.Errorf` with `%w` for error wrapping
- Log errors at the point where they can be handled
- Return errors to callers for decision-making

### Logging Strategy
- Structured logging with levels (Debug, Info, Warn, Error)
- Logger passed as dependency, not accessed globally (except via GetLogger)
- Debug logging for detailed diagnostics
- Info logging for important events
- Error logging for failures

### Context Propagation
- Pass `context.Context` as first parameter for I/O operations
- Use context for cancellation and timeouts
- Respect context cancellation in long-running operations

### Configuration Management
- YAML-based configuration files
- Support for multiple configuration files with merging
- Validation at load time
- Default values for optional settings

## Security Architecture

### Defense in Depth
- Multiple layers of security (constraints, runners, validation)
- Fail-safe defaults (deny by default)
- Explicit whitelisting over blacklisting

### Constraint System
- CEL-based constraint evaluation
- Constraints compiled at startup for early error detection
- Constraint failures block command execution
- Detailed logging of constraint evaluation

### Runner Isolation
- Sandboxed execution environments (firejail, sandbox-exec, docker)
- Minimal permissions by default
- Network isolation when possible
- Filesystem restrictions

### Input Validation
- Type checking for all parameters
- Constraint validation before execution
- Template validation at load time
- Path normalization and validation

## Testing Architecture

### Test Organization
- Unit tests in same package as source code (`*_test.go`)
- Integration tests in `tests/` directory
- Shell scripts for E2E testing
- Test utilities in `tests/common/`

### Test Patterns
- Table-driven tests for multiple scenarios
- Test logger that discards output
- Mock implementations of interfaces
- Separate test fixtures and data

### Test Coverage
- Unit tests for business logic
- Integration tests for command execution
- E2E tests for full workflows
- Security tests for constraint validation

## Extension Points

### Adding New Runners
1. Implement the `Runner` interface
2. Add runner-specific options and requirements
3. Register runner in runner factory
4. Add tests for new runner
5. Document runner capabilities and limitations

### Adding New Commands
1. Create command file in `cmd/` package
2. Define command structure with Cobra
3. Implement command logic
4. Add command to root command in `init()`
5. Add tests and documentation

### Adding New Model Providers
1. Implement the `ModelProvider` interface
2. Add provider-specific configuration
3. Register provider in model factory
4. Add tests for provider integration
5. Document provider setup and usage

## Performance Considerations

### Constraint Compilation
- Constraints compiled once at startup
- Compiled constraints reused for all executions
- Reduces overhead for repeated tool calls

### Template Caching
- Templates parsed once during handler creation
- Reused for all executions of the same tool
- Reduces parsing overhead

### Concurrent Execution
- Tools can be executed concurrently
- Context-based cancellation for timeouts
- Proper cleanup of resources

## Scalability Considerations

### Multiple Configuration Files
- Support for loading multiple configuration files
- Configuration merging for combining tool sets
- Efficient tool registration and lookup

### Large Tool Sets
- Efficient tool registration
- Fast tool lookup by name
- Minimal memory overhead per tool

### Long-Running Operations
- Context-based timeouts
- Graceful cancellation
- Resource cleanup on timeout or cancellation
