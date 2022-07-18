# OCM Agent Operator Development

## Development Environment Setup

### golang

A recent Go distribution (>=1.17) with enabled Go modules.

```shell
$ go version
go version go1.17.11 linux/amd64
```

### operator-sdk

The Operator is being developed based on the [Operators SDK](https://github.com/operator-framework/operator-sdk).

Ensure this is installed and available in your `$PATH`.

[v1.21.0](https://github.com/operator-framework/operator-sdk/releases/tag/v1.21.0) is being used for `ocm-agent-operator` development.

```shell
$ operator-sdk version
operator-sdk version: "v1.21.0", commit: "89d21a133750aee994476736fa9523656c793588", kubernetes version: "1.23", go version: "go1.17.10", GOOS: "linux", GOARCH: "amd64"
```

## Makefile

All available standardized commands for the `Makefile` are available via:

```shell
$ make
Usage: make <OPTIONS> ... <TARGETS>

Available targets are:

go-build                         Build binary
go-check                         Golang linting and other static analysis
go-test                          runs go test across operator
boilerplate-update               Make boilerplate update itself
help                             Show this help screen.
run                              Run operator locally against the configured Kubernetes cluster in ~/.kube/config
tools                            Install local go tools for OAO
```

## Dependencies

The tool dependencies that are required locally to be present are all part of [tools.go](https://github.com/openshift/ocm-agent-operator/blob/master/tools.go) file. This file will refer the version of the required module from [go.mod](https://github.com/openshift/ocm-agent-operator/blob/master/go.mod) file.

In order to install the tool dependencies locally, run `make tools` at the root of the cloned repository, which will fetch the tools for you and install the binaries at location `$GOPATH/bin` by default:

```shell
$ make tools
```

This will make sure that the installed binaries are always as per the required version mentioned in `go.mod` file. If the version of the module is changed, need to run the command again locally to have new version of tools.

---

**NOTE**

If any of the dependencies are failing to install due to checksum mismatch, try setting `GOPROXY` env variable using `export GOPROXY="https://proxy.golang.org"` and run `make tools` again

---

## Linting

To run lint locally, call `make lint`

```shell
$ make lint
```

## Testing

To run unit tests locally, call `make test`

```shell
$ make go-test
```

## Building

To run go build locally, call `make go-build`

```shell
$ make go-build
```

## Build using boilerplate container

To run lint, test and build in `app-sre/boilerplate` container, call `boilerplate/_lib/container-make`. This will call `make` inside the `app-sre/boilerplate` container.

```shell
$ boilerplate/_lib/container-make
```
