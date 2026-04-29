# OCM Agent Operator Testing Guide

This guide provides comprehensive instructions for writing, running, and maintaining tests for the OCM Agent Operator.

## Table of Contents

- [Testing Philosophy](#testing-philosophy)
- [Test Framework](#test-framework)
- [Test Structure](#test-structure)
- [Unit Testing](#unit-testing)
- [E2E Testing](#e2e-testing)
- [Mock Generation](#mock-generation)
- [Running Tests](#running-tests)
- [Writing New Tests](#writing-new-tests)
- [Code Coverage](#code-coverage)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Testing Philosophy

Tests are a primary focus of this project. We take them seriously and expect all contributions to include appropriate tests.

**Key Principles**:
- Tests are **required** for all new features and bug fixes
- Tests should be **clear, maintainable, and meaningful**
- Test coverage should be **maintained or improved** with each PR
- Tests should **run quickly** to support rapid development
- Tests should be **deterministic** and not flaky

## Test Framework

The OCM Agent Operator uses the following testing frameworks:

### Ginkgo

[Ginkgo](https://github.com/onsi/ginkgo) is a BDD (Behavior-Driven Development) testing framework for Go.

**Why Ginkgo?**
- Expressive, readable test structure
- Built-in support for parallel execution
- Rich reporting and debugging features
- Standard in Kubernetes operator development

### Gomega

[Gomega](https://onsi.github.io/gomega/) is a matcher/assertion library that pairs with Ginkgo.

**Why Gomega?**
- Readable, chainable assertions
- Rich set of matchers for common scenarios
- Excellent error messages for debugging
- Async/Eventually support for testing asynchronous code

### GoMock

[GoMock](https://github.com/golang/mock) is used for generating and using mock objects.

**Why GoMock?**
- Generates type-safe mocks from interfaces
- Supports complex expectation scenarios
- Standard in Go testing ecosystem
- Integrates well with Ginkgo/Gomega

## Test Structure

### Directory Organization

```
ocm-agent-operator/
├── controllers/
│   ├── ocmagent/
│   │   ├── ocmagent_controller.go
│   │   ├── ocmagent_controller_test.go
│   │   └── ocmagent_suite_test.go
│   └── fleetnotification/
│       ├── fleetnotification_controller.go
│       ├── fleetnotification_controller_test.go
│       └── fleetnotification_suite_test.go
├── pkg/
│   ├── ocmagenthandler/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── handler_suite_test.go
│   └── util/
│       └── test/
│           └── mockgenerator/
│               └── client/
│                   ├── client.go          # Interface with //go:generate directive
│                   └── mocks/
│                       └── mock_client.go  # Generated mock
└── test/
    └── e2e/
        ├── README.md
        ├── ocmagent_test.go
        └── e2e_suite_test.go
```

### Test File Naming Conventions

- **Test files**: `*_test.go`
- **Suite files**: `*_suite_test.go` (bootstraps Ginkgo for the package)
- **Mock files**: `mock_*.go` in a `mocks/` subdirectory

## Unit Testing

### Running Unit Tests

```bash
# Run all unit tests
make go-test

# Alternative: use go test directly
go test ./...

# Run tests with Ginkgo for better output
ginkgo -r

# Run tests in a specific package
go test ./controllers/ocmagent/...

# Run tests with coverage
go test ./... -coverprofile=coverage.out

# Run in container (matches CI)
boilerplate/_lib/container-make go-test
```

### Test Output

Successful test run:
```
Running Suite: OcmAgent Controller Suite
=========================================
Random Seed: 1234567890

Will run 15 of 15 specs
••••••••••••••• 

Ran 15 of 15 Specs in 2.345 seconds
SUCCESS! -- 15 Passed | 0 Failed | 0 Pending | 0 Skipped
```

### Test Structure Example

```go
package ocmagent_test

import (
    "context"
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    
    "github.com/openshift/ocm-agent-operator/controllers/ocmagent"
)

// Test suite setup
func TestOcmAgentController(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "OcmAgent Controller Suite")
}

var _ = Describe("OcmAgent Controller", func() {
    Context("When reconciling an OcmAgent resource", func() {
        It("Should create a deployment", func() {
            // Arrange
            ctx := context.Background()
            ocmAgent := &v1alpha1.OcmAgent{
                // ... setup
            }

            // Act
            err := k8sClient.Create(ctx, ocmAgent)

            // Assert
            Expect(err).NotTo(HaveOccurred())
            
            deployment := &appsv1.Deployment{}
            Eventually(func() error {
                return k8sClient.Get(ctx, types.NamespacedName{
                    Name:      "ocm-agent",
                    Namespace: "openshift-ocm-agent-operator",
                }, deployment)
            }, timeout, interval).Should(Succeed())
            
            Expect(deployment.Spec.Replicas).To(Equal(ptr.To(int32(1))))
        })
    })
})
```

### Using Mocks in Tests

```go
import (
    "github.com/golang/mock/gomock"
    "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks"
)

var _ = Describe("Handler", func() {
    var (
        ctrl       *gomock.Controller
        mockClient *mocks.MockClient
        handler    *Handler
    )

    BeforeEach(func() {
        ctrl = gomock.NewController(GinkgoT())
        mockClient = mocks.NewMockClient(ctrl)
        handler = NewHandler(mockClient)
    })

    AfterEach(func() {
        ctrl.Finish()
    })

    It("Should call client.Create", func() {
        // Setup expectations
        mockClient.EXPECT().
            Create(gomock.Any(), gomock.Any()).
            Return(nil)

        // Execute
        err := handler.CreateDeployment(context.Background())

        // Verify
        Expect(err).NotTo(HaveOccurred())
    })
})
```

## E2E Testing

E2E (End-to-End) tests validate the operator behavior in a real Kubernetes cluster.

### E2E Test Location

E2E tests are in `test/e2e/`

### Prerequisites for E2E Tests

1. **Access to an OpenShift or Kubernetes cluster**
2. **OCM Agent image** to deploy
3. **Cluster credentials** (kubeadmin access)

### Running E2E Tests Locally

```bash
# 1. Build the e2e test binary
make e2e-binary-build

# 2. Deploy the operator to your test cluster
oc create -f deploy/crds/ocmagent.managed.openshift.io_ocmagents.yaml
find test/deploy -type f -name '*.yaml' -exec oc create -f {} \;

# 3. Install Ginkgo (if not already installed)
go install github.com/onsi/ginkgo/ginkgo@latest

# 4. Get cluster credentials
# Replace (cluster-id) with your actual cluster ID
ocm get /api/clusters_mgmt/v1/clusters/(cluster-id)/credentials | jq -r .kubeconfig > /tmp/kubeconfig

# 5. Run the E2E test suite
DISABLE_JUNIT_REPORT=true KUBECONFIG=/tmp/kubeconfig ginkgo --tags=osde2e -v test/e2e
```

### E2E Test Structure

```go
package e2e_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("OCM Agent Operator E2E", func() {
    Context("When deploying an OcmAgent", func() {
        It("Should create all required resources", func() {
            // Test that deployment, service, configmap, etc. are created
        })

        It("Should expose metrics", func() {
            // Test that metrics endpoint is accessible
        })

        It("Should handle OcmAgent updates", func() {
            // Test updating the OcmAgent CR
        })
    })
})
```

### E2E Test Environment Variables

- `KUBECONFIG`: Path to cluster kubeconfig
- `DISABLE_JUNIT_REPORT`: Set to `true` to disable JUnit XML output
- `E2E_TIMEOUT`: Test timeout (default: 30m)

## Mock Generation

### Why Generate Mocks?

Mocks allow unit testing of components without requiring real Kubernetes clusters or external dependencies.

### Interfaces with Mock Generation

Interfaces that need mocks have `//go:generate` directives:

```go
//go:generate mockgen -destination=mocks/mock_client.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Client

type MyInterface interface {
    DoSomething(ctx context.Context) error
}
```

### Generating Mocks

**Important**: Always generate mocks in the boilerplate container for consistency with CI.

```bash
# Generate all mocks
boilerplate/_lib/container-make generate
```

**Why use the container?**
- Ensures same `mockgen` version as CI
- Prevents inconsistent mock files
- Avoids "mock out of date" CI failures

### Mock File Location

Generated mocks are typically placed in a `mocks/` subdirectory:

```
pkg/util/test/mockgenerator/client/
├── client.go              # Interface with //go:generate directive
└── mocks/
    └── mock_client.go     # Generated mock
```

### Using Generated Mocks

```go
import (
    "github.com/golang/mock/gomock"
    mocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks"
)

ctrl := gomock.NewController(GinkgoT())
defer ctrl.Finish()

mockClient := mocks.NewMockClient(ctrl)
mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
```

## Running Tests

### All Tests

```bash
make go-test
```

### Specific Package

```bash
go test ./controllers/ocmagent/...
```

### Specific Test

```bash
# Using go test
go test ./controllers/ocmagent/... -run TestOcmAgentController

# Using ginkgo with focus
ginkgo --focus="Should create a deployment" ./controllers/ocmagent/
```

### Parallel Execution

```bash
# Run tests in parallel
ginkgo -r -p

# Specify number of parallel processes
ginkgo -r -p -nodes=4
```

### Verbose Output

```bash
# Verbose with ginkgo
ginkgo -r -v

# Very verbose with go test
go test -v ./...
```

### Watch Mode

```bash
# Re-run tests on file changes
ginkgo watch -r
```

### With Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out

# Check coverage percentage
go test ./... -cover
```

## Writing New Tests

### Bootstrapping a New Test Suite

```bash
# Navigate to package
cd pkg/mypackage

# Bootstrap Ginkgo suite
ginkgo bootstrap

# Generate test file
ginkgo generate mypackage.go
```

This creates:
- `mypackage_suite_test.go` - Suite setup
- `mypackage_test.go` - Test file

### Test File Template

```go
package mypackage_test

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestMyPackage(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "MyPackage Suite")
}

var _ = Describe("MyComponent", func() {
    var (
        component *MyComponent
    )

    BeforeEach(func() {
        component = NewMyComponent()
    })

    Context("When doing something", func() {
        It("Should succeed", func() {
            result, err := component.DoSomething()
            Expect(err).NotTo(HaveOccurred())
            Expect(result).To(Equal(expectedValue))
        })

        It("Should handle errors", func() {
            result, err := component.DoInvalidThing()
            Expect(err).To(HaveOccurred())
            Expect(result).To(BeNil())
        })
    })
})
```

### Common Gomega Matchers

```go
// Equality
Expect(value).To(Equal(expected))
Expect(value).NotTo(Equal(unexpected))

// Nil checks
Expect(err).NotTo(HaveOccurred())
Expect(err).To(HaveOccurred())
Expect(value).To(BeNil())
Expect(value).NotTo(BeNil())

// Boolean
Expect(condition).To(BeTrue())
Expect(condition).To(BeFalse())

// Collections
Expect(slice).To(HaveLen(3))
Expect(slice).To(ContainElement(item))
Expect(slice).To(BeEmpty())

// Async/Eventually
Eventually(func() error {
    return checkCondition()
}, timeout, interval).Should(Succeed())

Eventually(func() int {
    return getCount()
}, "10s", "1s").Should(Equal(5))

// Panics
Expect(func() { panickyFunction() }).To(Panic())
```

### Testing Best Practices

1. **Use Descriptive Test Names**
   ```go
   It("Should create a deployment when OcmAgent is created", func() {
   ```

2. **Follow Arrange-Act-Assert Pattern**
   ```go
   It("Should validate input", func() {
       // Arrange
       input := "invalid"
       
       // Act
       err := validator.Validate(input)
       
       // Assert
       Expect(err).To(HaveOccurred())
   })
   ```

3. **Use BeforeEach for Setup**
   ```go
   BeforeEach(func() {
       ctrl = gomock.NewController(GinkgoT())
       mockClient = mocks.NewMockClient(ctrl)
   })
   ```

4. **Use AfterEach for Cleanup**
   ```go
   AfterEach(func() {
       ctrl.Finish()
   })
   ```

5. **Test Both Success and Failure Cases**
   ```go
   Context("When creating resources", func() {
       It("Should succeed with valid input", func() { /* ... */ })
       It("Should fail with invalid input", func() { /* ... */ })
       It("Should handle API errors", func() { /* ... */ })
   })
   ```

## Code Coverage

### Checking Coverage

```bash
# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage percentage
go test ./... -cover

# View coverage by file
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

### Coverage Goals

- Maintain or improve overall coverage with each PR
- Aim for meaningful tests, not just coverage numbers
- Focus on testing critical paths and edge cases
- Don't worry about 100% coverage for generated code

### Excluded from Coverage

- `**/mocks/` - Generated mock files
- `**/zz_generated*.go` - Generated Kubernetes code
- Boilerplate code

## CI/CD Integration

### CI Test Execution

Tests run automatically in CI using:

```bash
boilerplate/_lib/container-make go-test
```

### CI Linting

```bash
boilerplate/_lib/container-make lint
```

### Matching CI Locally

To exactly match CI behavior:

```bash
# Run tests in container
boilerplate/_lib/container-make go-test

# Run linting in container
boilerplate/_lib/container-make lint

# Validate generated code
boilerplate/_lib/container-make validate
```

### Codecov Integration

Code coverage reports are uploaded to [Codecov](https://codecov.io/gh/openshift/ocm-agent-operator).

Configuration in `.codecov.yml`:
- Ignores `**/mocks` directories
- Ignores `**/zz_generated*.go` files

## Best Practices

### Do's

- ✅ Write tests for all new features
- ✅ Update tests when modifying existing code
- ✅ Use descriptive test names
- ✅ Test both success and failure scenarios
- ✅ Use mocks for external dependencies
- ✅ Keep tests fast and deterministic
- ✅ Run tests before submitting PRs
- ✅ Generate mocks in boilerplate container

### Don'ts

- ❌ Don't skip tests for "simple" changes
- ❌ Don't write flaky tests
- ❌ Don't test implementation details
- ❌ Don't use time.Sleep in tests (use Eventually)
- ❌ Don't generate mocks locally (use container)
- ❌ Don't commit test failures
- ❌ Don't reduce coverage without justification

## Troubleshooting

### Issue: Tests Fail with "mock out of date"

**Solution**: Regenerate mocks in container
```bash
boilerplate/_lib/container-make generate
git add .
git commit -m "chore: regenerate mocks"
```

### Issue: Tests Pass Locally but Fail in CI

**Solution**: Run tests in container to match CI
```bash
boilerplate/_lib/container-make go-test
```

### Issue: Ginkgo Not Found

**Solution**: Install tools
```bash
make tools
export PATH=$PATH:$(go env GOPATH)/bin
```

### Issue: Tests Time Out

**Solution**: Increase timeout or investigate slow tests
```go
Eventually(func() error {
    // ...
}, "30s", "1s").Should(Succeed())  // Increased timeout
```

### Issue: Flaky Tests

**Causes**:
- Race conditions
- Hardcoded timing assumptions
- Shared state between tests

**Solutions**:
- Use `Eventually` instead of fixed waits
- Ensure proper test isolation
- Use gomock strict mode

### Issue: Coverage Decreased

**Solution**: Add tests for new code
```bash
# Check which files lack coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -v "100.0%"
```

## Additional Resources

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/#provided-matchers)
- [GoMock Documentation](https://github.com/golang/mock)
- [Controller Runtime Testing](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
- [Kubernetes Testing Best Practices](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing.md)
