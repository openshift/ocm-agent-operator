---
name: run-tests
description: Run linting, unit tests, and race detection on the codebase
---

Run the full test suite for the OCM Agent Operator to verify code quality and correctness.

## Process

Execute the following steps in sequence:

1. **Run linting**
   ```bash
   make lint
   ```
   This runs golangci-lint with the boilerplate configuration to check code quality and style.

2. **Run unit tests**
   ```bash
   make go-test
   ```
   This runs all unit tests using the Ginkgo framework.

3. **Run race detection**
   ```bash
   go test -race ./...
   ```
   This checks for race conditions in concurrent code.

## Output

Report the results of each step clearly:
- ✅ if the step passed
- ❌ if the step failed (include relevant error details)

If any step fails, stop and report the failure before proceeding to the next step.

## Context

This skill is useful before:
- Creating a pull request
- Committing changes
- After making code modifications

Additional arguments: $ARGUMENTS
