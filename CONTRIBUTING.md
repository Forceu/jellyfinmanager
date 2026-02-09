# Contributing to Jellyfin Manager

Thank you for your interest in contributing to Jellyfin Manager! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions. We aim to maintain a welcoming and inclusive community.

## How to Contribute

### Reporting Bugs

Before creating bug reports, please check existing issues to avoid duplicates. When creating a bug report, include:

- **Clear title and description**
- **Steps to reproduce** the issue
- **Expected behavior** vs actual behavior
- **Environment details** (OS, Go version, Jellyfin version)
- **Error messages** or logs if applicable

### Suggesting Enhancements

Enhancement suggestions are welcome! Please provide:

- **Clear description** of the feature
- **Use case** - why would this be useful?
- **Possible implementation** ideas (optional)

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** with clear, descriptive commits
3. **Test your changes** thoroughly
4. **Update documentation** if needed
5. **Submit a pull request**

#### Pull Request Guidelines

- Keep changes focused - one feature/fix per PR
- Write clear commit messages
- Update README.md if adding features
- Ensure code follows Go conventions
- Add tests for new functionality when applicable

## Development Setup

### Prerequisites

- Go 1.25 or later
- Docker (for testing Docker builds)
- Access to a Jellyfin server for testing

### Building from Source

```bash
# Clone your fork
git clone https://github.com/forceu/jellyfinmanager.git
cd jellyfinmanager

# Build
go build -o jellyfinmanager

# Run
./jellyfinmanager -h
```

### Building Docker Image

```bash
# Build the image
docker build -t jellyfinmanager:dev .

# Test the image
docker run --rm jellyfinmanager:dev -h
```

### Testing

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...
```

## Project Structure

```
jellyfinmanager/
├── api/
│   ├── jellyfin/     # Jellyfin API client
│   └── tvdb/         # TVDB API client
├── models/           # Data structures
├── main.go           # CLI and orchestration
├── defaults.go       # Default values (non-docker)
└── defaults_docker.go # Default values (docker)
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` to format your code
- Keep functions focused and single-purpose
- Add comments for exported functions and types
- Use meaningful variable names

## Commit Messages

Write clear, descriptive commit messages:

```
Add support for custom backup locations

- Add -backup-dir flag to specify custom directory
- Update documentation with new flag
- Add validation for directory path
```

Format:
- First line: Brief summary (50 chars or less)
- Blank line
- Detailed description if needed (wrap at 72 chars)

## AI-Generated Code

This project includes AI-generated code. If you're adding AI-generated code:

1. Review and test it thoroughly
2. Ensure it meets our coding standards
3. Understand how it works
4. Mention AI usage in your PR description if significant

## Questions?

Feel free to open an issue for questions or discussion!

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (GPL License).
