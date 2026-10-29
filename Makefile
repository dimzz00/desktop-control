# SPDX-License-Identifier: AGPL-3.0-or-later

SHELL := /bin/sh

GO ?= go
VERSION := $(shell cat VERSION)
LDFLAGS := -X github.com/dimzz00/kendali/internal/versioncmd.BuildVersion=$(VERSION)
COVERAGE_MIN ?= 80

# `make tools` installs the pinned linters here instead of touching GOPATH/bin,
# so lint and tools always agree on the same versions.
TOOLS_DIR ?= $(CURDIR)/bin/tools
STATICCHECK_VERSION ?= v0.8.2
GOLANGCI_LINT_VERSION ?= v2.15.0

.PHONY: build test vet fmt-check lint coverage license verify hooks tools

build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/rc ./cmd/rc

test:
	$(GO) test -race -count=1 ./...

vet:
	$(GO) vet ./...

fmt-check:
	@dirs="$$($(GO) list -f '{{.Dir}}' ./...)" || exit 1; \
	files="$$(gofmt -l $${dirs})"; \
	if [ -n "$$files" ]; then \
		echo "fmt-check: these files need gofmt:"; \
		echo "$$files"; \
		exit 1; \
	fi

lint:
	@missing=""; \
	for tool in staticcheck golangci-lint; do \
		if ! command -v "$$tool" >/dev/null 2>&1 && [ ! -x "$(TOOLS_DIR)/$$tool" ]; then \
			missing="$$missing $$tool"; \
		fi; \
	done; \
	if [ -n "$$missing" ]; then \
		echo "lint: missing tool(s):$$missing"; \
		echo "lint: run 'make tools' to install the pinned versions into $(TOOLS_DIR)"; \
		exit 1; \
	fi; \
	PATH="$(TOOLS_DIR):$$PATH"; export PATH; \
	staticcheck ./... && golangci-lint run ./...

coverage:
	./scripts/coverage.sh $(COVERAGE_MIN)

license:
	./scripts/license.sh

verify: fmt-check vet lint test coverage license
	@echo "verify: ok"

hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit scripts/*.sh

tools:
	GOBIN="$(TOOLS_DIR)" $(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	GOBIN="$(TOOLS_DIR)" $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
