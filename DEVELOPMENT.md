# OCM Agent Operator Development Guide

This guide provides comprehensive instructions for setting up a local development environment and working with the OCM Agent Operator codebase.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Building the Operator](#building-the-operator)
- [Running Locally](#running-locally)
- [Development Workflow](#development-workflow)
- [Code Generation](#code-generation)
- [Container-based Development](#container-based-development)
- [Debugging](#debugging)
- [Common Tasks](#common-tasks)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools

1. **Go** (version 1.24.0 or later)
   ```bash
   go version
   # Should output: go version go1.24.0 or later
   ```

2. **Operator SDK** (version 1.21.0)
   ```bash
   operator-sdk version
   # Expected: operator-sdk version: "v1.21.0"
   ```
   
   Installation:
   ```bash
   # macOS
   brew install operator-sdk

   # Linux
   curl -LO https://github.com/operator-framework/operator-sdk/releases/download/v1.21.0/operator-sdk_linux_amd64
   chmod +x operator-sdk_linux_amd64
   sudo mv operator-sdk_linux_amd64 /usr/local/bin/operator-sdk
   ```

3. **kubectl** or **oc** CLI
   ```bash
   kubectl version
   # or
   oc version
   ```

4. **Git**
   ```bash
   git --version
   ```

5. **Make**
   ```bash
   make --version
   ```

### Optional but Recommended

- **Docker** or **Podman** - For container-based builds
- **Kind** or **Minikube** - For local Kubernetes clusters
- **VSCode** or **GoLand** - For development IDE

## Environment Setup

### 1. Clone the Repository

```bash
# Fork the repository on GitHub first, then:
git clone https://github.com/YOUR_USERNAME/ocm-agent-operator.git
cd ocm-agent-operator

# Add upstream remote
git remote add upstream https://github.com/openshift/ocm-agent-operator.git
```

### 2. Install Development Tools

The project uses specific tool versions defined in `tools.go`:

```bash
make tools
```

This installs:
- **ginkgo** - Test runner
- **mockgen** - Mock generator for testing
- **controller-gen** - Code generator for Kubernetes types
- **openapi-gen** - OpenAPI schema generator

Tools are installed to `$GOPATH/bin` (default: `~/go/bin`), ensure this is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### 3. Verify Installation

```bash
# Check installed tools
which ginkgo
which mockgen

# Run a quick build to verify setup
make go-build
```

### 4. Set Up Go Proxy (if needed)

If you encounter checksum mismatch errors during `make tools`:

```bash
export GOPROXY="https://proxy.golang.org"
make tools
```

## Building the Operator

### Standard Build

```bash
# Build the operator binary
make go-build

# Output: ./build/_output/bin/ocm-agent-operator
```

### Clean Build

```bash
# Clean previous builds
rm -rf ./build/_output

# Rebuild
make go-build
```

### Build with FIPS Support

FIPS mode is enabled by default via the Makefile:

```bash
# FIPS is enabled automatically
make go-build
```

### Cross-platform Builds

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 make go-build

# Build for macOS
GOOS=darwin GOARCH=amd64 make go-build
```

## Running Locally

### Prerequisites for Local Runs

1. **Access to an OpenShift or Kubernetes cluster**:
   ```bash
   # Configure kubectl/oc to point to your cluster
   oc login https://api.your-cluster.example.com:6443
   
   # Or use kubeconfig
   export KUBECONFIG=/path/to/kubeconfig
   ```

2. **Install CRDs** (if not already installed):
   ```bash
   oc create -f deploy/crds/ocmagent.managed.openshift.io_ocmagents.yaml
   oc create -f deploy/crds/ocmagent.managed.openshift.io_managednotifications.yaml
   oc create -f deploy/crds/ocmagent.managed.openshift.io_managedfleetnotifications.yaml
   ```

3. **Create the operator namespace**:
   ```bash
   oc create namespace openshift-ocm-agent-operator
   ```

### Run the Operator

```bash
# Run with standard logging
make run

# Run with verbose logging
make run-verbose
```

The operator will:
- Connect to the cluster specified in `~/.kube/config`
- Watch for `OcmAgent` resources in the `openshift-ocm-agent-operator` namespace
- Reconcile resources based on custom resource definitions

### Environment Variables

Key environment variables set automatically by `make run`:

```bash
OPERATOR_NAMESPACE=openshift-ocm-agent-operator
```

For custom configurations:

```bash
# Custom namespace
OPERATOR_NAMESPACE=my-namespace make run

# Verbose logging
make run-verbose  # Sets zap-log-level=5
```

## Development Workflow

### Typical Development Cycle

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make code changes**

3. **Run code generation** (if you modified API types):
   ```bash
   make generate
   make manifests
   ```

4. **Run tests**:
   ```bash
   make go-test
   ```

5. **Run linting**:
   ```bash
   make lint
   ```

6. **Test locally** (optional but recommended):
   ```bash
   make run
   ```

7. **Commit and push**:
   ```bash
   git add .
   git commit -s -m "feat: add my feature"
   git push origin feature/my-feature
   ```

8. **Create a pull request** on GitHub

### Working with API Types

When modifying CRD definitions in `api/v1alpha1/`:

```bash
# 1. Edit the types (e.g., api/v1alpha1/ocmagent_types.go)

# 2. Generate deepcopy methods and update manifests
make generate
make manifests

# 3. Verify changes
git diff config/crd/bases/

# 4. Run tests
make go-test

# 5. Update documentation if needed
```

### Working with Controllers

When modifying controllers in `controllers/`:

```bash
# 1. Edit controller logic

# 2. Update or add unit tests

# 3. Run tests
make go-test

# 4. Test with a real cluster
make run

# 5. Verify behavior in cluster
oc get ocmagent -n openshift-ocm-agent-operator
oc describe ocmagent <name> -n openshift-ocm-agent-operator
```

## Code Generation

The operator uses several code generators:

### Generate All

```bash
make generate
```

This runs:
- **controller-gen** - Generates deepcopy methods for API types
- **mockgen** - Generates mocks for interfaces

### Generate CRD Manifests

```bash
make manifests
```

Outputs to `config/crd/bases/`

### Generate Mocks

Mocks are generated from interfaces with `//go:generate` directives:

```bash
# Generate all mocks (use container for consistency)
boilerplate/_lib/container-make generate
```

**Important**: Always generate mocks in the boilerplate container to match CI environment.

### Verifying Generated Code

```bash
# Run validation (same as CI)
boilerplate/_lib/container-make validate
```

This ensures all generated code is up-to-date.

## Container-based Development

For consistency with CI, use container-based builds:

### Available Container Commands

```bash
# Run all checks (lint + test + build)
boilerplate/_lib/container-make

# Run specific targets
boilerplate/_lib/container-make go-test
boilerplate/_lib/container-make lint
boilerplate/_lib/container-make generate
boilerplate/_lib/container-make validate
```

### Why Use Container-based Builds?

- **Consistency**: Matches exact CI environment
- **Reproducibility**: Same Go version, same tool versions
- **Isolation**: Doesn't affect local environment
- **CI Parity**: Catches issues before pushing

### Updating Boilerplate

The project uses [openshift/boilerplate](https://github.com/openshift/boilerplate) for build infrastructure:

```bash
make boilerplate-update
```

## Debugging

### Debugging the Operator Locally

1. **Use verbose logging**:
   ```bash
   make run-verbose
   ```

2. **Add debug logs** in your code:
   ```go
   import "github.com/go-logr/logr"

   func (r *OcmAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
       log := logr.FromContext(ctx)
       log.V(1).Info("Debug message", "key", "value")
   }
   ```

3. **Use Delve** for breakpoint debugging:
   ```bash
   dlv debug ./main.go -- --zap-log-level=5
   ```

### Debugging in VS Code

Create `.vscode/launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Operator",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/main.go",
            "env": {
                "OPERATOR_NAMESPACE": "openshift-ocm-agent-operator"
            },
            "args": ["--zap-log-level=5"]
        }
    ]
}
```

### Debugging Tests

```bash
# Run tests with verbose output
ginkgo -v -r

# Run specific test
ginkgo -v --focus="OcmAgent controller" controllers/

# Debug a test in VS Code
# Add breakpoints and use "Debug Test" CodeLens
```

### Viewing Operator Logs in Cluster

```bash
# Get operator pod
oc get pods -n openshift-ocm-agent-operator

# View logs
oc logs -f <operator-pod-name> -n openshift-ocm-agent-operator

# View previous logs (if pod crashed)
oc logs <operator-pod-name> -n openshift-ocm-agent-operator --previous
```

## Common Tasks

### Adding a New Controller

```bash
# Use operator-sdk to scaffold
operator-sdk create api \
  --group ocmagent \
  --version v1alpha1 \
  --kind MyResource \
  --resource \
  --controller

# Generate code
make generate manifests

# Implement reconciliation logic in controllers/

# Add tests
```

### Adding a New API Field

```bash
# 1. Edit api/v1alpha1/*_types.go
# Add field to spec or status

# 2. Regenerate code
make generate manifests

# 3. Update controller to handle new field

# 4. Add tests

# 5. Update documentation
```

### Updating Dependencies

```bash
# Update a specific dependency
go get github.com/example/package@v1.2.3

# Tidy up
go mod tidy

# Verify
go mod verify

# Run tests
make go-test
```

### Running Specific Tests

```bash
# Run tests for a specific package
go test ./controllers/ocmagent/...

# Run a specific test
go test ./controllers/ocmagent/... -run TestOcmAgentController

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Troubleshooting

### Common Issues

**Issue**: `make tools` fails with checksum mismatch

**Solution**:
```bash
export GOPROXY="https://proxy.golang.org"
go clean -modcache
make tools
```

---

**Issue**: Mock generation creates different output than CI

**Solution**: Always use container-based generation:
```bash
boilerplate/_lib/container-make generate
```

---

**Issue**: Operator can't connect to cluster

**Solution**:
```bash
# Verify kubeconfig
oc whoami
oc cluster-info

# Check KUBECONFIG variable
echo $KUBECONFIG

# Try explicit kubeconfig
KUBECONFIG=/path/to/kubeconfig make run
```

---

**Issue**: CRDs not found when running operator

**Solution**:
```bash
# Install CRDs
oc create -f deploy/crds/
```

---

**Issue**: Tests fail with "no such file or directory" for mocks

**Solution**:
```bash
# Regenerate mocks
boilerplate/_lib/container-make generate
```

---

**Issue**: `make lint` fails

**Solution**:
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Or use container-based linting
boilerplate/_lib/container-make lint
```

### Getting Help

- Check [CONTRIBUTING.md](./CONTRIBUTING.md) for contribution guidelines
- Review [TESTING.md](./TESTING.md) for testing best practices
- See [docs/](./docs/) for architecture and design documentation
- Ask in project discussions or relevant Slack channels
- File an issue on GitHub for bugs or questions

## Additional Resources

- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [Controller Runtime Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)
- [OpenShift Boilerplate](https://github.com/openshift/boilerplate)
