BINARY=terraform-provider-graviteeam
HOSTNAME=registry.terraform.io
NAMESPACE=maxvanp
NAME=graviteeam
VERSION=0.1.0
GO?=go
OS_ARCH=$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)
GOLANGCI_LINT?=golangci-lint
TFPLUGINDOCS?=tfplugindocs
LINT_TIMEOUT?=5m

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
	./scripts/probe-local-compose-plugins.py --check

lint:
	$(GOLANGCI_LINT) run --timeout $(LINT_TIMEOUT)
	$(GOLANGCI_LINT) fmt --diff

test: coverage-audit
	$(GO) test ./... -v $(TESTARGS) -timeout 120m

testacc:
	TF_ACC=1 $(GO) test ./... -v -p 1 $(TESTARGS) -timeout 120m

docs:
	$(TFPLUGINDOCS) generate --provider-name graviteeam

docs-check: docs
	@git diff --exit-code docs/index.md docs/resources docs/data-sources || (echo "generated docs are out of date, run 'make docs'" && exit 1)

.PHONY: build install clean fmt vet coverage-audit lint test testacc docs docs-check
