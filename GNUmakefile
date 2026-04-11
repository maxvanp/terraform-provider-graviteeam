BINARY=terraform-provider-graviteeam
HOSTNAME=registry.terraform.io
NAMESPACE=maxvanp
NAME=graviteeam
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)
GOLANGCI_LINT?=golangci-lint
TFPLUGINDOCS?=tfplugindocs
LINT_TIMEOUT?=5m

default: build

build:
	go build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)
	cp $(BINARY) ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)/

clean:
	rm -f $(BINARY)

fmt:
	$(GOLANGCI_LINT) fmt

vet:
	go vet ./...

lint:
	$(GOLANGCI_LINT) run --timeout $(LINT_TIMEOUT)
	$(GOLANGCI_LINT) fmt --diff

test:
	go test ./... -v $(TESTARGS) -timeout 120m

testacc:
	TF_ACC=1 go test ./... -v -p 1 $(TESTARGS) -timeout 120m

docs:
	$(TFPLUGINDOCS) generate --provider-name graviteeam

docs-check: docs
	@git diff --exit-code docs/ || (echo "docs are out of date, run 'make docs'" && exit 1)

.PHONY: build install clean fmt vet lint test testacc docs docs-check
