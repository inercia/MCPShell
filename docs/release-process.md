# Release Process

This document describes how to create a new release of MCPShell.

## Automated Release with GitHub Actions

MCPShell uses [GoReleaser](https://goreleaser.com/) and GitHub Actions to automatically build and publish releases when a new tag is pushed.

### Prerequisites

- Clean git working directory (no uncommitted changes)
- Push access to the repository
- All tests passing on main branch

### Creating a Release

1. **Prepare the release** (automated via Makefile):
   ```bash
   make release
   ```
   
   This will:
   - Check that your repository is clean
   - Show existing tags
   - Prompt you for a new version tag (e.g., `v1.2.3`)
   - Update version references in documentation
   - Commit the documentation changes
   - Create the git tag locally

2. **Push the tag to trigger the release**:
   ```bash
   git push origin main v1.2.3
   ```
   
   Replace `v1.2.3` with your actual tag. The Makefile output will show you the exact command to run.

3. **GitHub Actions will automatically**:
   - Run all tests
   - Build binaries for multiple platforms:
     - Linux (amd64, arm64)
     - macOS (amd64, arm64)
     - Windows (amd64)
   - Create archives (tar.gz for Linux/macOS, zip for Windows)
   - Generate checksums
   - Create a GitHub release with all artifacts
   - Generate a changelog from commit messages

### Supported Platforms

The release process builds binaries for the following platforms:

| OS      | Architecture | Format  |
|---------|--------------|---------|
| Linux   | amd64        | tar.gz  |
| Linux   | arm64        | tar.gz  |
| macOS   | amd64 (Intel)| tar.gz  |
| macOS   | arm64 (Apple Silicon) | tar.gz |
| Windows | amd64        | zip     |

### Testing Releases Locally

Before creating an official release, you can test the GoReleaser configuration:

```bash
# Validate the GoReleaser configuration
make release-test

# Build a snapshot release locally (without publishing)
make release-snapshot
```

The snapshot release will create all binaries in the `./dist/` directory without creating a GitHub release.

## Version Numbering

MCPShell follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v2.0.0): Incompatible API changes
- **MINOR** version (v1.1.0): New functionality in a backward-compatible manner
- **PATCH** version (v1.0.1): Backward-compatible bug fixes

### Pre-release Versions

You can also create pre-release versions:
- Alpha: `v1.2.0-alpha.1`
- Beta: `v1.2.0-beta.1`
- Release Candidate: `v1.2.0-rc.1`

GoReleaser will automatically mark these as pre-releases on GitHub.

## Changelog Generation

The changelog is automatically generated from commit messages. To ensure good changelogs, follow these commit message conventions:

- `feat: ...` - New features
- `fix: ...` - Bug fixes
- `perf: ...` - Performance improvements
- `docs: ...` - Documentation changes (excluded from changelog)
- `test: ...` - Test changes (excluded from changelog)
- `chore: ...` - Maintenance tasks (excluded from changelog)

Example:
```
feat: add support for custom templates
fix: resolve timeout issue in agent runtime
perf: optimize command execution pipeline
```

## Release Assets

Each release includes:

1. **Binaries** - Compiled executables for each platform
2. **Archives** - Compressed archives containing:
   - The binary
   - LICENSE file
   - README.md
   - All documentation (docs/)
   - All examples (examples/)
3. **Checksums** - SHA256 checksums for all files
4. **Source code** - Automatic GitHub source archives (zip and tar.gz)

## Troubleshooting

### Release failed

If the GitHub Action fails:

1. Check the [Actions tab](https://github.com/inercia/MCPShell/actions) on GitHub
2. Review the error logs
3. Fix the issue
4. Delete the tag locally and remotely:
   ```bash
   git tag -d v1.2.3
   git push origin :refs/tags/v1.2.3
   ```
5. Start the release process again

### GoReleaser configuration errors

Test your configuration locally before pushing:

```bash
make release-test
```

### Binary doesn't work on target platform

Test locally with:

```bash
make release-snapshot
```

Then test the binaries in `./dist/` on your target platforms.

## CI/CD Pipeline

The repository also includes a CI workflow that runs on every push and pull request:

- **Tests**: Runs on Linux, macOS, and Windows
- **Linting**: Runs golangci-lint
- **Build**: Builds the binary
- **Validation**: Validates example configurations

This ensures that the main branch is always in a releasable state.

## Manual Release (Advanced)

If you need to create a release without using GitHub Actions:

1. Install GoReleaser:
   ```bash
   brew install goreleaser  # macOS
   # or
   go install github.com/goreleaser/goreleaser/v2@latest
   ```

2. Create and push the tag:
   ```bash
   git tag -a v1.2.3 -m "Version 1.2.3"
   git push origin v1.2.3
   ```

3. Run GoReleaser manually:
   ```bash
   export GITHUB_TOKEN="your-github-token"
   goreleaser release --clean
   ```

This is not recommended for regular releases but can be useful for testing or emergency situations.
