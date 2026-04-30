# OCM Agent Operator Testing

Tests are a primary focus of this project. All contributions must include appropriate tests.

## Test Framework

- **Ginkgo** - BDD testing framework
- **Gomega** - Matcher/assertion library  
- **GoMock** - Interface mocking

## Running Tests

### Unit Tests
```bash
make go-test                    # All tests
go test ./...                   # Alternative
ginkgo -r                       # With Ginkgo

# Specific package
go test ./controllers/ocmagent/...

# Specific test
go test ./controllers/ocmagent/... -run TestName
ginkgo --focus="description" ./controllers/

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# In container (matches CI)
boilerplate/_lib/container-make test
```

### E2E Tests

Prerequisites:
- Access to OpenShift/Kubernetes cluster
- OCM Agent image
- Cluster credentials

```bash
# 1. Build E2E binary
make e2e-binary-build

# 2. Deploy operator
oc create -f deploy/crds/ocmagent.managed.openshift.io_ocmagents.yaml
find test/deploy -type f -name '*.yaml' -exec oc create -f {} \;

# 3. Get credentials
ocm get /api/clusters_mgmt/v1/clusters/(cluster-id)/credentials | jq -r .kubeconfig > /tmp/kubeconfig

# 4. Run E2E
DISABLE_JUNIT_REPORT=true KUBECONFIG=/tmp/kubeconfig ginkgo --tags=osde2e -v test/e2e
```

## Writing Tests

### Bootstrap New Tests
```bash
cd pkg/mypackage
ginkgo bootstrap              # Creates suite file
ginkgo generate mypackage.go  # Creates test file
```

### Test Structure
```go
package mypackage_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("MyComponent", func() {
    var component *MyComponent
    
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
        })
    })
})
```

### Common Matchers
```go
// Equality
Expect(value).To(Equal(expected))
Expect(value).NotTo(Equal(unexpected))

// Errors
Expect(err).NotTo(HaveOccurred())
Expect(err).To(HaveOccurred())

// Nil
Expect(value).To(BeNil())
Expect(value).NotTo(BeNil())

// Boolean
Expect(condition).To(BeTrue())
Expect(condition).To(BeFalse())

// Collections
Expect(slice).To(HaveLen(3))
Expect(slice).To(ContainElement(item))
Expect(slice).To(BeEmpty())

// Async
Eventually(func() error {
    return checkCondition()
}, timeout, interval).Should(Succeed())

Eventually(func() int {
    return getCount()
}, "10s", "1s").Should(Equal(5))
```

### Using Mocks
```go
import (
    "github.com/golang/mock/gomock"
    mocks "github.com/openshift/ocm-agent-operator/pkg/util/test/generated/mocks"
)

var _ = Describe("Handler", func() {
    var (
        ctrl       *gomock.Controller
        mockClient *mocks.MockClient
    )
    
    BeforeEach(func() {
        ctrl = gomock.NewController(GinkgoT())
        mockClient = mocks.NewMockClient(ctrl)
    })
    
    AfterEach(func() {
        ctrl.Finish()
    })
    
    It("Should call client.Create", func() {
        mockClient.EXPECT().
            Create(gomock.Any(), gomock.Any()).
            Return(nil)
        
        err := handler.CreateDeployment(context.Background())
        Expect(err).NotTo(HaveOccurred())
    })
})
```

## Mock Generation

### Generating Mocks

**CRITICAL**: Always use container to match CI mockgen version.

```bash
boilerplate/_lib/container-make generate
```

### Adding Mock Directives

Above interfaces in source files:
```go
//go:generate mockgen -destination=mocks/mock_client.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Client

type MyInterface interface {
    DoSomething(ctx context.Context) error
}
```

Then run `boilerplate/_lib/container-make generate`.

## Best Practices

### Do's
- ✅ Write tests for all new features
- ✅ Update tests when modifying code
- ✅ Use descriptive test names
- ✅ Test both success and failure scenarios
- ✅ Use mocks for external dependencies
- ✅ Run tests before submitting PRs
- ✅ Generate mocks in container

### Don'ts
- ❌ Don't skip tests for "simple" changes
- ❌ Don't write flaky tests
- ❌ Don't use `time.Sleep` (use `Eventually`)
- ❌ Don't generate mocks locally
- ❌ Don't commit test failures

### Test Organization
- Use `BeforeEach` for setup
- Use `AfterEach` for cleanup
- Follow Arrange-Act-Assert pattern
- Test edge cases and error handling

## Code Coverage

```bash
# Generate coverage
go test ./... -coverprofile=coverage.out

# View percentage
go test ./... -cover

# View by file
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

**Goals**:
- Maintain or improve coverage with each PR
- Focus on meaningful tests, not just coverage numbers
- Test critical paths and edge cases

**Excluded from coverage**:
- `**/mocks/` - Generated mocks
- `**/zz_generated*.go` - Generated Kubernetes code

## CI Integration

CI runs:
```bash
boilerplate/_lib/container-make go-test
boilerplate/_lib/container-make lint
```

To match CI locally:
```bash
boilerplate/_lib/container-make go-test
boilerplate/_lib/container-make validate
```

Coverage uploaded to [Codecov](https://codecov.io/gh/openshift/ocm-agent-operator).

## Troubleshooting

**"mock out of date"**:
```bash
boilerplate/_lib/container-make generate
```

**Tests pass locally but fail in CI**:
```bash
boilerplate/_lib/container-make test
```

**Ginkgo not found**:
```bash
make tools
export PATH=$PATH:$(go env GOPATH)/bin
```

**Flaky tests**:
- Use `Eventually` instead of fixed waits
- Ensure test isolation
- Avoid shared state

**Coverage decreased**:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -v "100.0%"
```

## Resources

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
- [GoMock Documentation](https://github.com/golang/mock)
