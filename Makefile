SHELL := /bin/bash
ROOT := $(CURDIR)

.PHONY: all build test fmt tidy generate dev

all: build

build: ## Compile the composepack CLI
	./make/build.sh

test: ## Run Go test suite
	./make/test.sh

fmt: ## Format Go sources with gofmt
	./make/fmt.sh

tidy: ## Run go mod tidy to sync deps
	./make/tidy.sh

generate: ## Run go generate hooks (wire, assets, etc.)
	./make/generate.sh

dev: fmt test build ## Convenience target for local dev loop

help: ## Show available targets
	@grep -E '^[-a-zA-Z0-9_]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?##"} {printf("%-15s %s\n", $$1, $$2)}'

dev-install: ## For development, install the composepack binary into the local bin directory
	./make/build.sh
	./make/dev-install.sh
