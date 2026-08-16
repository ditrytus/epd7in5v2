GOLANGCI_LINT_VERSION ?= v2.12.2

.PHONY: build vet test test-cover lint lint-fix tools check

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

test-cover:
	go test -cover ./...

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

# Installs the pinned golangci-lint into $(go env GOPATH)/bin.
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# Everything CI runs, in the same order.
check: build vet lint test
