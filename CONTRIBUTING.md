# Contributing to Ocm Agent Operator

Thank you for your interest in contributing! This guide provides comprehensive instructions for setting up your development environment, running tests, and contributing code to this operator.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Prerequisites](#prerequisites)
- [Setup](#setup)
- [Pre-commit Hooks with prek](#pre-commit-hooks-with-prek)
- [Claude Code Integration](#claude-code-integration)
- [Commit Message Conventions](#commit-message-conventions)
- [PR Process & Review Requirements](#pr-process--review-requirements)
- [Validation and Linting](#validation-and-linting)
- [Testing](#testing)
- [Boilerplate Framework](#boilerplate-framework)
- [CI/CD Integration](#cicd-integration)


## Code of Conduct

As contributors and maintainers of this project, we are committed to making participation a harassment-free experience for everyone. In short, be excellent to each other.

## Prerequisites

Before contributing, ensure you have the following tools installed:

- **Go 1.22+**: [Installation guide](https://golang.org/doc/install)
- **golangci-lint**: Used for Go code linting.
- **[prek](https://prek.j178.dev/)**: Git hook manager that runs validation automatically on commit.

## Setup

```bash
# Install prek (macOS)
brew install prek

# Install prek (Linux)
curl -fsSL https://prek.j178.dev/install.sh | bash

# Install pre-commit hooks in this repository
prek install
```

`prek install` sets up git hooks that automatically run file hygiene checks and `golangci-lint` before each commit.

## Pre-commit Hooks with prek

The validation runs automatically via pre-commit hooks, or you can run it manually:

```bash
# Run all validations (file hygiene + golangci-lint)
prek run --all-files

# Run only golangci-lint via make
make go-check
```

## Claude Code Integration

### Stop Hook: prek Validation on Every Turn

If you use **Claude Code**, this repository includes a [stop hook](https://docs.anthropic.com/en/docs/claude-code/hooks) in `.claude/settings.json` that runs `prek run --all-files` every time Claude finishes a turn. 

If `prek` finds violations (trailing whitespace, linting errors, etc.), the hook **blocks Claude from stopping** and feeds the errors back so Claude can fix them automatically. This shortens the feedback loop and ensures high-quality output without manual intervention.

## Commit Message Conventions

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification for all commit messages.

### AI Attribution

When using AI assistants (like Claude, Gemini, or GitHub Copilot) to generate or significantly refactor code, you **MUST** include an attribution in the commit message trailer.

Use the `Co-authored-by` trailer format:

```text
feat(api): add new validation for PagerDutyIntegration

This commit adds a new validating webhook.

Co-authored-by: Claude <claude@anthropic.com>
```

## PR Process & Review Requirements

### Submission Guidelines
1. **Branching**: Create a feature branch for your changes.
2. **Tests**: New features and bug fixes should include appropriate unit or integration tests.
3. **Docs**: Update relevant documentation if changing user-facing behavior.

### Review Process
- **CI Passing**: All automated tests in PROW must pass.
- **Peer Review**: At least one `/lgtm` (Look Good To Me) from a maintainer is required.
- **Approval**: An `/approve` label from a designated code owner is required for merging.
- **Merge Bot**: Once approved and CI is green, the OpenShift Merge Bot will automatically merge the PR.

## Validation and Linting

This project uses `golangci-lint` configured via the boilerplate framework.

```bash
# Run full validation (boilerplate + lint)
make lint

# Run only golangci-lint
make go-check
```

## Testing

```bash
# Run unit tests
make test

# Run code coverage
make coverage
```


## Boilerplate Framework

This repository uses the [openshift/boilerplate](https://github.com/openshift/boilerplate/) framework. 

Key boilerplate targets:
- `make validate` — Check code generation and boilerplate consistency.
- `make lint` — Static analysis and YAML validation.
- `make test` — Run the standard test suite.

## CI/CD Integration

The project uses **OpenShift PROW** for continuous integration. Every pull request triggers a set of jobs defined in the `openshift/release` repository that execute `make validate`, `make lint`, and `make test`.
