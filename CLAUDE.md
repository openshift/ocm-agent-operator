# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The OCM Agent Operator is a Kubernetes operator that manages the OCM Agent service on managed OpenShift clusters. It's built using the Operator SDK framework and follows standard Kubernetes operator patterns.

## Development Commands

### Setup
```bash
make tools                    # Install required development tools (ginkgo, mockgen, etc.)
```

### Build and Test
```bash
make go-build                 # Build the operator binary
make go-test                  # Run unit tests (also: go test ./...)
make lint                     # Run Go linting and static analysis
```

### Run Locally
```bash
make run                      # Run operator against ~/.kube/config cluster
make run-verbose              # Run with verbose logging (zap-log-level=5)
```

### Container-based Development
```bash
boilerplate/_lib/container-make        # Run make inside boilerplate container
boilerplate/_lib/container-make generate  # Generate mocks in container environment
```

## Architecture

### Core Components
- **API Types** (`api/v1alpha1/`): Defines the OcmAgent, ManagedNotification, and ManagedFleetNotification custom resources
- **Controllers** (`controllers/`):
  - `ocmagent/`: Reconciles OcmAgent resources, manages deployments, services, secrets, etc.
  - `fleetnotification/`: Handles fleet-wide notification resources
- **OCM Agent Handler** (`pkg/ocmagenthandler/`): Business logic for managing OCM Agent resources (deployment, configmap, secrets, network policies, PDB, service monitor)

### Key Custom Resources
- **OcmAgent**: Main resource defining OCM agent configuration, image, token secret, replicas, and fleet mode
- **ManagedNotification**: Handles cluster-specific notifications
- **ManagedFleetNotification**: Manages fleet-wide notifications across multiple clusters

### Dependencies
- Built with Kubernetes 1.31.x APIs
- Uses controller-runtime framework for operator logic
- Integrates with Prometheus for monitoring via ServiceMonitor resources
- Uses Ginkgo/Gomega for testing
- Requires Go 1.22.7+

## Testing

### Framework
- Uses Ginkgo BDD testing framework with Gomega matchers
- Mock generation via GoMock for interface testing
- Tests organized by package with `_suite_test.go` files

### Test Commands
```bash
make go-test                  # Run all tests
ginkgo -r                     # Run tests recursively with Ginkgo
```

### Mock Generation
Regenerate mocks after interface changes:
```bash
boilerplate/_lib/container-make generate
```

## Code Generation

The operator uses code generation for:
- Kubernetes deepcopy methods
- OpenAPI schemas
- Mock interfaces for testing

Always run `make` after modifying API types to regenerate required code.