# Security Rules for MCPShell

## Core Principles
- **NEVER** allow arbitrary command execution without strict constraints
- **ALWAYS** prefer read-only operations
- **ALWAYS** validate inputs with CEL constraints
- **ALWAYS** use sandboxed runners when possible

## Common Constraints

### Command Injection Prevention
```yaml
constraints:
  - "!param.contains(';')"      # Prevent command chaining
  - "!param.contains('&&')"     # Prevent command chaining
  - "!param.contains('|')"      # Prevent piping
  - "!param.contains('`')"      # Prevent command substitution
  - "!param.contains('$(')"     # Prevent command substitution
```

### Path Traversal Prevention
```yaml
constraints:
  - "!path.contains('../')"                    # Prevent directory traversal
  - "path.startsWith('/allowed/directory/')"   # Restrict to specific directory
  - "path.matches('^[a-zA-Z0-9_\\-./]+$')"    # Only safe characters
```

### Input Validation
```yaml
constraints:
  - "param.size() > 0 && param.size() <= 1000"  # Length limits
  - "['ls', 'cat', 'echo'].exists(cmd, cmd == command)"  # Command whitelist
  - "['.txt', '.log', '.md'].exists(ext, filepath.endsWith(ext))"  # File extensions
```

## Runner Security
- Use most restrictive runner available
- Disable networking: `allow_networking: false`
- Restrict filesystem access
- Specify OS requirements

## Environment Variables
- **ONLY** pass explicitly whitelisted variables
- **NEVER** log sensitive data
- Document why each variable is needed

## Security Checklist for New Tools
- [ ] All parameters have constraints
- [ ] Command injection blocked
- [ ] Path traversal prevented
- [ ] Input length limits enforced
- [ ] Appropriate runner selected
- [ ] Environment variables whitelisted
- [ ] Tool is read-only or justified
- [ ] Tested with malicious inputs
