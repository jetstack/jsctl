REPO_ROOT ?= $(shell pwd)
GO ?= $(shell which go)

local:
	goreleaser release --snapshot --skip-publish --rm-dist

.PHONY: docs-gen
docs-gen:
	$(GO) run $(REPO_ROOT)/tools/cobra/main.go "$(REPO_ROOT)/docs/reference"
