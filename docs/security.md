# Security Considerations

## ⚠️ WARNING: Potential Security Risks

The MCPShell allows Large Language Models (LLMs) to execute commands on your local machine. This functionality comes with significant security implications that all users must understand before deployment.

## Primary Security Concerns

### Command Execution Risks

When you allow an LLM to execute shell commands on your system:

- **Data Destruction**: An LLM could issue commands that delete files or directories (`rm -rf`, etc.)
- **Data Exfiltration**: Commands could be used to send your sensitive data to external servers
- **System Modification**: System configurations could be altered in harmful ways
- **Resource Exhaustion**: Commands could be designed to consume excessive CPU, memory, or disk space
- **Privilege Escalation**: If running with elevated permissions, the damage potential increases significantly

### Parameter Injection Risks

Even with "safe" commands, parameters can be dangerous:

- **Path Traversal**: Parameters containing `../` sequences could access files outside expected directories
- **Command Injection**: Parameters containing shell metacharacters (`;`, `&&`, `|`, etc.) could execute additional unintended commands
- **Resource Overloading**: Parameters designed to trigger excessive resource usage (e.g., extremely large file sizes)

## Best Practices for Secure Usage

### 1. Prefer Read-Only Commands

Whenever possible, limit LLM-executed commands to those without side effects:

✅ **Safer Commands** (examples):

- `ls`, `dir` - List directory contents
- `cat`, `type` - View file contents
- `grep`, `find` - Search operations
- `ps`, `top` - Process information

❌ **Higher Risk Commands** to avoid or restrict heavily:

- `rm`, `del` - Delete files
- `mv`, `move` - Move files
- `chmod` - Change permissions
- Any command that writes to disk or modifies system state

### 2. Implement Strict Constraints

Always define and enforce constraints on commands:

- **Allowlist-based approach**: Only permit specific, pre-approved commands
- **Directory restrictions**: Limit file operations to specific directories
- **Command pattern validation**: Ensure commands match expected patterns before execution
- **Parameter validation**: Validate all parameters against strict rules

### 3. Parameter Validation

Add validation for all command parameters:

- **Type checking**: Ensure parameters are of expected types
- **Range validation**: For numeric parameters, ensure they fall within safe ranges
- **Pattern matching**: For string parameters, validate against strict patterns
- **Size limitations**: Restrict the size of inputs to prevent resource exhaustion
- **Character filtering**: Sanitize inputs to remove potentially dangerous characters

Example constraint approach:

```yaml
  constraints:
    - "value.length < 100"
    - "not value.includes('..')"
    - "not value.includes(';'"
}
```

### 4. Use the Restricted _runners_

- Use one of the restricted [runners](config-runners.md)
- Limit the directories and files the runner can access.

### 5. Run with Minimal Privileges

- Run the adapter with the least privileges necessary
- Create a dedicated user account with limited permissions
- Use containerization when possible to isolate execution

### 6. Audit and Monitor

- Log all commands executed by the LLM
- Regularly review logs for suspicious activity
- Implement alerting for potentially dangerous commands

## Disclaimer of Liability

**THE MCPShell IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED.**

By using the MCPShell, you acknowledge and accept all risks associated with allowing an LLM to execute commands on your system. The developers, contributors, and associated organizations are not responsible for any damage, data loss, security breaches, or other negative consequences resulting from the use of this software.

It is your responsibility to:

1. Understand the security implications of each command you allow
1. Implement appropriate constraints and validations
1. Monitor system activity and respond to suspicious behavior
1. Maintain regular backups of important data
1. Deploy in a manner consistent with your own security requirements

**If you cannot accept these risks, do not use the MCPShell for command execution.**

## Reporting Security Issues

If you discover security vulnerabilities in the MCPShell, please report them responsibly by [creating an issue](https://github.com/inercia/MCPShell/issues) with appropriate security labels.
