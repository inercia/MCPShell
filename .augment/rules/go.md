# Go Coding Standards for MCPShell

## Package Documentation
- **ALWAYS** include package-level documentation: `// Package <name> <description>.`
- Explain package purpose and responsibilities

## Error Handling & Logging
- Wrap errors with `fmt.Errorf` and `%w`: `fmt.Errorf("failed to compile constraint '%s': %w", expr, err)`
- Error messages: lowercase, no punctuation
- **ALWAYS** use `common.Logger` (never `fmt.Println` or `log.Println`)
- Logger passed as parameter to functions
- Levels: Debug (diagnostics), Info (events), Warn (non-critical), Error (failures)

## Panic Recovery
- Use `defer common.RecoverPanic()` at entry points and goroutines

## Project-Specific Patterns
- YAML config with tags: `yaml:"field_name,omitempty"`
- Templates: Go `text/template` + Sprig functions, variables as `{{ .param_name }}`
- Context: First parameter for I/O operations, use `context.WithTimeout`
- Constructors: Provide `New*` functions for complex types
- Type assertions: Check success, handle failures gracefully

## Code Quality
- Run `make format` before commits (runs `go fmt ./...` and `go mod tidy`)
- Pass `golangci-lint` checks
- Godoc-style comments for exported APIs
- Never manually edit `go.mod`
