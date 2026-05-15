# Claude Code Hooks

Security and validation hooks for OCM Agent Operator development.

## Overview

This repository uses **prek** (git hook manager) for quality checks and validation. Claude Code hooks integrate with prek to provide immediate feedback during development.

## Architecture

```
┌─────────────────────────────────────┐
│   Developer / Claude Code Agent     │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   Stop Hook (every turn)            │
│   - Runs prek validation            │
│   - Blocks if issues found          │
│   - Claude fixes automatically      │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   Prek Hooks (CI config)            │
│   - golangci-lint (static analysis) │
│   - RBAC wildcard check             │
│   - go build validation             │
│   - go mod tidy check               │
│   - file hygiene (trailing space)  │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   Prek Hooks (full config)          │
│   + rh-pre-commit (InfoSec)         │
│   + gitleaks (secret scanning)      │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   Git Commit                         │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│   CI/CD (Tekton Pipelines)          │
└─────────────────────────────────────┘
```

## Available Hooks

### [stop-prek-validation.sh](./stop-prek-validation.sh)
**Purpose**: Run prek validation every time Claude Code stops

**Triggers**: On Claude Code session stop (Stop hook)

**Behavior**:
- Runs `prek run --config hack/prek.ci.toml --all-files` automatically
- Uses CI-compatible config (skips network-dependent hooks like rh-pre-commit, gitleaks)
- Blocks Claude from stopping if issues found
- Feeds errors back to Claude for automatic fixes
- Includes infinite loop guard (allows stop on retry)

**Benefits**:
- Catches issues between prompts (not just at commit time)
- Enables longer autonomous work without human intervention
- Shortens feedback loop for quality checks

**Installation**: Configured in `.claude/settings.json`

---

### [pre-edit.sh](./pre-edit.sh)
**Purpose**: Prevent editing generated files and warn about high-risk changes

**Status**: Available for standalone use (not configured as Claude Code hook)

**Checks**:
- Generated files (`zz_generated.*.go`)
- Generated mocks (`**/generated/mock_*.go`)
- Vendored code (`vendor/`)
- Boilerplate files (managed upstream)
- High-risk security files (RBAC, auth, NetworkPolicy)
- CI/CD pipelines (`.tekton/*.yaml`)
- Dockerfiles

**Manual Usage**:
```bash
.claude/hooks/pre-edit.sh path/to/file.go
```

---

### [secret-scanner.sh](./secret-scanner.sh)
**Purpose**: Legacy secret scanner script

**Status**: Reference implementation (superseded by gitleaks in prek)

**Modern Alternative**: Use `prek run gitleaks` or configure in `prek.toml`

---

## Prek Configuration

This repository maintains **two prek configurations**:

### 1. **prek.toml** (Full validation)
Used for local development with internal network access.

**Hooks**:
- File hygiene (trailing whitespace, EOF, syntax checks)
- **rh-pre-commit**: Red Hat InfoSec security checks (requires `gitlab.cee.redhat.com` access)
- **gitleaks**: Secret detection (configured via `.gitleaks.toml`)
- **golangci-lint**: Static analysis
- **go-build**: Compile check
- **go-mod-tidy**: Dependency drift detection
- **rbac-wildcard-check**: RBAC validation

**Usage**:
```bash
prek run --all-files
```

### 2. **hack/prek.ci.toml** (CI-compatible)
Used by Claude Code stop hook and CI environments without internal network access.

**Excludes**:
- `rh-pre-commit` (requires Red Hat internal network)
- `gitleaks` (may not be available in all CI environments)

**Usage**:
```bash
hack/ci.sh
# or
prek run --config hack/prek.ci.toml --all-files
```

**Why two configs?**
The CI-compatible config allows Claude Code and external CI systems to run quality checks without requiring access to Red Hat's internal GitLab instance.

## Setup

### Prerequisites
```bash
# Install prek (choose one)
uv tool install prek      # recommended
pipx install prek         # alternative
pip install --user prek   # fallback
```

