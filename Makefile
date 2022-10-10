REPO_ROOT ?= $(shell pwd)
GO ?= $(shell which go)

 .PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


.PHONY: test
test: ## Run all unit tests
	$(GO) test ./...

.PHONY: docs-gen
docs-gen: ## Regenerate command line docs
	$(GO) run $(REPO_ROOT)/tools/cobra/main.go "$(REPO_ROOT)/docs/reference"
