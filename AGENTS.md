# OCM Agent Operator - AI Agent Quick Reference

This file provides quick reference documentation for AI agents working with the OCM Agent Operator codebase.

## Project Overview

The OCM Agent Operator is a Kubernetes operator that manages the [OCM Agent](https://github.com/openshift/ocm-agent) service on managed OpenShift clusters. Built using the Operator SDK framework, it follows standard Kubernetes operator patterns.

**Purpose**: Automates deployment, configuration, and lifecycle management of the OCM Agent, enabling OpenShift Cluster Manager (OCM) to communicate with and manage clusters remotely.

**Key Capabilities**:
- Deploys and manages OCM Agent instances via custom resources
- Handles OCM authentication tokens and cluster credentials
- Configures AlertManager integration for cluster monitoring
- Manages fleet-wide notifications across multiple clusters
- Supports cluster proxy configuration for restricted network environments
- Provides Prometheus metrics for monitoring operator health

## Repository Structure

```
ocm-agent-operator/
├── api/v1alpha1/              # CRD definitions (OcmAgent, ManagedNotification, ManagedFleetNotification)
├── controllers/               # Reconciliation logic for custom resources
│   ├── ocmagent/             # Main OcmAgent controller
│   └── fleetnotification/    # Fleet notification controller
├── pkg/                      # Supporting packages
│   ├── ocmagenthandler/      # Business logic for managing OCM Agent resources
│   ├── localmetrics/         # Prometheus metrics
│   └── util/                 # Utilities and test helpers
├── config/                   # Kubernetes manifests and CRD definitions
├── deploy/                   # Deployment manifests
├── docs/                     # Documentation
├── test/                     # E2E tests
└── boilerplate/              # Build infrastructure from openshift/boilerplate
```

## Quick Commands

```bash
# Setup
make tools                      # Install dev tools (ginkgo, mockgen, etc.)

# Build and Test
make go-build                   # Build operator binary
make go-test                    # Run unit tests
make lint                       # Run linting
make                           # Run checks + test + build (same as CI)

# Run Locally
make run                        # Run against ~/.kube/config cluster
make run-verbose                # Run with verbose logging (zap-log-level=5)

# Container-based (matches CI exactly)
boilerplate/_lib/container-make           # Run all checks
boilerplate/_lib/container-make generate  # Generate mocks (ALWAYS use this)
boilerplate/_lib/container-make test      # Run tests in container
boilerplate/_lib/container-make lint      # Run linting in container
```

**Prerequisites**: Go 1.24.0+, Operator SDK v1.21.0, kubectl/oc. See [DEVELOPMENT.md](./DEVELOPMENT.md) for detailed setup.

**Important**: Always use `boilerplate/_lib/container-make generate` for mock generation to match CI environment.

## Architecture

### Custom Resource Definitions (CRDs)

API group/version: `ocmagent.managed.openshift.io/v1alpha1`

#### OcmAgent
Main CR defining OCM Agent deployment.

**Key Spec Fields**: `ocmBaseUrl`, `services`, `tokenSecret`, `replicas`, `fleetMode`

**Status Fields**: Deployment state, conditions, observed generation

#### ManagedNotification
Notification templates for Service Log notifications on individual clusters.

#### ManagedFleetNotification
Fleet-wide notifications across multiple clusters.

### Controllers

#### OCMAgent Controller
**Location**: [controllers/ocmagent/ocmagent_controller.go](controllers/ocmagent/ocmagent_controller.go)

**Managed Resources**:
- ServiceAccount, Role, RoleBinding (named `ocm-agent`)
- Deployment (runs [ocm-agent](https://quay.io/openshift/ocm-agent) image)
- ConfigMap, Secret (from CR spec)
- Service (port 8081), NetworkPolicy, PodDisruptionBudget
- ServiceMonitor (metrics collection)

**Watched Resources**:
- Deployed resources in operator namespace
- Cluster pull secret (`openshift-config/pull-secret`) - OCM auth token
- Cluster proxy config (`proxy/cluster`) - HTTP_PROXY, HTTPS_PROXY, NO_PROXY

**AlertManager Integration**: Creates ConfigMap `ocm-agent` in `openshift-monitoring` namespace with `serviceURL` for [configure-alertmanager-operator](https://github.com/openshift/configure-alertmanager-operator).

#### Fleet Notification Controller
Reconciles `ManagedFleetNotification` resources to distribute notifications across fleets.

### Core Packages

#### pkg/ocmagenthandler/
Business logic for managing OCM Agent resources. Abstracts resource creation/update from controller reconciliation loop.

**Why**: Separation of concerns - controllers handle reconciliation, handlers manage resource specifications.

#### pkg/localmetrics/
Prometheus metrics for operator health:

- **`ocm_agent_operator_pull_secret_invalid`** (Gauge) - Set to `1` if pull secret retrieval/parsing fails
- **`ocm_agent_operator_ocm_agent_resource_absent`** (Gauge) - Set to `1` if OcmAgent CR not found

Exposed on `ocm-agent-operator-metrics` Service, port 8686.

#### pkg/util/
Test utilities, including mock generators for controller-runtime clients.

## Testing

**Framework**: Ginkgo (BDD), Gomega (matchers), GoMock (mocking)

```bash
# Unit Tests
make go-test                    # All tests
ginkgo -r                       # With Ginkgo runner
go test ./pkg/specific/...      # Specific package

# E2E Tests (requires cluster access)
make e2e-binary-build
# Deploy operator, then:
KUBECONFIG=/tmp/kubeconfig ginkgo --tags=osde2e -v test/e2e

# Mock Generation (CRITICAL: Use container)
boilerplate/_lib/container-make generate
```

**Important**: Always generate mocks in boilerplate container to match CI mockgen version. See [TESTING.md](./TESTING.md) for comprehensive testing guide.

## Code Generation

```bash
make generate                   # Deepcopy methods, mocks
make manifests                  # CRD YAML manifests
```

**When**: After modifying API types in `api/v1alpha1/`. Always commit generated files with changes.

**Why**: CRD types require deepcopy methods for Kubernetes API machinery. OpenAPI schemas enable admission validation.

## Development Workflow

1. Create feature branch: `git checkout -b feature/my-feature`
2. Make code changes (controllers, handlers, API types)
3. Run code generation (if API changed): `make generate && make manifests`
4. Update tests
5. Run tests: `make go-test`
6. Run linting: `make lint`
7. Test on cluster (optional): `make run`
8. Commit and push: `git commit -s -m "feat: description"`

See [DEVELOPMENT.md](./DEVELOPMENT.md) for detailed workflows, debugging, and troubleshooting.

## Common Operations

### Adding a New Controller
```bash
operator-sdk create api --group ocmagent --version v1alpha1 --kind MyResource --resource --controller
make generate manifests
# Implement reconciliation logic, add tests
```

### Modifying CRD Types
```bash
# 1. Edit api/v1alpha1/*_types.go
# 2. Regenerate
make generate manifests
# 3. Update controller logic
# 4. Add tests
```

### Updating Dependencies
```bash
go get github.com/example/package@v1.2.3
go mod tidy
make go-test
```

## Key Dependencies

- `sigs.k8s.io/controller-runtime` v0.33.x - Operator framework
- `k8s.io/api`, `k8s.io/client-go` v0.33.2 - Kubernetes APIs
- `github.com/openshift/api` - OpenShift API types (proxy, config)
- `github.com/prometheus-operator/prometheus-operator` v0.67.1 - ServiceMonitor
- `github.com/onsi/ginkgo/v2` v2.22.0, `github.com/onsi/gomega` v1.36.1 - Testing
- `github.com/golang/mock` v1.6.0 - Mock generation
- `go.uber.org/zap` v1.27.0 - Logging

## Troubleshooting

**Mock generation fails or differs from CI**:
```bash
boilerplate/_lib/container-make generate
```

**Tests fail locally but pass in CI** (or vice versa):
```bash
boilerplate/_lib/container-make test
```

**Operator can't reconcile**:
```bash
oc logs -n openshift-ocm-agent-operator deployment/ocm-agent-operator
oc get crd ocmagents.ocmagent.managed.openshift.io
oc describe role ocm-agent -n openshift-ocm-agent-operator
```

**Metrics not appearing**:
```bash
oc get servicemonitor ocm-agent-metrics -n openshift-ocm-agent-operator
oc get endpoints ocm-agent-operator-metrics -n openshift-ocm-agent-operator
```

## Documentation Index

- **[DEVELOPMENT.md](./DEVELOPMENT.md)** - Complete development environment setup, building, running, debugging
- **[TESTING.md](./TESTING.md)** - Testing framework, writing tests, E2E tests, mock generation
- **[CONTRIBUTING.md](./CONTRIBUTING.md)** - Contribution guidelines, PR process, code standards
- **[docs/design.md](./docs/design.md)** - Architecture and interaction between operator and CRDs
- **[docs/how-to-test.md](./docs/how-to-test.md)** - Manual testing procedures
- **[docs/metrics.md](./docs/metrics.md)** - Available Prometheus metrics

## Ownership & Related Projects

**Maintained by**: SREP team (see `OWNERS` and `OWNERS_ALIASES`)

**Related**:
- [ocm-agent](https://github.com/openshift/ocm-agent) - Agent deployed by this operator
- [configure-alertmanager-operator](https://github.com/openshift/configure-alertmanager-operator) - Consumes OCM Agent service URL
- [openshift/boilerplate](https://github.com/openshift/boilerplate) - Build infrastructure and CI tooling
