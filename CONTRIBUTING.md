# Contributing to OCX Protocol

Thank you for your interest in contributing to OCX Protocol! This document provides guidelines and instructions for contributing.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Testing](#testing)
- [Documentation](#documentation)
- [Community](#community)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Rust 1.70 or later (for verification library)
- Linux environment (Ubuntu 22.04+ recommended)
- Git
- Docker (optional, for integration tests)

### Setting Up Development Environment

```bash
# Clone the repository
git clone https://github.com/KuroKernel/ocx-protocol.git
cd ocx-protocol

# Install Go dependencies
go mod download

# Build all components
make build

# Run tests
make test

# Run smoke tests
./scripts/smoke.sh
```

## Development Workflow

### 1. Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/ocx-protocol.git
   cd ocx-protocol
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/KuroKernel/ocx-protocol.git
   ```

### 2. Create a Branch

Create a branch for your work:

```bash
git checkout -b feature/your-feature-name
```

Branch naming conventions:
- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions or modifications

### 3. Make Changes

- Write clean, readable code
- Follow existing code style
- Add tests for new functionality
- Update documentation as needed
- Keep commits focused and atomic

### 4. Test Your Changes

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Run smoke tests
./scripts/smoke.sh

# Check code formatting
gofmt -s -w .
go vet ./...
```

### 5. Commit Your Changes

Follow our [commit guidelines](#commit-guidelines).

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## Coding Standards

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting
- Run `go vet` and address warnings
- Keep functions focused and small
- Write meaningful variable and function names
- Add comments for exported functions and types

Example:

```go
// VerifyReceipt checks the cryptographic validity of an execution receipt.
// It returns true if the receipt signature is valid and matches the public key.
func VerifyReceipt(receipt *Receipt, publicKey []byte) (bool, error) {
    if receipt == nil {
        return false, fmt.Errorf("receipt cannot be nil")
    }

    // Implementation...
}
```

### Rust Code

- Follow [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/)
- Use `cargo fmt` for formatting
- Run `cargo clippy` and address warnings
- Write idiomatic Rust code

### General Principles

- **DRY**: Don't Repeat Yourself
- **KISS**: Keep It Simple, Stupid
- **YAGNI**: You Aren't Gonna Need It
- **Separation of Concerns**: Keep components focused
- **Error Handling**: Always handle errors explicitly

## Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Build process or auxiliary tool changes
- `perf`: Performance improvements
- `ci`: CI/CD changes

### Examples

```
feat(api): Add rate limiting to execution endpoint

Implement token bucket algorithm for rate limiting API requests.
Prevents DoS attacks by limiting to 10 requests/second per client.

Closes #123
```

```
fix(receipt): Correct timestamp precision in receipts

Changed from nanosecond to millisecond precision to ensure
deterministic receipt generation across different systems.

Fixes #456
```

```
docs: Update README with Docker deployment instructions

Added comprehensive Docker and Kubernetes deployment examples
to help users deploy OCX Protocol in containerized environments.
```

### Commit Message Rules

- Use imperative mood ("Add feature" not "Added feature")
- First line should be ≤ 72 characters
- Body should wrap at 72 characters
- Separate subject from body with blank line
- Reference issues and PRs in footer

## Pull Request Process

### Before Submitting

1. **Update documentation** for any changed functionality
2. **Add tests** for new features or bug fixes
3. **Run full test suite** and ensure all tests pass
4. **Update CHANGELOG.md** with your changes
5. **Rebase on latest main** to avoid merge conflicts

### PR Requirements

- Descriptive title following commit conventions
- Clear description of changes and motivation
- Reference related issues
- All CI checks must pass
- At least one approving review required
- Up-to-date with main branch

### PR Template

```markdown
## Description

Brief description of changes.

## Motivation and Context

Why is this change required? What problem does it solve?

## Type of Change

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update

## How Has This Been Tested?

Describe the tests you ran and their results.

## Checklist

- [ ] My code follows the code style of this project
- [ ] I have added tests to cover my changes
- [ ] All new and existing tests passed
- [ ] I have updated the documentation accordingly
- [ ] I have added an entry to CHANGELOG.md
```

### Review Process

1. Maintainers will review within 1-2 business days
2. Address review comments promptly
3. Keep discussions professional and constructive
4. Once approved, maintainers will merge

## Testing

### Test Requirements

- **Unit tests** for all new functions and methods
- **Integration tests** for API endpoints and workflows
- **Table-driven tests** for functions with multiple cases
- **Error cases** must be tested
- **Edge cases** should be covered

### Writing Tests

```go
func TestVerifyReceipt(t *testing.T) {
    tests := []struct {
        name        string
        receipt     *Receipt
        publicKey   []byte
        wantValid   bool
        wantErr     bool
    }{
        {
            name:      "valid receipt",
            receipt:   validReceipt,
            publicKey: validPublicKey,
            wantValid: true,
            wantErr:   false,
        },
        {
            name:      "tampered receipt",
            receipt:   tamperedReceipt,
            publicKey: validPublicKey,
            wantValid: false,
            wantErr:   false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            valid, err := VerifyReceipt(tt.receipt, tt.publicKey)
            if (err != nil) != tt.wantErr {
                t.Errorf("VerifyReceipt() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if valid != tt.wantValid {
                t.Errorf("VerifyReceipt() = %v, want %v", valid, tt.wantValid)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./pkg/receipt/...

# Run with race detector
go test -race ./...
```

## Documentation

### What to Document

- **Public APIs**: All exported functions, types, and methods
- **Configuration**: Environment variables and configuration options
- **Examples**: Usage examples for new features
- **Architecture**: Design decisions and system architecture
- **Deployment**: Installation and deployment instructions

### Documentation Style

- Use clear, concise language
- Provide code examples where helpful
- Include diagrams for complex concepts
- Keep documentation up-to-date with code changes

### Updating Documentation

- Update inline code comments (godoc)
- Update README.md for user-facing changes
- Add entries to docs/ for detailed explanations
- Update API documentation for endpoint changes

## Community

### Getting Help

- **GitHub Issues**: For bug reports and feature requests
- **GitHub Discussions**: For questions and general discussion
- **Email**: contact@ocx.world for private inquiries

### Reporting Issues

When reporting issues, include:

1. **Description**: Clear description of the issue
2. **Steps to Reproduce**: Detailed steps to reproduce the problem
3. **Expected Behavior**: What you expected to happen
4. **Actual Behavior**: What actually happened
5. **Environment**: OS, Go version, OCX version, etc.
6. **Logs**: Relevant log output or error messages

### Feature Requests

When proposing features:

1. **Use Case**: Describe the problem you're trying to solve
2. **Proposed Solution**: Your suggested approach
3. **Alternatives**: Other approaches you considered
4. **Impact**: Who benefits and how

## Security

### Reporting Security Vulnerabilities

**Do not** open public issues for security vulnerabilities.

Instead:
1. Email security@ocx.world
2. Include detailed description and proof of concept
3. We will respond within 48 hours
4. Allow time for fix before public disclosure

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to OCX Protocol!
