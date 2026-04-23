---
name: run-tests
description: Run all tests including unit tests, linting, and race detection
---

# Run Tests

Comprehensive test suite execution for the operator.

## Usage

```
/run-tests
```

Or with specific options:
```
/run-tests with race detection
/run-tests quick  # Skip integration tests
/run-tests verbose
```

## What Gets Run

### 1. Linting
```bash
make lint
```
Runs golangci-lint with the configuration from `boilerplate/openshift/golang-osd-operator/golangci.yml`.

### 2. Unit Tests
```bash
make test
```
Runs all unit tests with coverage reporting.

### 3. Race Detection (Optional)
```bash
go test -race ./...
```
Detects data races and concurrency issues.

### 4. Build Verification
```bash
make go-build
```
Ensures the operator builds successfully.

## Test Levels

### Quick Tests (Default)
```bash
make lint
make test
make go-build
```
**Time**: ~2-5 minutes  
**Use case**: Before every commit

### Full Tests (Recommended before PR)
```bash
make lint
make test
make coverage  # With coverage report
go test -race ./...
make go-build
```
**Time**: ~5-10 minutes  
**Use case**: Before creating PR

### Integration Tests (If available)
```bash
make test-integration
```
**Time**: ~10-30 minutes  
**Use case**: Before merging to master

## Interpreting Results

### Linting Failures

**CRITICAL** (Must fix):
- `errcheck`: Unchecked errors
- `contextcheck`: Missing context propagation
- `bodyclose`: Unclosed HTTP response bodies
- `noctx`: Missing context in HTTP requests

**HIGH** (Should fix):
- `errorlint`: Improper error wrapping
- `nilerr`: Returning nil error incorrectly
- `nilnil`: Returning nil, nil

**MEDIUM** (Fix if trivial):
- Code style issues
- Unused variables
- Simplifications

### Test Failures

Check the output for:
- **File and line number** of failure
- **Expected vs actual** values
- **Stack trace** for panics

Common causes:
- Missing mock expectations
- Incorrect assertions
- Resource not properly set up
- Context timeout

### Race Detection

If `-race` finds issues:
- **File and line numbers** of conflicting accesses
- **Goroutine IDs** and call stacks
- **Read/write conflicts** on shared variables

**Action**: Fix immediately - races are critical bugs

## Continuous Testing

### Watch Mode
```bash
# Install gotestsum first: go install gotest.tools/gotestsum@latest
gotestsum --watch -- -short ./...
```

### Pre-commit Hook
Tests run automatically before commits via `.claude/hooks/before-commit`.

## Test Coverage

### View Coverage Report
```bash
make coverage
# Opens coverage.html in browser
```

### Coverage Targets
- **Overall**: >80% preferred
- **Controllers**: >85% required
- **Core logic**: >90% required
- **Generated code**: Can be excluded

## Troubleshooting

### "Cannot find package"
```bash
go mod download
go mod tidy
```

### "Build failed"
```bash
make go-build VERBOSE=1  # See detailed errors
```

### "Test timeout"
Increase timeout:
```bash
go test -timeout 10m ./...
```

### "Race detector overhead"
Run races on subset:
```bash
go test -race ./controllers/...  # Just controllers
```

## CI/CD Integration

These tests also run in CI via:
- `.tekton/` pipelines (Konflux)
- GitHub Actions (if configured)
- Pre-commit hooks

**Note**: Make sure tests pass locally before pushing to avoid CI failures.

## Quick Reference

```bash
# Before commit
make lint && make test

# Before PR
make lint && make test && go test -race ./...

# Check coverage
make coverage

# Run specific test
go test -v ./controllers -run TestReconcile

# Run with race detector
go test -race ./...

# Verbose output
go test -v ./...

# Only fast tests
go test -short ./...
```
