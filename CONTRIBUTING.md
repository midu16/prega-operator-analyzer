# Contributing to Prega Operator Analyzer

Thank you for your interest in contributing to Prega Operator Analyzer! We welcome contributions from the community and are grateful for your support.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
  - [Reporting Bugs](#reporting-bugs)
  - [Suggesting Enhancements](#suggesting-enhancements)
  - [Submitting Pull Requests](#submitting-pull-requests)
- [Development Setup](#development-setup)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Review Process](#review-process)
- [Community](#community)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

### Our Standards

- **Be Respectful**: Treat everyone with respect and kindness
- **Be Collaborative**: Work together constructively
- **Be Professional**: Keep discussions focused and professional
- **Be Inclusive**: Welcome diverse perspectives and experiences
- **Be Patient**: Remember that everyone was a beginner once

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/prega-operator-analyzer.git
   cd prega-operator-analyzer
   ```
3. **Add the upstream repository**:
   ```bash
   git remote add upstream https://github.com/ORIGINAL_OWNER/prega-operator-analyzer.git
   ```
4. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## How to Contribute

### Reporting Bugs

Bug reports help us improve the project. When filing a bug report, please include:

#### Before Submitting a Bug Report

- **Check existing issues** to see if the problem has already been reported
- **Use the latest version** to verify the bug still exists
- **Check the documentation** to ensure you're using the tool correctly

#### How to Submit a Good Bug Report

1. **Use the bug report template** (if available)
2. **Provide a clear title** that summarizes the issue
3. **Describe the bug** in detail:
   - What you expected to happen
   - What actually happened
   - Steps to reproduce the issue
4. **Include environment details**:
   - Go version (`go version`)
   - Operating system and version
   - Container runtime (if using containers)
   - Prega Operator Analyzer version
5. **Attach logs and screenshots** if applicable
6. **Add relevant labels** (bug, high-priority, etc.)

#### Bug Report Example

```markdown
**Title**: Permission denied when running in container with mounted volume

**Description**:
When running the container with a mounted volume, I get a permission denied error.

**Steps to Reproduce**:
1. Run: `podman run -v $(pwd)/output:/app/output prega-operator-analyzer:latest`
2. Observe error: "permission denied: /app/output/release-notes.txt"

**Expected Behavior**:
The container should create the output file successfully.

**Environment**:
- OS: Fedora 38
- Podman version: 4.5.0
- Container image: quay.io/midu/prega-operator-analyzer:latest

**Logs**:
[Attach relevant logs here]
```

### Suggesting Enhancements

We welcome ideas for new features and improvements!

#### Before Submitting an Enhancement

- **Check existing issues** for similar suggestions
- **Review the roadmap** to see if it's already planned
- **Consider the scope** - will it benefit the majority of users?

#### How to Submit an Enhancement Suggestion

1. **Open a new issue** with the "enhancement" label
2. **Provide a clear title** describing the enhancement
3. **Explain the motivation**:
   - What problem does it solve?
   - What use case does it address?
4. **Describe the proposed solution** in detail
5. **Provide examples** or mockups if applicable
6. **Consider alternatives** - describe other approaches you've considered

### Submitting Pull Requests

Pull requests are the best way to propose changes to the codebase.

#### Pull Request Process

1. **Ensure your fork is up to date**:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**:
   - Write clear, readable code
   - Follow the coding standards
   - Add tests for new functionality
   - Update documentation as needed

4. **Test your changes**:
   ```bash
   # Run unit tests
   make test
   
   # Run all workflow tests
   make test-all
   
   # Build and test container
   make container-functional-test
   ```

5. **Commit your changes**:
   ```bash
   git add .
   git commit -m "feat: add new feature description"
   ```

6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Open a Pull Request**:
   - Go to the original repository on GitHub
   - Click "New Pull Request"
   - Select your branch
   - Fill out the PR template

#### Pull Request Guidelines

- **One feature per PR**: Keep pull requests focused on a single feature or fix
- **Update tests**: Add or update tests for your changes
- **Update documentation**: Ensure README and other docs are current
- **Follow commit conventions**: Use conventional commit messages
- **Keep it small**: Smaller PRs are easier to review and merge
- **Respond to feedback**: Address review comments promptly
- **Rebase if needed**: Keep your branch up to date with main

#### Pull Request Template

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## How Has This Been Tested?
Describe the tests you ran to verify your changes

## Checklist
- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published
```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Podman (for container testing)
- Make

### Setup Steps

1. **Install dependencies**:
   ```bash
   make deps
   ```

2. **Build the project**:
   ```bash
   make build
   ```

3. **Run tests**:
   ```bash
   make test
   ```

4. **Run the application locally**:
   ```bash
   ./bin/prega-operator-analyzer --help
   ```

### Development Tools

- **OPM**: Install for testing index operations
  ```bash
  make install-opm
  ```

- **Vibe-tools** (optional): For enhanced release notes
  ```bash
  make install-vibe-tools
  ```

## Coding Standards

### Go Style Guide

- Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` to format your code
- Run `go vet` to check for common mistakes
- Use meaningful variable and function names
- Keep functions small and focused (ideally < 50 lines)
- Add comments for exported functions and types

### Code Organization

```
prega-operator-analyzer/
├── cmd/              # Command-line interface
├── pkg/              # Library code
│   ├── errors.go     # Error handling
│   ├── formatter.go  # Release note formatting
│   ├── parser.go     # Index parsing
│   └── vibe_tools.go # Vibe tools integration
├── testdata/         # Test data files
└── .github/          # CI/CD workflows
```

### Error Handling

- Use custom error types from `pkg/errors.go`
- Wrap errors with context using `WrapError()`
- Log errors appropriately using logrus
- Return errors to the caller when possible

### Example

```go
// Good
func ProcessRepository(repo string) error {
    if repo == "" {
        return WrapError(nil, ErrorTypeValidation, "repository URL cannot be empty", nil)
    }
    
    result, err := cloneRepository(repo)
    if err != nil {
        return WrapError(err, ErrorTypeGit, "failed to clone repository", map[string]interface{}{
            "repository": repo,
        })
    }
    
    return nil
}

// Bad
func ProcessRepository(repo string) error {
    result, err := cloneRepository(repo)
    if err != nil {
        return err  // No context
    }
    return nil
}
```

## Testing Guidelines

### Writing Tests

- Write tests for all new functionality
- Aim for >80% code coverage
- Use table-driven tests for multiple test cases
- Mock external dependencies

### Test Structure

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "result",
            wantErr:  false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("FunctionName() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test
go test -v ./pkg -run TestParseName

# Run all workflow tests
make test-all
```

## Commit Message Guidelines

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Changes that don't affect code meaning (formatting, etc.)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvement
- **test**: Adding or updating tests
- **chore**: Changes to build process or auxiliary tools
- **ci**: Changes to CI configuration files and scripts

### Examples

```
feat(parser): add support for NDJSON format

Added support for parsing newline-delimited JSON files in addition
to regular JSON format. This enables processing of large operator
indexes more efficiently.

Closes #123
```

```
fix(container): resolve permission issues with mounted volumes

Updated startup script to properly handle volume mount permissions
by adding chmod 777 to test output directories.

Fixes #456
```

```
docs(readme): update installation instructions

Added detailed steps for installing on different platforms and
clarified prerequisites.
```

## Review Process

### What to Expect

1. **Initial Review**: A maintainer will review your PR within 3-5 business days
2. **Feedback**: You may receive requests for changes or clarifications
3. **Iteration**: Make requested changes and push updates to your branch
4. **Approval**: Once approved, a maintainer will merge your PR
5. **Release**: Your changes will be included in the next release

### Review Criteria

- Code quality and style
- Test coverage
- Documentation completeness
- Backwards compatibility
- Performance impact
- Security considerations

## Community

### Getting Help

- **GitHub Issues**: For bugs and feature requests
- **Discussions**: For questions and general discussion
- **Documentation**: Check the [README.md](README.md) first

### Maintainers

Current maintainers of this project:
- [List maintainer names and GitHub handles]

### Recognition

Contributors will be recognized in:
- Project README
- Release notes
- GitHub contributors page

Thank you for contributing to Prega Operator Analyzer! 🎉

## License

By contributing to Prega Operator Analyzer, you agree that your contributions will be licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.


