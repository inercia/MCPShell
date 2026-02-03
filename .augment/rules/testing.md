# Testing Standards for MCPShell

## Test Organization
- Test files: `*_test.go` in same package as source
- Integration tests: `tests/` directory
- E2E tests: `tests/test_*.sh` shell scripts

## Unit Testing
- Use table-driven tests for multiple scenarios
- Test logger: `var testLogger, _ = common.NewLogger("", "", common.LogLevelNone, false)`
- Test both success and failure cases
- **ALWAYS** test constraint validation logic
- **ALWAYS** test error handling paths
- **ALWAYS** test parameter type conversions
- **ALWAYS** test template rendering
- **ALWAYS** test runner selection

## Integration Testing
- Shell scripts in `tests/` directory
- Use utilities from `tests/common/common.sh`: `info()`, `success()`, `fail()`, `skip()`
- Run with `make test-e2e`

## Constraint Testing
- Test constraint compilation (valid/invalid expressions)
- Test constraint evaluation with various values
- **ALWAYS** test security constraints block malicious inputs
- Test command injection, path traversal, input limits

## Test Execution
- Unit tests: `make test`
- Integration tests: `make test-e2e`
- Race detection: `go test -race ./...`
- Coverage: `go test -cover ./...`
