BINARY=terraform-provider-graviteeam
HOSTNAME=registry.terraform.io
NAMESPACE=maxvanp
NAME=graviteeam
VERSION=0.1.0
GO?=$(or $(shell command -v go 2>/dev/null),$(wildcard /usr/local/go/bin/go),go)
OS_ARCH=$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)
GOLANGCI_LINT?=golangci-lint
TFPLUGINDOCS?=$(or $(shell command -v tfplugindocs 2>/dev/null),$(wildcard $(HOME)/go/bin/tfplugindocs),tfplugindocs)
TFPLUGINDOCS_VERSION?=v0.25.0
LINT_TIMEOUT?=5m
DOCS_GENERATE=env 'PATH=/usr/local/go/bin:$(HOME)/go/bin:$(PATH)' $(TFPLUGINDOCS) generate --provider-name graviteeam

default: build

build:
	$(GO) build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)
	cp $(BINARY) ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)/

clean:
	rm -f $(BINARY)

fmt:
	$(GOLANGCI_LINT) fmt

vet:
	$(GO) vet ./...

coverage-audit:
	./scripts/audit-openapi-coverage.py --check-doc
	./scripts/audit-test-coverage.py --check
	./scripts/audit-resource-schema-fields.py --check
	./scripts/audit-special-resource-coverage.py --check
	./scripts/probe-local-compose-plugins.py --check

local-gap-probe:
	./scripts/probe-openapi-gaps.py --check

release-check:
	goreleaser check
	goreleaser release --snapshot --clean --skip=sign
	./scripts/verify-release-artifacts.sh
	python3 scripts/test-release-artifacts.py

lint:
	$(GOLANGCI_LINT) run --timeout $(LINT_TIMEOUT)
	$(GOLANGCI_LINT) fmt --diff

test: coverage-audit
	$(GO) test ./... -v $(TESTARGS) -timeout 120m

testacc:
	TF_ACC=1 $(GO) test ./... -v -p 1 $(TESTARGS) -timeout 120m

coverage-baseline:
	$(GO) test ./... -coverprofile=/tmp/graviteeam-coverage.out -covermode=atomic
	./scripts/update-test-coverage-baseline.py --go "$(GO)" --coverprofile /tmp/graviteeam-coverage.out

verify-local: coverage-baseline docs-check coverage-audit
	$(GO) test ./... -count=1
	git diff --check

docs-tool:
	@if ! env 'PATH=/usr/local/go/bin:$(HOME)/go/bin:$(PATH)' command -v $(TFPLUGINDOCS) >/dev/null 2>&1 && [ ! -x "$(TFPLUGINDOCS)" ]; then \
		echo "installing tfplugindocs $(TFPLUGINDOCS_VERSION)"; \
		env 'PATH=/usr/local/go/bin:$(PATH)' GOBIN="$(HOME)/go/bin" $(GO) install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION); \
	fi

docs: docs-tool
	@log=$$(mktemp); \
	if ! $(DOCS_GENERATE) >"$$log" 2>&1; then \
		cat "$$log"; \
		rm -f "$$log"; \
		exit 1; \
	fi; \
	rm -f "$$log"

docs-check: docs-tool
	@log=$$(mktemp); \
	if ! $(DOCS_GENERATE) >"$$log" 2>&1; then \
		cat "$$log"; \
		rm -f "$$log"; \
		exit 1; \
	fi; \
	rm -f "$$log"
	@git diff --exit-code -- docs/index.md docs/resources docs/data-sources || (echo "generated docs are out of date; run 'make docs' and commit the generated changes" && exit 1)

.PHONY: build install clean fmt vet coverage-audit local-gap-probe release-check lint test testacc coverage-baseline verify-local docs-tool docs docs-check
