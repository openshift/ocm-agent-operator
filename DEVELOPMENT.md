# OCM Agent Operator Development

## Prerequisites

### Required Tools

- **Go** 1.24.0 or later
- **Operator SDK** v1.21.0
- **kubectl/oc** CLI
- **Git** and **Make**

Installation:
```bash
# Verify versions
go version              # go1.24.0+
operator-sdk version    # v1.21.0
kubectl version

# Install development tools
make tools              # Installs ginkgo, mockgen, etc. to $GOPATH/bin
```

**Note**: If `make tools` fails with checksum mismatch, set `GOPROXY="https://proxy.golang.org"`

### Optional
- Docker/Podman for container builds
- Kind/Minikube for local clusters

## Environment Setup

```bash
# Clone and setup
git clone https://github.com/YOUR_USERNAME/ocm-agent-operator.git
cd ocm-agent-operator
git remote add upstream https://github.com/openshift/ocm-agent-operator.git

# Install tools
make tools
export PATH=$PATH:$(go env GOPATH)/bin

# Verify
make go-build
```

## Building and Running

### Build
```bash
make go-build           # Build binary
make                    # Build + lint + test (same as CI)
```

### Run Locally
```bash
# Install CRDs first
oc create -f deploy/crds/ocmagent.managed.openshift.io_ocmagents.yaml

# Create namespace
oc create namespace openshift-ocm-agent-operator

# Run operator
make run                # Standard logging
make run-verbose        # Verbose (zap-log-level=5)
```

Environment variables:
- `OPERATOR_NAMESPACE=openshift-ocm-agent-operator` (set automatically by `make run`)
- `KUBECONFIG` to use non-default kubeconfig

## Development Workflow

Standard cycle:
```bash
# 1. Create branch
git checkout -b feature/my-feature

# 2. Make changes

# 3. If API types changed
make generate manifests

# 4. Test
make go-test
make lint

# 5. Optional: test on cluster
make run

# 6. Commit
git commit -s -m "feat: description"
git push origin feature/my-feature
```

### Modifying API Types

When editing `api/v1alpha1/*_types.go`:
```bash
# 1. Edit types
# 2. Regenerate
make generate           # Deepcopy methods
make manifests          # CRD YAML
# 3. Update controller logic
# 4. Add tests
```

### Modifying Controllers

```bash
# 1. Edit controllers/
# 2. Update/add tests
# 3. Run tests
make go-test
# 4. Test with real cluster
make run
```

## Code Generation

```bash
make generate           # Generate deepcopy, mocks
make manifests          # Generate CRD manifests

# For mocks, ALWAYS use container
boilerplate/_lib/container-make generate
```

**Why container?** Ensures same mockgen version as CI, prevents "mock out of date" failures.

## Container-based Development

Match CI environment exactly:
```bash
boilerplate/_lib/container-make           # All checks
boilerplate/_lib/container-make test      # Tests
boilerplate/_lib/container-make lint      # Linting
boilerplate/_lib/container-make generate  # Mocks
boilerplate/_lib/container-make validate  # Verify generated code
```

## Debugging

### Local Debugging
```bash
# Verbose logging
make run-verbose

# With Delve
dlv debug ./main.go -- --zap-log-level=5
```

### VS Code
Create `.vscode/launch.json`:
```json
{
    "version": "0.2.0",
    "configurations": [{
        "name": "Debug Operator",
        "type": "go",
        "request": "launch",
        "mode": "debug",
        "program": "${workspaceFolder}/main.go",
        "env": {"OPERATOR_NAMESPACE": "openshift-ocm-agent-operator"},
        "args": ["--zap-log-level=5"]
    }]
}
```

### Cluster Logs
```bash
oc get pods -n openshift-ocm-agent-operator
oc logs -f <pod-name> -n openshift-ocm-agent-operator
oc logs <pod-name> -n openshift-ocm-agent-operator --previous  # If crashed
```

## Common Tasks

### Add New Controller
```bash
operator-sdk create api \
  --group ocmagent --version v1alpha1 \
  --kind MyResource --resource --controller
make generate manifests
# Implement reconciliation, add tests
```

### Update Dependencies
```bash
go get github.com/example/package@v1.2.3
go mod tidy
make go-test
```

### Run Specific Tests
```bash
go test ./controllers/ocmagent/...                    # Package
go test ./controllers/ocmagent/... -run TestName      # Specific test
ginkgo --focus="test description" ./controllers/      # With Ginkgo
```

## Troubleshooting

**`make tools` checksum mismatch**:
```bash
export GOPROXY="https://proxy.golang.org"
go clean -modcache
make tools
```

**Mock generation differs from CI**:
```bash
boilerplate/_lib/container-make generate
```

**Tests fail locally but pass in CI**:
```bash
boilerplate/_lib/container-make test
```

**Can't connect to cluster**:
```bash
oc whoami
oc cluster-info
echo $KUBECONFIG
```

**CRDs not found**:
```bash
oc create -f deploy/crds/
```

**Lint failures**:
```bash
boilerplate/_lib/container-make lint
```

## Resources

- [Operator SDK Documentation](https://sdk.operatorframework.io/)
- [Controller Runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [OpenShift Boilerplate](https://github.com/openshift/boilerplate)
