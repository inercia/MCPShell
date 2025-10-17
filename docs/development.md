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

This project uses GitHub Actions to automatically build and release binaries. When a tag is pushed, the workflow:

1. Builds binaries for multiple platforms (Linux, macOS, Windows)
1. Creates a GitHub release
1. Attaches the binaries to the release

To create a new release:

```bash
# Create a new release, by tagging the code
make release

# Push the tag to trigger the release workflow
git push origin v0.1.0
```

The release will appear on the GitHub Releases page with binaries for each supported platform.

## Make Targets

- `make build`: Build the application
- `make clean`: Remove build artifacts
- `make test`: Run tests
- `make run`: Run the application
- `make install`: Install the application
- `make lint`: Run linting
- `make help`: Show help
