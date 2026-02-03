# Documentation Standards for MCPShell

## Code Documentation
- **ALWAYS** include package-level documentation: `// Package <name> <description>.`
- Document all exported functions, types, and important fields
- Start comments with the name of what's being documented
- Use complete sentences with proper punctuation

## Configuration Documentation
- Document all options in `docs/config.md`
- Document environment variables in `docs/config-env.md`
- Provide well-commented examples in `examples/`
- Include security rationale for constraints
- Show both simple and advanced patterns
- Cross-link related documentation (config, env vars, usage guides)

## Security Documentation
- Maintain comprehensive `docs/security.md`
- Include prominent security warnings
- Explain risks of LLM command execution
- Provide secure configuration examples

## Markdown Standards
- Use ATX-style headers (`#`, `##`, `###`)
- Specify language for code blocks (`yaml`, `go`, `bash`)
- Use descriptive link text
- Use relative links for internal docs
