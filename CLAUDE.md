# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The OCM Agent Operator is a Kubernetes operator that manages the OCM Agent service on managed OpenShift clusters. It's built using the Operator SDK framework and follows standard Kubernetes operator patterns.

### Architecture Summary

**Core Components:**
- **API Types** (`api/v1alpha1/`): Defines the OcmAgent, ManagedNotification, and ManagedFleetNotification custom resources
- **Controllers** (`controllers/`):
  - `ocmagent/`: Reconciles OcmAgent resources, manages deployments, services, secrets, etc.
  - `fleetnotification/`: Handles fleet-wide notification resources
- **OCM Agent Handler** (`pkg/ocmagenthandler/`): Business logic for managing OCM Agent resources (deployment, configmap, secrets, network policies, PDB, service monitor)

**Key Custom Resources:**
- **OcmAgent**: Main resource defining OCM agent configuration, image, token secret, replicas, and fleet mode
- **ManagedNotification**: Handles cluster-specific notifications
- **ManagedFleetNotification**: Manages fleet-wide notifications across multiple clusters

**Key Directories:**
- `api/v1alpha1/`: CRD definitions and types
- `controllers/`: Reconciliation logic
- `pkg/ocmagenthandler/`: Resource creation and management
- `pkg/util/`: Utilities and test helpers
- `deploy/`: Kubernetes manifests and CRDs
- `test/e2e/`: End-to-end tests

**Critical Invariants:**
- FIPS compliance required (`FIPS_ENABLED=true`)
- RBAC must never use wildcard permissions (`["*"]`)
- All secrets must be referenced, never embedded
- NetworkPolicies enforce strict isolation

## Development Commands

### Install
```bash
make tools                    # Install development tools (ginkgo, mockgen, controller-gen, etc.)
uv tool install prek          # Install prek (or: pipx install prek)
prek install                  # Install git hooks
```

### Build
```bash
make go-build                 # Build the operator binary
make docker-build             # Build container image
```

### Lint
```bash
make go-check                 # Run golangci-lint (comprehensive)
prek run --all-files          # Run all hooks (fast feedback)
prek run golangci-lint        # Lint only
hack/ci.sh                    # Run CI-compatible hooks (excludes network-dependent hooks)
```

**Note**: For CI environments, use `hack/ci.sh` which runs a CI-specific prek config (`hack/prek.ci.toml`) that excludes hooks requiring internal network access (rh-pre-commit, gitleaks).

### Typecheck
```bash
go build ./...                # Compile check (runs in prek hooks)
```

### Test
```bash
make go-test                  # Run all unit tests
ginkgo -r ./...               # Run with Ginkgo runner
go test ./pkg/mypackage       # Test specific package
ginkgo -focus="TestName" ./... # Run focused test
```

### Integration Test
```bash
# E2E tests run in Tekton CI
# See test/e2e/ for end-to-end tests
```

### Dev Server
```bash
make run                      # Run operator against ~/.kube/config cluster
make run-verbose              # Run with verbose logging (zap-log-level=5)
```

### Clean
```bash
make clean                    # Clean build artifacts
```

### Code Generation
```bash
boilerplate/_lib/container-make generate  # Regenerate mocks, deepcopy, OpenAPI
# Run this after modifying:
# - API types (api/v1alpha1/*.go)
# - Interfaces requiring mocks
# - go:generate directives
```

## Agent Rules

### Agents MUST

**Prefer minimal diffs:**
- Make focused, incremental changes
- Avoid reformatting unrelated code
- Don't introduce abstractions unless necessary
- Three similar lines is better than premature abstraction

**Preserve coding style:**
- Follow existing patterns in the file/package
- Use Ginkgo BDD style for tests (`Describe`, `Context`, `It`)
- Use GoMock for mocking interfaces
- Follow standard Go formatting (gofmt)

**Reuse existing abstractions:**
- Check `pkg/util/` before creating new utilities
- Use existing handler patterns in `pkg/ocmagenthandler/`
- Follow controller patterns in `controllers/`
- Don't reinvent common operations

**Run targeted validation first:**
```bash
# After code changes, run in this order:
go build ./...                        # Fast compile check (5s)
go test ./pkg/changed-package         # Affected tests (10s)
prek run                              # Staged files only (15s)
make go-test                          # Full suite before commit (60s)
```

**Avoid broad refactors unless requested:**
- Fix the specific issue, not surrounding code
- Don't cleanup unrelated files
- Don't rename unless it fixes a bug
- Don't optimize unless there's a performance issue

**Avoid editing generated files:**
- Never edit `zz_generated.*.go` files
- Never edit `go.sum` without changing `go.mod`
- Never edit `deploy/crds/*.yaml` (regenerate with `make manifests`)
- Never edit mocks directly (regenerate with `make generate`)

**Avoid changing lockfiles unnecessarily:**
- Only modify `go.sum` when adding/updating dependencies
- Run `go mod tidy` after `go.mod` changes
- Verify with `git diff go.mod go.sum` before committing

