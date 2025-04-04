# Contributing to FileKeeper

Thank you for your interest in contributing to FileKeeper! This document provides guidelines and instructions for contributing.

## Development Setup

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/yourusername/filekeeper.git
   cd filekeeper
   ```

2. **Initialize Go modules**
   ```bash
   go mod init github.com/ykargin/filekeeper
   go mod tidy
   ```

3. **Install dependencies**
   ```bash
   go get gopkg.in/yaml.v3
   ```

## Running Tests

FileKeeper includes a comprehensive test suite. You can run all tests with:

```bash
go test -v ./...
```

Or use the provided convenience script:

```bash
./run_tests.sh
```

### Test Coverage

To get coverage metrics:

```bash
go test -coverprofile=coverage.txt -covermode=atomic ./...
go tool cover -html=coverage.txt -o coverage.html
```

Then open `coverage.html` in your browser to see the coverage report.

## Code Standards

- Follow standard Go coding conventions and formatting using `gofmt`
- Ensure meaningful variable and function names
- Add appropriate comments for public functions and types
- Use English for all comments and documentation
- Use the project's linters (run via GitHub Actions)

## Adding Tests

When adding new features or fixing bugs, please include appropriate tests:

1. **Unit tests**: Test individual functions in isolation
2. **Integration tests**: Test interactions between components
3. **Positive cases**: Test expected/normal behavior
4. **Negative cases**: Test error handling and edge cases

The current tests cover:
- Configuration loading and validation
- File retention period parsing
- Directory processing and file deletion logic
- Secure deletion functionality
- Empty directory detection and removal
- Systemd service file generation

## Pull Request Process

1. Create a new branch for your feature or bugfix
2. Add appropriate tests for your changes
3. Ensure all tests pass by running `./run_tests.sh`
4. Commit your changes with clear commit messages
5. Open a pull request against the `dev` branch
6. Wait for CI tests to pass and for review

## Versioning

FileKeeper follows [Semantic Versioning](https://semver.org/):

- MAJOR version for incompatible API changes
- MINOR version for new functionality in a backward-compatible manner
- PATCH version for backward-compatible bug fixes

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Focus on the best outcome for the project and its users

Thank you for your contributions!