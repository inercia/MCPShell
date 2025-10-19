# Configuration File

The MCPShell can be configured using a YAML configuration file to define the tools that the MCP server provides.

## Basic Structure

The configuration file uses the following structure:

```yaml
mcp:
  run:
    shell: "<shell>"
  description: <global description>
  tools:
    - name: "<tool_name>"
      description: "<tool description>"
      params:
        <param name>:
          type: <string|number|boolean>
          description: "<parameter description>"
          required: <true|false>
          default: <value>
      constraints:
        - "<constraint expression>"
      run:
        command: "<command to execute>"
        env:
          - <env var>
        runners:
          - name: "<runner name>"
            requirements:
              os: "<OS>"
              executables:
                - "<executable>"
            options:
              <option>:<value>
      output:
        prefix: "<text to prepend to the output>"
```

## MCPShell Configuration

The top-level `mcp` section contains configuration for the MCP server:

- `description`: global description of the toolkit.
- `run`: Global run configuration settings
  - `shell`: Optional string specifying which shell to use for command execution.
    If not provided, the system will use the SHELL environment variable or fall back to `/bin/sh`.
- `tools`: Array of tool definitions (required)

## Tools Definitions

Each tool is defined with the following properties:

- `name`: The name of the tool (required)
- `description`: A description of what the tool does (required).
  This is specially important in order to instruct the LLM what this tool does.
  Otherwise, the LLM will not know that it can use this tool for fullfilling
  the user requests.
- `params`: A map of parameters that the tool accepts
- `constraints`: A list of CEL expressions to validate before command execution (optional)
- `run`: Configuration for how the tool executes (required)
- `output`: Configuration for tool output formatting (optional)

### Parameter Definition

Each parameter has the following properties:

- `type`: The parameter type (string, number, or boolean). Optional, defaults to "string" if not specified.
- `description`: A description of the parameter. Be verbose on this description,
  as it will be used by the LLM for knowing how to pass this information to the tool.
- `required`: Whether the parameter is required (default: false)
- `default`: A default value to use when the parameter is not provided by the LLM.
  The value must match the parameter type (string, number, or boolean).

Default values provide fallback values for optional parameters when they aren't specified by the LLM or command line. This allows tools to have sensible defaults while still allowing explicit values to be provided when needed. Default values are applied before constraint evaluation.

### Constraints

