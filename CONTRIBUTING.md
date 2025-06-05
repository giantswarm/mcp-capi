# Contributing to MCP CAPI

Thank you for your interest in contributing to the MCP CAPI Server! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct (to be added).

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/mcp-capi.git`
3. Add upstream remote: `git remote add upstream https://github.com/giantswarm/mcp-capi.git`
4. Create a feature branch: `git checkout -b feature/your-feature-name`

## Development Setup

### Prerequisites

- Go 1.23 or later
- Access to a CAPI management cluster
- golangci-lint (install with `make setup`)

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Run linter
make lint
```

## Making Changes

### Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Ensure `make lint` passes
- Add tests for new functionality

### Commit Messages

- Use clear, descriptive commit messages
- Reference issues when applicable: "Fix #123: Add cluster validation"
- Keep commits focused and atomic

### Testing

- Write unit tests for new functions
- Update existing tests when modifying functionality
- Aim for >80% code coverage
- Run `make test-coverage` to check coverage

## Submitting Changes

1. Push your changes to your fork
2. Create a Pull Request against the `main` branch
3. Ensure all CI checks pass
4. Provide a clear description of changes
5. Link related issues

### PR Guidelines

- PRs should be focused on a single concern
- Include tests for new features
- Update documentation as needed
- Respond to review feedback promptly

## Project Structure

```
mcp-capi/
├── cmd/mcp-capi/       # Main application
├── pkg/                # Public packages
│   ├── capi/          # CAPI client code
│   ├── tools/         # MCP tool implementations
│   ├── resources/     # MCP resource handlers
│   └── prompts/       # MCP prompt definitions
├── internal/           # Private packages
├── docs/              # Documentation
└── examples/          # Usage examples
```

## Adding New Features

### Adding a Tool

1. Create tool definition in `pkg/tools/`
2. Implement handler function
3. Register tool in server initialization
4. Add tests
5. Update documentation

### Adding a Resource

1. Create resource handler in `pkg/resources/`
2. Define URI scheme
3. Register resource in server
4. Add tests
5. Document the resource

### Adding a Prompt

1. Create prompt definition in `pkg/prompts/`
2. Implement prompt handler
3. Register prompt in server
4. Add examples
5. Update documentation

## Documentation

- Update README.md for user-facing changes
- Add/update package documentation
- Include examples for new features
- Keep architecture docs current

## Release Process

Releases are automated via GitHub Actions when a tag is pushed:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push upstream v0.1.0
```

## Getting Help

- Check existing issues and PRs
- Join our community discussions
- Ask questions in issues

## Recognition

Contributors will be recognized in our releases and documentation.

Thank you for contributing! 