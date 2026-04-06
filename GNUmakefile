BINARY=terraform-provider-graviteeam
HOSTNAME=registry.terraform.io
NAMESPACE=maxvanp
NAME=graviteeam
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

default: build

build:
	go build -o $(BINARY)

install: build
	mkdir -p ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)
	cp $(BINARY) ~/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(NAME)/$(VERSION)/$(OS_ARCH)/

clean:
	rm -f $(BINARY)

fmt:
	gofmt -w internal/ main.go

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./... -v $(TESTARGS) -timeout 120m

testacc:
	TF_ACC=1 go test ./... -v -p 1 $(TESTARGS) -timeout 120m

docs:
	tfplugindocs generate --provider-name graviteeam

docs-check: docs
	@git diff --exit-code docs/ || (echo "docs are out of date, run 'make docs'" && exit 1)

.PHONY: build install clean fmt vet lint test testacc docs docs-check