**Avoid speculative changes:**
- Don't add features "for future use"
- Don't add error handling for impossible conditions
- Don't add validation for internal functions
- Only validate at system boundaries (user input, external APIs)

**Respect layering boundaries:**
- Controllers call handlers, not the reverse
- Handlers manage Kubernetes resources
- API types are pure data structures (no business logic)
- Utils are stateless helpers

**Respect existing architecture:**
- Don't introduce new frameworks or patterns
- Use controller-runtime conventions
- Follow Operator SDK patterns
- Maintain separation between controllers and handlers

### Validation Strategy

**Always run before committing:**
1. **Formatting**: `go fmt ./...` (automatic in prek)
2. **Linting**: `make go-check` or `prek run golangci-lint`
3. **Typecheck**: `go build ./...` (automatic in prek)
4. **Relevant tests**: `go test ./pkg/changed-package`
5. **Security checks**: `prek run gitleaks` (automatic)

**Prek hooks (mandatory):**
- File hygiene (trailing whitespace, EOF)
- YAML/JSON/TOML syntax validation
- Security scanning (rh-pre-commit, gitleaks)
- golangci-lint (mirrors CI exactly)
- Go build check
- Go mod tidy check
- RBAC wildcard check

**Stop hook validation:**
- Claude Code stop hook runs `prek run --all-files` automatically
- Blocks Claude from stopping if issues found
- Provides immediate feedback for fixes
- Configured in `.claude/settings.json`

**CI parity:**
- Prek hooks mirror Tekton CI checks
- Use `boilerplate/_lib/container-make` for exact CI environment
- All prek checks also run in CI
- CI remains authoritative gate

### Safe Autonomous Workflow

**Read before editing:**
- Always read files before modifying them
- Understand context and existing patterns
- Check for related code in same package

**Search for existing patterns first:**
```bash
# Find similar code
grep -r "pattern" --include="*.go" .
# Find interface definitions
grep -r "type.*interface" --include="*.go" .
# Find test patterns
grep -r "Describe\|Context\|It" --include="*_test.go" .
```

**Reuse utilities:**
- Check `pkg/util/` before creating helpers
- Use existing test mocks in `pkg/util/test/`
- Follow existing error handling patterns

**Make incremental commits:**
- One logical change per commit
- Commit message format: `<type>: <description>`
- Types: `feat`, `fix`, `test`, `refactor`, `docs`, `chore`
- Run `prek run` before each commit

**Explain risky changes:**
- Authentication/authorization modifications
- RBAC changes (especially wildcards)
- Network policy updates
- CI/CD pipeline changes
- Dockerfile modifications
- Secret handling changes

**Prefer deterministic tooling:**
- Use fixed versions in `go.mod`
- Use pinned prek hook versions (see `prek.toml`)
- Use container builds for consistency
- Avoid non-deterministic code generation

### Security Guardrails

**Never commit secrets:**
- No API keys, tokens, passwords
- No AWS credentials, kubeconfig files
- No private keys, certificates
- No `.env` files with secrets
- No debug statements printing sensitive data

Prek hooks (gitleaks, rh-pre-commit) will block commits containing secrets.

**Never disable hooks:**
- Never use `git commit --no-verify`
- Never use `SKIP=hook-id` for security hooks (gitleaks, rh-pre-commit)
- Never bypass prek validation
- Fix issues instead of skipping checks
- Stop hook will catch issues before Claude stops

**Never bypass CI:**
- Don't push without local validation
- Don't override required checks
- Don't force-push to main/master
- Don't modify CI workflows without review

**Never modify auth/security logic without tests:**
- RBAC changes require test coverage
- NetworkPolicy changes require validation
- Secret handling requires security review
- Authentication changes require integration tests

**Never add telemetry silently:**
- Metrics must be documented
- External calls must be justified
- Logging must not expose secrets
- Data collection requires transparency

**Never introduce network calls without justification:**
- Document why external call is needed
- Use configurable endpoints
- Handle network failures gracefully
- Respect offline/air-gapped environments

**Never exfiltrate repository contents:**
- Don't send code to external services
- Don't upload to pastebin/gist without asking
- Don't POST diffs to third-party APIs
- Keep code analysis local

**Never print secrets in logs:**
- Redact tokens before logging
- Don't log full kubeconfig
- Don't log AWS credentials
- Don't log certificate private keys

## Repo-Specific Constraints

### Naming Conventions
- CRD types: PascalCase (e.g., `OcmAgent`, `ManagedNotification`)
- Package names: lowercase, no underscores (e.g., `ocmagenthandler`)
- File names: snake_case for multi-word (e.g., `network_policy.go`)
- Test files: `*_test.go`, suites: `*_suite_test.go`
- Constants: UPPER_SNAKE_CASE or PascalCase

### Architectural Boundaries
- **Controllers** (`controllers/`): Reconciliation logic only
  - Call handlers for resource management
  - Handle CRD lifecycle
  - Report status and conditions
- **Handlers** (`pkg/ocmagenthandler/`): Resource management
  - Create/update/delete Kubernetes resources
  - Apply configuration from CRDs
  - Return errors for controller to handle