### Install Git Hooks
```bash
prek install
```

This sets up pre-commit hooks that run validation automatically.

## Usage

### Automatic Validation
Prek runs automatically:
- **On every turn**: Stop hook runs `prek run --all-files`
- **On commit**: Pre-commit hook runs relevant checks

### Manual Validation
```bash
# Run all checks
prek run --all-files

# Run specific check
prek run gitleaks
prek run golangci-lint
prek run rbac-wildcard-check
```

## Hook Categories

### Stop Hooks
**Purpose**: Validate before Claude Code stops

**Current**:
- `stop-prek-validation.sh`: Run prek checks

**Benefits**:
- Immediate feedback (not delayed until commit)
- Automatic fixes by Claude
- Prevents accumulation of violations

### Pre-commit Hooks
**Purpose**: Validate before git commit

**Managed by**: prek (configured in `prek.toml`)

**Checks**:
- File hygiene and syntax
- Security scanning (rh-pre-commit, gitleaks)
- Static analysis (golangci-lint)
- Build validation (go build, go mod tidy)
- Custom checks (RBAC wildcards)

## Security Guardrails

### Secret Prevention
**Implementation**: gitleaks via prek

**Configuration**: `.gitleaks.toml`

**Detects**:
- AWS credentials
- GitHub tokens
- API keys
- Private keys
- Database connection strings
- OCM-specific tokens
- High-entropy secrets

**Action**: BLOCK commit

### InfoSec Scanning
**Implementation**: rh-pre-commit via prek

**Source**: Red Hat InfoSec Developer Workbench

**Checks**: Internal security policies and compliance

**Action**: BLOCK commit on violations

### RBAC Validation
**Implementation**: rbac-wildcard-check via prek

**Detects**:
- Wildcard resources: `["*"]`
- Wildcard verbs: `["*"]`

**Action**: BLOCK commit

### Generated File Protection
**Implementation**: pre-edit.sh (standalone)

**Detects**:
- `zz_generated.*.go`
- Generated mocks
- CRD manifests

**Action**: BLOCK edit (suggest regeneration)

## Hook Performance

**Targets:**
- Stop hook: <30s for full validation
- Pre-commit hook: <30s on typical changeset
- Individual checks: <10s each

**Optimization:**
- Prek runs hooks in parallel where possible
- Hooks only check changed files (where applicable)
- Build artifacts cached between runs

## Troubleshooting

### Hook Not Running
```bash
# Verify prek is installed
prek --version

# Reinstall git hooks
prek install

# Check hook configuration
cat prek.toml
```

### Hook Fails Incorrectly
```bash
# Run hook manually for debugging
prek run <hook-id> --verbose

# Check hook configuration
cat prek.toml

# Update prek
uv tool upgrade prek  # or pipx upgrade prek
```

### Bypass Hook (Emergency Only)
```bash
# Skip specific hook
SKIP=hook-id git commit

# NEVER use (bypasses all validation)
git commit --no-verify  # FORBIDDEN
```

**Security hooks (gitleaks, rh-pre-commit) should NEVER be bypassed.**

## Version Management

### Prek Version
Pinned in `.prek-version` for CI consistency:
```bash
cat .prek-version  # v0.3.9
```

Update when new prek releases are available.

### Hook Dependencies
Defined in `prek.toml` with immutable refs:
- `rh-pre-commit-2.3.0`
- `v8.18.0` (gitleaks)
- `v2.0.2` (golangci-lint)

## References

- [Prek Documentation](https://prek.j178.dev/)
- [Gitleaks](https://github.com/gitleaks/gitleaks)
- [RH InfoSec Tools](https://gitlab.cee.redhat.com/infosec-public/developer-workbench/tools)
- [golangci-lint](https://golangci-lint.run/)
- [CLAUDE.md](../../CLAUDE.md) - Development guidelines
