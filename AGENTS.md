# OCM Agent Operator - AI Agent Documentation

This file provides comprehensive documentation for AI agents working with the OCM Agent Operator codebase.

## Project Overview

The OCM Agent Operator is a Kubernetes operator that manages the [OCM Agent](https://github.com/openshift/ocm-agent) service on managed OpenShift clusters. It's built using the Operator SDK framework and follows standard Kubernetes operator patterns.

**Purpose**: Automates the deployment, configuration, and lifecycle management of the OCM Agent, which enables OpenShift Cluster Manager (OCM) to communicate with and manage clusters remotely.

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

## Build & Development Commands

All build infrastructure comes from [openshift/boilerplate](https://github.com/openshift/boilerplate). The Makefile is minimal and includes boilerplate targets.

### Prerequisites

- **Go**: 1.24.0 or later (module-aware mode)
- **Operator SDK**: v1.21.0
- **kubectl/oc**: For cluster interaction
- **ginkgo**: For running tests (installed via `make tools`)
- **mockgen**: For generating test mocks (installed via `make tools`)

**Why**: Operator SDK version compatibility with controller-runtime and Kubernetes APIs is critical for code generation and operator functionality.

### Setup

```bash
make tools                      # Install required development tools (ginkgo, mockgen, etc.)
                               # Reads from tools.go and installs to $GOPATH/bin
```

**How to apply**: Run this after cloning the repo or when tool dependencies change in go.mod. If checksum mismatch occurs, set `GOPROXY="https://proxy.golang.org"`.

### Build and Test

```bash
make go-build                   # Build the operator binary
make go-test                    # Run unit tests (also: go test ./...)
make lint                       # Run golangci-lint and static analysis
make                           # Default target: go-check + go-test + go-build
```

**Why**: The default `make` target runs the same checks as CI, ensuring local changes pass before pushing.

### Run Locally

```bash
make run                        # Run operator against ~/.kube/config cluster
make run-verbose                # Run with verbose logging (zap-log-level=5)
```

**How to apply**: Requires OPERATOR_NAMESPACE="openshift-ocm-agent-operator" environment variable. The operator will reconcile OcmAgent resources in the configured cluster.

### Container-based Development

```bash
boilerplate/_lib/container-make           # Run make inside boilerplate container
boilerplate/_lib/container-make generate  # Generate mocks in container environment
boilerplate/_lib/container-make test      # Run tests in container (matches CI)
boilerplate/_lib/container-make lint      # Run linting in container (matches CI)
```

**Why**: Container-based builds match the CI environment exactly, avoiding "works locally but fails in CI" issues. Always use `container-make generate` for mock generation to ensure consistent mockgen versions.

## Architecture

### Custom Resource Definitions (CRDs)

API group/version: `ocmagent.managed.openshift.io/v1alpha1`

#### OcmAgent

Defines the deployment of the OCM Agent on a cluster.

```bash
oc get ocmagent -n openshift-ocm-agent-operator
```

**Spec Fields**:
- `ocmBaseUrl`: OCM API base URL
- `services`: OCM service endpoints configuration
- `tokenSecret`: Reference to secret containing OCM access token
- `replicas`: Number of OCM Agent replicas
- `fleetMode`: Enable/disable fleet-wide notification handling

**Status Fields**: Tracks deployment state, conditions, and observed generation

#### ManagedNotification

Defines notification templates for Service Log notifications on individual clusters.

```bash
oc get managednotification -n openshift-ocm-agent-operator
```

**Purpose**: Allows OCM to send structured notifications to cluster administrators via Service Logs.

#### ManagedFleetNotification

Manages fleet-wide notifications across multiple clusters.

**Why**: Enables broadcasting critical updates or maintenance notifications to entire fleets of clusters simultaneously.

### Controllers

#### OCMAgent Controller

**Location**: `controllers/ocmagent/ocmagent_controller.go`

**Responsibility**: Ensures the deployment or removal of an OCM Agent based on the presence of an `OcmAgent` custom resource.

**Managed Resources**:
- `ServiceAccount` (named `ocm-agent`)
- `Role` and `RoleBinding` (both named `ocm-agent`) - API permissions for OCM Agent
- `Deployment` (named `ocm-agent`) - Runs the [ocm-agent](https://quay.io/openshift/ocm-agent) image
- `ConfigMap` (name from CR) - Agent configuration
- `Secret` (name from CR) - OCM access token
- `Service` (named `ocm-agent`) - Exposes OCM Agent API on port 8081
- `NetworkPolicy` - Restricts ingress to specific cluster clients
- `PodDisruptionBudget` - Ensures availability during disruptions
- `ServiceMonitor` (named `ocm-agent-metrics`) - Prometheus metrics collection

**Watched Resources**:
- Changes to the above resources in the deployed namespace
- Changes to cluster pull secret (`openshift-config/pull-secret`) containing OCM auth token
- Cluster proxy configuration (`proxy/cluster`) for HTTP_PROXY, HTTPS_PROXY, NO_PROXY injection

**AlertManager Integration**: Creates a `ConfigMap` named `ocm-agent` in the `openshift-monitoring` namespace containing:

| Key | Description | Example |
| --- | --- | --- |
| `serviceURL` | OCM Agent service URI | `http://ocm-agent.openshift-ocm-agent-operator.svc.cluster.local:8081/alertmanager-receiver` |

**Why**: The [configure-alertmanager-operator](https://github.com/openshift/configure-alertmanager-operator) uses this ConfigMap to route alerts to the OCM Agent.

**Cluster Proxy Support**: Automatically monitors `proxy/cluster` object and injects HTTP_PROXY, HTTPS_PROXY, and NO_PROXY environment variables into the OCM Agent deployment based on cluster-wide proxy settings.

#### Fleet Notification Controller

**Responsibility**: Reconciles `ManagedFleetNotification` resources to distribute notifications across fleets.

**How to apply**: When creating fleet-wide notifications, ensure the fleet selector matches the intended cluster scope.

### Core Packages

#### pkg/ocmagenthandler/

Business logic for managing OCM Agent resources. Abstracts resource creation/update logic from the controller reconciliation loop.

**Key Interfaces**:
- `OcmAgentHandler`: Manages Deployment, ConfigMap, Secret, NetworkPolicy, PDB, ServiceMonitor

**Why**: Separation of concerns - controllers handle reconciliation logic, handlers manage resource templates and specifications.

#### pkg/localmetrics/

Prometheus metrics for monitoring operator health.

**Available Metrics**:

**`ocm_agent_operator_pull_secret_invalid`** (Gauge)
- Description: Set to `1` if the operator cannot retrieve or parse the cluster's `cloud.openshift.com` pull secret, `0` otherwise
- Labels: `ocmagent_name`
- Example: `ocm_agent_operator_pull_secret_invalid{ocmagent_name="ocmagent"} = 0`

**`ocm_agent_operator_ocm_agent_resource_absent`** (Gauge)
- Description: Set to `1` if the operator cannot find the `OcmAgent` custom resource, `0` otherwise
- Example: `ocm_agent_operator_ocm_agent_resource_absent = 1`

**How to apply**: These metrics are exposed on the `ocm-agent-operator-metrics` Service on port 8686. Query them in Prometheus or alert on non-zero values.

#### pkg/util/

Test utilities and helpers.

**Key Components**:
- `pkg/util/test/mockgenerator/`: Mock interface generators for controller-runtime clients
- Mock generation uses `//go:generate` directives above interfaces

**Why**: Mocks enable unit testing of controllers without requiring a real Kubernetes cluster.

## Testing

### Framework

- **Ginkgo**: BDD testing framework for Go
- **Gomega**: Matcher/assertion library
- **GoMock**: Interface mocking for dependency injection

**Why**: Ginkgo provides expressive BDD-style tests. Gomega's matchers make assertions readable. GoMock enables isolated unit tests.

### Test Organization

Tests are organized by package with `_suite_test.go` files bootstrapping Ginkgo suites. Each package has its own test suite.

### Unit Tests

```bash
make go-test                    # Run all unit tests
go test ./...                   # Alternative using go test directly
ginkgo -r                       # Run tests recursively with Ginkgo
```

**How to apply**: Always run tests before submitting PRs. PRs are expected to add, modify, or delete tests on a case-by-case basis.

### E2E Tests

**Location**: `test/e2e/`

**Requirements**:
- A working OCM Agent image to deploy
- Access to a test cluster (OSD/ROSA recommended)
- Kubeadmin credentials

**Running E2E Tests Locally**:

```bash
# 1. Build e2e test suite
make e2e-binary-build

# 2. Deploy your operator version to a test cluster
oc create -f deploy/crds/ocmagent.managed.openshift.io_ocmagents.yaml
find test/deploy -type f -name '*.yaml' -exec oc create -f {} \;

# 3. Install Ginkgo
go install github.com/onsi/ginkgo/ginkgo@latest

# 4. Get cluster credentials (replace cluster-id)
ocm get /api/clusters_mgmt/v1/clusters/(cluster-id)/credentials | jq -r .kubeconfig > /tmp/kubeconfig

# 5. Run e2e suite
DISABLE_JUNIT_REPORT=true KUBECONFIG=/tmp/kubeconfig ginkgo --tags=osde2e -v test/e2e
```

**Why**: E2E tests validate operator behavior in real clusters, catching integration issues that unit tests miss.

### Mock Generation

Mocks are generated from interfaces using GoMock's `mockgen` utility.

**Regenerating Mocks**:

```bash
boilerplate/_lib/container-make generate
```

**Why**: Always regenerate mocks in the boilerplate container to ensure the same mockgen version used in CI. Version mismatches can cause test failures or inconsistent mock behavior.

**How to apply**: After modifying interfaces, run this command and commit the updated mock files.

### Writing Tests

1. **Bootstrap a new test suite**:
   ```bash
   cd pkg/samplepackage
   ginkgo bootstrap              # Creates samplepackage_suite_test.go
   ginkgo generate samplepackage.go  # Creates samplepackage_test.go
   ```

2. **Add mock generation directives**: Include `//go:generate` comments above interfaces:
   ```go
   //go:generate mockgen -destination=mocks/mock_client.go -package=mocks github.com/example/pkg Client
   type Client interface {
       DoSomething() error
   }
   ```

3. **Run mock generation**: `boilerplate/_lib/container-make generate`

4. **Write tests using mocks**: Use generated mocks in test files with Ginkgo/Gomega

**Why**: Mocking external dependencies (Kubernetes clients, HTTP clients) enables fast, isolated unit tests.

## Code Generation

The operator uses code generation for:
- **Kubernetes deepcopy methods**: Required for CRD types
- **OpenAPI schemas**: CRD validation schemas
- **Mock interfaces**: Test mocks via GoMock

**Triggering Code Generation**:

```bash
make generate                   # Run all code generators
make manifests                  # Generate CRD manifests only
```

**How to apply**: Always run `make` after modifying API types (`api/v1alpha1/`) to regenerate required code. Commit generated files with your changes.

**Why**: CRD types require deepcopy methods for the Kubernetes API machinery. OpenAPI schemas enable admission validation.

## CI/CD

- **Tekton Pipelines**: `.tekton/` directory contains pipeline definitions for PR and push events
- **Boilerplate**: Provides standardized CI environment (version tracked in `boilerplate/`)
- **Codecov**: Code coverage reporting configured in `.codecov.yml` (ignores `**/mocks` and `**/zz_generated*.go`)
- **Konflux**: Konflux builds enabled via `KONFLUX_BUILDS=true` in Makefile

**How to apply**: CI runs the same targets as `boilerplate/_lib/container-make`. Use container-based builds locally to match CI behavior.

## Dependencies

### Key Dependencies

- `sigs.k8s.io/controller-runtime` v0.33.x - Operator framework
- `k8s.io/api` v0.33.2 - Kubernetes API types
- `k8s.io/client-go` v0.33.2 - Kubernetes client
- `github.com/openshift/api` - OpenShift API types (proxy, config)
- `github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring` v0.67.1 - ServiceMonitor CRDs
- `github.com/onsi/ginkgo/v2` v2.22.0 - Testing framework
- `github.com/onsi/gomega` v1.36.1 - Matcher library
- `github.com/golang/mock` v1.6.0 - Mock generation
- `go.uber.org/zap` v1.27.0 - Structured logging
- `github.com/sykesm/zap-logfmt` v0.0.4 - Logfmt encoder for zap

### Tool Dependencies

Defined in `tools.go` and installed via `make tools`:
- `github.com/onsi/ginkgo/v2/ginkgo` - Test runner
- `github.com/golang/mock/mockgen` - Mock generator

**Why**: `tools.go` ensures tool versions match `go.mod`, avoiding version skew between local and CI environments.

## Deployment

### On-Cluster Deployment (Image URL)

```bash
# 1. Apply CRDs
oc create -f deploy/crds/ocmagent.managed.openshift.io_ocmagents.yaml

# 2. Edit image in deployment manifest
# Edit test/deploy/50_ocm-agent-operator.Deployment.yaml

# 3. Apply all deployment resources
find test/deploy -type f -name '*.yaml' -exec oc create -f {} \;
```

### On-Cluster Deployment (Local Changes)

```bash
# 1. Login to cluster as cluster-admin
oc login https://api.cluster.example.com:6443

# 2. Run operator locally
make run
```

**How to apply**: The operator will connect to the cluster in `~/.kube/config` and reconcile resources in the `openshift-ocm-agent-operator` namespace.

## Development Workflow

### Typical Development Cycle

1. **Modify code** (controllers, handlers, API types)
2. **Run code generation** (if API types changed): `make generate`
3. **Update tests** as needed
4. **Run tests locally**: `make go-test`
5. **Run linting**: `make lint`
6. **Test in container** (optional, matches CI): `boilerplate/_lib/container-make`
7. **Test on cluster** (optional): `make run` against a test cluster
8. **Commit changes** and submit PR

**Why**: Following this workflow ensures code passes CI checks before pushing.

### Adding a New Controller

1. Use `operator-sdk create api` to scaffold the controller
2. Implement reconciliation logic in `controllers/`
3. Add handler logic in `pkg/` if needed
4. Write unit tests using Ginkgo/Gomega
5. Add E2E tests in `test/e2e/`
6. Run `make generate` to update CRD manifests
7. Update this documentation

### Modifying CRD Types

1. Edit types in `api/v1alpha1/`
2. Run `make generate` to regenerate deepcopy methods and OpenAPI schemas
3. Run `make manifests` to update CRD YAML in `config/`
4. Update controller logic to handle new/changed fields
5. Write tests for new functionality
6. Update documentation

**Why**: CRD changes require code generation and manifest updates. Forgetting these steps causes CI failures.

## Troubleshooting

### Common Issues

**Mock generation fails**:
- Ensure `make tools` has been run
- Use `boilerplate/_lib/container-make generate` instead of local mockgen
- Check that `//go:generate` directives have correct paths

**Tests fail locally but pass in CI** (or vice versa):
- Version mismatch between local tools and CI
- Use `boilerplate/_lib/container-make test` to match CI environment exactly

**Operator fails to reconcile resources**:
- Check operator logs: `oc logs -n openshift-ocm-agent-operator deployment/ocm-agent-operator`
- Verify CRDs are installed: `oc get crd ocmagents.ocmagent.managed.openshift.io`
- Check RBAC permissions: `oc describe role ocm-agent -n openshift-ocm-agent-operator`

**Metrics not appearing in Prometheus**:
- Verify ServiceMonitor exists: `oc get servicemonitor ocm-agent-metrics -n openshift-ocm-agent-operator`
- Check Service endpoints: `oc get endpoints ocm-agent-operator-metrics -n openshift-ocm-agent-operator`

## Reference Documentation

- [Design](./docs/design.md) - Interaction between operator and CRDs
- [Development](./docs/development.md) - Development environment setup
- [Testing](./docs/testing.md) - Test framework and best practices
- [How To Test](./docs/how-to-test.md) - Manual testing procedures
- [Metrics](./docs/metrics.md) - Available Prometheus metrics

## Ownership

Maintained by the SREP team. See `OWNERS` and `OWNERS_ALIASES` files for current maintainers and approval requirements.

## Related Projects

- [ocm-agent](https://github.com/openshift/ocm-agent) - The agent deployed by this operator
- [configure-alertmanager-operator](https://github.com/openshift/configure-alertmanager-operator) - Consumes OCM Agent service URL for alert routing
- [openshift/boilerplate](https://github.com/openshift/boilerplate) - Build infrastructure and CI tooling
