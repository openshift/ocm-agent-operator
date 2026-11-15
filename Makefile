export KONFLUX_BUILDS=true
# Pin the boilerplate backing image. boilerplate/update forces `latest` when
# this is unset, and latest was rebuilt on ubi 9.9 with Go 1.27, which breaks
# golangci-lint 2.7.2 and GOEXPERIMENT=boringcrypto. Remove this pin once
# boilerplate supports Go 1.27.
export LATEST_IMAGE_TAG := image-v8.4.4
FIPS_ENABLED=true
include boilerplate/generated-includes.mk

OPERATOR_NAME=ocm-agent-operator

.PHONY: boilerplate-update
boilerplate-update: ## Make boilerplate update itself
	@boilerplate/update

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run:
	OPERATOR_NAMESPACE="openshift-ocm-agent-operator" go run ./main.go

.PHONY: run-verbose
run-verbose:
	OPERATOR_NAMESPACE="openshift-ocm-agent-operator" go run ./main.go --zap-log-level=5

.PHONY: tools
tools: ## Install local go tools for OAO
	cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: help
help: ## Show this help screen.
		@echo 'Usage: make <OPTIONS> ... <TARGETS>'
		@echo ''
		@echo 'Available targets are:'
		@echo ''
		@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | sed 's/##//g' | awk 'BEGIN {FS = ":"}; {printf "\033[36m%-30s\033[0m %s\n", $$2, $$3}'
