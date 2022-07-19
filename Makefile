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
