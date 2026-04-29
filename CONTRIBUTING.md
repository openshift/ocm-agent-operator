# Contributing to OCM Agent Operator

Thank you for your interest in contributing to the OCM Agent Operator! This document provides guidelines for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [How to Contribute](#how-to-contribute)
- [Development Workflow](#development-workflow)
- [Pull Request Process](#pull-request-process)
- [Code Style and Standards](#code-style-and-standards)
- [Testing Requirements](#testing-requirements)
- [Review Process](#review-process)

## Code of Conduct

This project follows the [OpenShift Community Code of Conduct](https://github.com/openshift/community/blob/main/CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

Before contributing, please:

1. **Read the documentation**:
   - [README.md](./README.md) - Project overview
   - [DEVELOPMENT.md](./DEVELOPMENT.md) - Development setup
   - [TESTING.md](./TESTING.md) - Testing guidelines
   - [docs/design.md](./docs/design.md) - Architecture and design

2. **Set up your development environment**:
   - Install required tools (see [DEVELOPMENT.md](./DEVELOPMENT.md))
   - Fork the repository
   - Clone your fork locally

3. **Join the community**:
   - Check existing issues and pull requests
   - Ask questions in discussions or relevant Slack channels

## How to Contribute

### Reporting Bugs

If you find a bug:

1. **Check existing issues** to avoid duplicates
2. **Create a new issue** with:
   - Clear, descriptive title
   - Steps to reproduce
   - Expected vs. actual behavior
   - Environment details (Kubernetes version, operator version, etc.)
   - Relevant logs or error messages

### Suggesting Enhancements

For feature requests or enhancements:

1. **Check existing issues** for similar proposals
2. **Create a new issue** describing:
   - The problem you're trying to solve
   - Your proposed solution
   - Alternative approaches considered
   - Impact on existing functionality

### Contributing Code

1. **Find or create an issue** for the work you want to do
2. **Comment on the issue** to let others know you're working on it
3. **Fork and clone** the repository
4. **Create a branch** from `master` for your work
5. **Make your changes** following our coding standards
6. **Write or update tests** for your changes
7. **Run tests locally** to ensure everything passes
8. **Submit a pull request** (see PR process below)

## Development Workflow

### Setting Up Your Fork

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/ocm-agent-operator.git
cd ocm-agent-operator

# Add upstream remote
git remote add upstream https://github.com/openshift/ocm-agent-operator.git

# Verify remotes
git remote -v
```

### Keeping Your Fork Updated

```bash
# Fetch upstream changes
git fetch upstream

# Update your master branch
git checkout master
git merge upstream/master

# Push to your fork
git push origin master
```

### Creating a Feature Branch

```bash
# Create and switch to a new branch
git checkout -b feature/my-feature-name

# Or for bug fixes
git checkout -b fix/issue-number-description
```

**Branch naming conventions**:
- Features: `feature/short-description` or `JIRA-123-short-description`
- Bug fixes: `fix/issue-number-description`
- Documentation: `docs/description`
- Refactoring: `refactor/description`

### Making Changes

1. **Install dependencies**:
   ```bash
   make tools
   ```

2. **Make your changes** in the codebase

3. **Run code generation** if you modified API types:
   ```bash
   make generate
   make manifests
   ```

4. **Run tests**:
   ```bash
   make go-test
   make lint
   ```

5. **Test in a cluster** (recommended):
   ```bash
   make run  # Against your configured cluster
   ```

## Pull Request Process

### Before Submitting

Ensure your PR:

- [ ] Has a clear, descriptive title
- [ ] References related issues (e.g., "Fixes #123" or "Relates to #456")
- [ ] Includes appropriate tests
- [ ] Passes all tests locally (`make go-test`)
- [ ] Passes linting (`make lint`)
- [ ] Updates documentation if needed
- [ ] Follows code style guidelines
- [ ] Has meaningful commit messages

### PR Title Format

Use conventional commit style:

```
<type>(<scope>): <short summary>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `chore`: Build process, dependency updates
- `perf`: Performance improvements

**Examples**:
- `feat(controller): add support for fleet mode configuration`
- `fix(metrics): correct pull secret validation metric`
- `docs: update README with deployment instructions`
- `test(handler): add unit tests for ConfigMap generation`

### PR Description Template

```markdown
## Description
Brief description of the changes and motivation.

## Related Issues
Fixes #<issue_number>
Relates to #<issue_number>

## Changes Made
- List of key changes
- Impact on existing functionality
- Breaking changes (if any)

## Testing Done
- [ ] Unit tests added/updated
- [ ] E2E tests added/updated (if applicable)
- [ ] Tested manually in a cluster
- [ ] All tests pass locally

## Checklist
- [ ] Code follows project style guidelines
- [ ] Documentation updated
- [ ] Commits are signed-off (DCO)
- [ ] Ready for review
```

### Commit Messages

Follow these guidelines:

1. **Use the imperative mood** ("Add feature" not "Added feature")
2. **Keep the first line under 72 characters**
3. **Add detailed description** in the body if needed
4. **Reference issues** in the footer

Example:
```
feat(controller): add cluster proxy configuration support

Automatically inject HTTP_PROXY, HTTPS_PROXY, and NO_PROXY
environment variables into the OCM Agent deployment based on
the cluster-wide proxy configuration.

Fixes #123
```

### Developer Certificate of Origin (DCO)

All commits must be signed off to indicate you agree to the [Developer Certificate of Origin](https://developercertificate.org/):

```bash
git commit -s -m "Your commit message"
```

The `-s` flag adds the sign-off line automatically.

## Code Style and Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting (automatic via linting)
- Run `make lint` to check code quality
- Keep functions focused and small
- Add comments for exported functions and types
- Use meaningful variable and function names

### Testing Standards

- Write tests for all new functionality
- Maintain or improve code coverage
- Use Ginkgo/Gomega for test structure
- Mock external dependencies using GoMock
- Follow existing test patterns in the codebase

See [TESTING.md](./TESTING.md) for detailed testing guidelines.

### Documentation Standards

- Update documentation when adding/changing features
- Keep code comments up to date
- Document complex logic or non-obvious behavior
- Update API documentation for CRD changes

## Testing Requirements

### Required Tests

All PRs must include:

1. **Unit tests** for new code
2. **Updated tests** for modified code
3. **E2E tests** for significant features (when applicable)

### Running Tests

```bash
# Run all unit tests
make go-test

# Run tests in container (matches CI)
boilerplate/_lib/container-make test

# Run linting
make lint

# Run specific package tests
go test ./pkg/ocmagenthandler/...
```

### Test Coverage

- Maintain or improve overall test coverage
- Aim for meaningful tests, not just coverage numbers
- Test both success and failure scenarios
- Test edge cases and error handling

## Review Process

### What to Expect

1. **Automated checks** run on all PRs (linting, tests, coverage)
2. **Code review** by maintainers (see OWNERS file)
3. **Feedback and iteration** - be responsive to review comments
4. **Approval** required from designated reviewers
5. **Merge** once approved and all checks pass

### Review Timeline

- Initial review: Usually within 2-3 business days
- Follow-up reviews: Usually within 1-2 business days
- Complex changes may take longer

### Addressing Feedback

- Respond to all review comments
- Push additional commits to address feedback
- Mark conversations as resolved when addressed
- Re-request review after making changes

### Getting Help

If your PR is stuck or you need help:

- Comment on the PR asking for guidance
- Tag specific reviewers if needed
- Reach out in the relevant Slack channel
- Check project discussions or issues for similar questions

## Additional Resources

- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Kubernetes Operator Best Practices](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [OpenShift Documentation](https://docs.openshift.com/)
- [Go Testing Best Practices](https://golang.org/doc/code.html#Testing)

## Recognition

Contributors are recognized in several ways:

- Listed as contributors on the repository
- Mentioned in release notes for significant contributions
- Added to OWNERS file for consistent, high-quality contributions

Thank you for contributing to OCM Agent Operator!