Constraints are optional [CEL (Common Expression Language)](https://github.com/google/cel-spec)
expressions that are evaluated before command execution.
The tool will only execute if all constraints evaluate to `true`.
This provides a safety mechanism to prevent
potentially dangerous commands from being executed.

Constraints have access to all tool parameters by name. For example:

```yaml
constraints:
  - "text.startsWith('text:')"  # Ensures the text parameter starts with "text:"
  - "command.size() < 100"      # Ensures the command parameter is less than 100 characters
```

#### Understanding CEL Constraint Language

[CEL (Common Expression Language)](https://github.com/google/cel-spec) is a simple, portable
expression language developed by Google. In the MCPShell, CEL is used to define safety
constraints that validate parameters before command execution.

##### Basic Types and Operations

CEL supports three basic parameter types in MCPShell:

1. **String operations**:

   - `string.size()` - Returns the length of the string
   - `string.contains(substring)` - Checks if a string contains a substring
   - `string.startsWith(prefix)` - Checks if a string starts with a prefix
   - `string.endsWith(suffix)` - Checks if a string ends with a suffix
   - `string.matches(regex)` - Checks if a string matches a regular expression

   ```yaml
   constraints:
     - "name.size() <= 50"                      # Limit string length
     - "!filename.contains('../')"              # Prevent directory traversal
     - "text.matches('^[a-zA-Z0-9 ,.!?]*$')"    # Only allow alphanumeric and basic punctuation
   ```

1. **Numeric operations**:

   - Comparison operators: `==`, `!=`, `<`, `<=`, `>`, `>=`
   - Arithmetic operators: `+`, `-`, `*`, `/`, `%`

   ```yaml
   constraints:
     - "value > 0.0"                # Ensure positive values
     - "value <= 100.0"             # Set upper limit
     - "count % 2 == 0"             # Ensure even numbers only
   ```

1. **Boolean operations**:

   - Logical operators: `&&` (and), `||` (or), `!` (not)

   ```yaml
   constraints:
     - "flag == true"               # Check if flag is true
     - "!dangerous"                 # Ensure dangerous flag is not set
     - "valid && authorized"        # Require both valid and authorized flags
   ```

##### Advanced Features

1. **List operations and quantifiers**:

   - `list.exists(var, condition)` - Checks if any element in the list satisfies the condition
   - `list.all(var, condition)` - Checks if all elements in the list satisfy the condition

   ```yaml
   constraints:
     - "['ls', 'echo', 'cat'].exists(cmd, cmd == command)"  # Whitelist allowed commands
     - "['.jpg', '.png', '.gif'].exists(ext, filename.endsWith(ext))"  # Only allow certain file extensions
   ```

1. **Combining multiple constraints**:

   - Multiple constraints are implicitly AND-ed together
   - Use `||` inside a single constraint expression for OR logic

   ```yaml
   constraints:
     - "command == 'ls' || command == 'pwd'"  # Allow only ls or pwd
     - "!command.contains('rm')"              # Never allow rm command
   ```

##### Common Constraint Patterns

1. **Security constraints** to prevent command injection:

   ```yaml
   constraints:
     - "!text.contains(';')"                    # Prevent command chaining
     - "!text.contains('&')"                    # Prevent background execution
     - "!text.contains('|')"                    # Prevent piping
     - "!text.matches('.*[;&|`$].*')"           # Block shell special characters
   ```

1. **Filesystem safety constraints**:

   ```yaml
   constraints:
     - "!path.contains('../')"                  # Prevent directory traversal
     - "path.startsWith('/allowed/dir/')"       # Only allow specific directory
     - "path.matches('^[a-zA-Z0-9_\\-./]+$')"   # Only allow safe path characters
   ```

1. **Command whitelisting**:

   ```yaml
   constraints:
     - "['ls', 'ps', 'echo', 'cat', 'grep'].exists(c, c == command)"  # Allow only specific commands
   ```

1. **Input validation constraints**:

   ```yaml
   constraints:
     - "email.matches('^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$')"  # Validate email format
     - "phone.matches('^\\+?[0-9]{10,15}$')"                                 # Validate phone number
   ```

### `run` Configuration

The run configuration defines how the tool executes:

- `command`: A shell command to execute (required)
- `env`: A list of environment variable names to pass from the parent process to the command (optional)
  - Environment variables can be just names (ie, `KUBECONFIG`),
    assignments (ie, `KUBECONFIG=/some/path`) or event templated
    assignments (ie, `KUBECONFIG={{ .kubeconfig }}`).
- `timeout`: Maximum duration for command execution (optional)
  - Format: A duration string such as "30s", "5m", "1h30m"
  - If not specified, no timeout is applied (commands can run indefinitely)
  - Examples: "10s" (10 seconds), "2m" (2 minutes), "1h" (1 hour)
  - **Recommended**: Always set a timeout to prevent commands from hanging
- `runners`: An array of runner configurations that will be used to execute the command (optional)

Commands can use the Go template syntax, including the presence of parameters like `{{ .param_name }}`.

Example specifying environment variables and timeout:

```yaml
run:
  env:
    - KUBECONFIG     # Pass the KUBECONFIG environment variable to the command
    - HOME           # Pass the HOME environment variable to the command
    - TESTS=false    # Pass some env variables with some values
  timeout: "30s"     # Timeout after 30 seconds
  command: |
    kubectl get {{ .resource }}
```

This is useful for tools that need access to environment variables like API keys, configuration paths, or user information.

#### About Runners

Runners define how commands are executed, with options for sandboxing and cross-platform support. The `runners` array is optional - if not provided, a default "exec" runner will be used.

Here's a simple example with multiple runners:

```yaml
runners:
  - name: sandbox-exec
    options:
      allow_networking: false
  - name: firejail
    options:
      allow_networking: false
  - name: exec     # Fallback runner
```

For detailed information about runners, including options, selection process, and supported types, see [Runner Configuration](config-runners.md).

### `output` Configuration

The output configuration defines how the tool's output is formatted:

- `prefix`: Text to prepend to the command output (optional)

Similar to commands, prefixes can include parameter values using the same Go template syntax with `{{ .param_name }}`.

## Go Template Features

The MCPShell uses Go's text/template package for parameter substitution, which supports a variety of powerful features:

### Basic Substitution

```console
{{ .param_name }}
```

### Conditional Logic

```console
{{ if .param_name }}value is present{{ else }}value is not present{{ end }}
```

### Nested Conditions

```console
{{ if .param1 }}
  {{ if .param2 }}both params exist{{ else }}only param1 exists{{ end }}
{{ else }}
  param1 doesn't exist
{{ end }}
```

### Comparison Operators

```console
{{ if eq .param "value" }}equal{{ end }}
{{ if ne .param "value" }}not equal{{ end }}
{{ if lt .param 5 }}less than{{ end }}
{{ if gt .param 5 }}greater than{{ end }}
```

### String Operations

```console
{{ .param | lower }}    <!-- lowercase -->
{{ .param | upper }}    <!-- uppercase -->
{{ .param | title }}    <!-- title case -->
```

For more advanced template features, refer to the [Go text/template documentation](https://pkg.go.dev/text/template).

### Functions

In addition to the standard functions available in the Golang templating library,
[these functions](https://github.com/Masterminds/sprig/blob/master/docs/index.md)
are also available.
