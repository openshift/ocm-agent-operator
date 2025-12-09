# OCM Agent Operator Testing

Tests are playing a primary role and we take them seriously.
It is expected from PRs to add, modify or delete tests on case by case scenario.
To contribute you need to be familiar with:

* [Ginkgo](https://github.com/onsi/ginkgo) - BDD Testing Framework for Go
* [Gomega](https://onsi.github.io/gomega/) - Matcher/assertion library

## Prerequisites

Make sure that the [tool dependencies](https://github.com/openshift/ocm-agent-operator/blob/master/docs/development.md#dependencies) are already in place. The `ginkgo` and `mockgen` binaries that are required for testing will be installed as part of tool dependencies.

## Bootstrapping the tests
```
$ cd pkg/samplepackage
$ ginkgo bootstrap
$ ginkgo generate samplepackage.go

find .
./samplepackage.go
./samplepackage_suite_test.go
./samplepackage_test.go
```

## How to run the tests

* You can run the tests using `make test` or `go test ./...`

## Writing tests

### Mocking interfaces

This project makes use of [`GoMock`](https://github.com/uber-go/mock) to mock service interfaces. This comes with the `mockgen` utility which can be used to generate or re-generate mock interfaces that can be used to simulate the behaviour of an external dependency.

It is considered good practice to include a [go generate](https://golang.org/pkg/cmd/go/internal/generate/) directive above the interface which defines the specific `mockgen` command that will generate your mocked interface.

As an example, please see the [controller-runtime client](https://github.com/openshift/ocm-agent-operator/tree/master/pkg/util/test/mockgenerator/client/client.go).

It is important that generating mocks be done via Boilerplate's build files, as this will ensure that the same version of mockgen is used that would be used during CI build pipeline process, which can have an impact on the content of the files that are built.

```
$ boilerplate/_lib/container-make generate
```