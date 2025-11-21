# Development

## Prerequisites

- Go 1.18 or higher
- Git

## Project Structure

```console
.
├── cmd/           # Command definitions
├── docs/          # Documentation
├── pkg/           # Source code
├── main.go        # Entry point
├── Makefile       # Build scripts
└── README.md      # This file
```

## Building from source

1. Clone the repository:

   ```console
   git clone https://github.com/inercia/MCPShell.git
   cd mcpshell
   ```

1. Build the application:

   ```console
   make build
   ```

   This will create a binary in the `build` directory.

1. Alternatively, install the application:

   ```console
   make install
   ```

### Continuous Integration

This project uses GitHub Actions for continuous integration:

- **Pull Request Testing**: Every pull request triggers an automated workflow that:
  - Runs all unit tests
  - Performs race condition detection
  - Checks code formatting
  - Runs linters to ensure code quality

These checks help maintain code quality and prevent regressions as the project evolves.

## Releases

This project uses [GoReleaser](https://goreleaser.com/) and GitHub Actions to automatically build and release binaries for multiple platforms. When a tag is pushed, the workflow:

1. Runs all tests
1. Builds binaries for multiple platforms (Linux, macOS, Windows) and architectures (amd64, arm64)
1. Creates archives with documentation and examples
1. Generates checksums
1. Creates a GitHub release with all artifacts
1. Generates a changelog from commit messages

### Quick Start

To create a new release:

```bash
# Create and tag a new release (updates docs and creates tag)
make release

# Push the tag to trigger the automated release workflow
git push origin main v1.2.3
```

The release will appear on the GitHub Releases page with binaries for each supported platform.

### Testing Releases Locally

Before creating an official release:

```bash
# Validate the GoReleaser configuration
make release-test

# Build a snapshot release locally (no tag required)
make release-snapshot
```

For detailed information about the release process, see the [Release Process Guide](release-process.md).

## Make Targets

Run `make help` to see all available targets. Key targets include:

- `make build`: Build the application
- `make clean`: Remove build artifacts
- `make test`: Run unit tests
- `make test-e2e`: Run end-to-end tests
- `make run`: Run the application
- `make install`: Install the application
- `make lint`: Run linting
- `make format`: Format Go code
- `make validate-examples`: Validate all YAML configs
- `make release`: Create a new release (tag + docs update)
- `make release-test`: Validate GoReleaser configuration
- `make release-snapshot`: Build snapshot release locally
- `make help`: Show all available targets