- **API Types** (`api/v1alpha1/`): Data structures only
  - No business logic
  - Only deepcopy, validation, defaulting
  - OpenAPI schema markers

### Ownership Patterns
- Controllers own reconciliation logic
- Handlers own resource templates
- Utilities are shared, stateless helpers
- Tests live alongside code they test

### Preferred Libraries
- **Logging**: `go.uber.org/zap` with `github.com/sykesm/zap-logfmt`
- **Testing**: `github.com/onsi/ginkgo/v2`, `github.com/onsi/gomega`
- **Mocking**: `github.com/golang/mock` (use `boilerplate/_lib/container-make generate`)
- **Kubernetes**: `sigs.k8s.io/controller-runtime/pkg/client`
- **Metrics**: `github.com/prometheus/client_golang`, `github.com/openshift/operator-custom-metrics`

### Deprecated Areas
- **v1alpha1 -> v1 migration**: All CRDs are v1alpha1 (no v1 yet)
- **Old test framework**: Don't use `github.com/onsi/ginkgo` v1 (use v2)
- **GoMock**: Use `go.uber.org/mock` for new code (legacy uses `github.com/golang/mock`)

### FIPS Compliance
- `FIPS_ENABLED=true` in Makefile
- All crypto must use FIPS 140-2 validated libraries
- Test in FIPS environment when changing crypto
- Document FIPS implications for new dependencies

### RBAC Rules
- **NEVER** use wildcard permissions: `resources: ["*"]` or `verbs: ["*"]`
- Always specify exact resource types and verbs
- pre-commit hook `rbac-wildcard-check` enforces this
- See `deploy/*.yaml` for approved RBAC patterns

### Boilerplate Integration
- Standard Makefiles in `boilerplate/openshift/golang-osd-operator/`
- Don't modify boilerplate files directly
- Extend via `Makefile` in repo root
- Update boilerplate: `make boilerplate-update`
- Container builds: `boilerplate/_lib/container-make`

### Testing Requirements
- Unit tests required for all new functions
- Use Ginkgo BDD style consistently
- Mock Kubernetes client for unit tests
- No real cluster calls in unit tests
- E2E tests in `test/e2e/` for integration

## Code Generation Dependencies

After modifying API types or interfaces, regenerate code:

```bash
boilerplate/_lib/container-make generate
```

**What this generates:**
- **Deepcopy methods**: `zz_generated.deepcopy.go` (for Kubernetes types)
- **OpenAPI schemas**: For CRD validation
- **Mock interfaces**: For testing (`pkg/util/test/generated/`)

**When to regenerate:**
- After changing `api/v1alpha1/*.go` types
- After changing interfaces with `//go:generate mockgen` directives
- After modifying struct tags (json, yaml, kubebuilder markers)
- If tests fail due to missing/outdated mocks

**Why use container-make:**
- Ensures same tool versions as CI (mockgen, controller-gen)
- Prevents generated code drift between local and CI
- Isolates from local Go environment

## Claude Code Integration

### Settings and Hooks

This repository includes Claude Code configuration in `.claude/settings.json`:

**Pre-configured permissions:**
- ✅ Allow: Common read-only commands (make, go test, git status, grep, find)
- ⚠️ Ask: Destructive operations (git commit, git push, docker build)
- ❌ Deny: Dangerous operations (--no-verify, force push to main, rm -rf /)

**Quality hooks:**
- **stop-prek-validation.sh**: Runs `prek run --all-files` on every turn, blocks Claude from stopping if issues found

**Validation framework:**
- **prek**: Git hook manager configured in `prek.toml`
- Runs automatically on git commit and Claude Code stop
- Includes: file hygiene, security scanning (rh-pre-commit, gitleaks), linting, build checks

See [.claude/hooks/README.md](./.claude/hooks/README.md) for hook documentation.

### Specialized Agents

This repository provides specialized agents for common workflows:
- **lint-agent**: Code quality and formatting
- **test-agent**: Testing and test quality assurance
- **security-agent**: Security scanning and policy enforcement
- **docs-agent**: Documentation maintenance
- **ci-agent**: CI/CD validation

See [.claude/agents/README.md](./.claude/agents/README.md) for agent documentation.

### Reusable Skills

**prow-ci**: Access and analyze OpenShift Prow CI results
- View job status and logs
- Debug CI failures
- Access artifacts and test results

See [.claude/skills/README.md](./.claude/skills/README.md) for skill documentation.

### Security Enforcement

**Layered defense:**
1. **Stop hook** (.claude/hooks/stop-prek-validation.sh): Validate on every Claude Code turn
2. **Prek hooks** (prek.toml): Validate before commit (rh-pre-commit, gitleaks, golangci-lint, RBAC checks)
3. **CI pipelines** (.tekton/): Comprehensive validation
4. **Code review**: Human oversight

**Key security features:**
- **rh-pre-commit**: Red Hat InfoSec security scanning
- **gitleaks**: Secret detection (configured via `.gitleaks.toml`)
- **RBAC wildcard check**: Prevents wildcard permissions
- **Stop hook**: Immediate feedback loop for quality issues
