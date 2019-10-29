NAME=lino-price-feeder
COMMIT := $(shell git --no-pager describe --tags --always --dirty)
GO111MODULE = on

all: build install

_raw_build_cmd:
	GO111MODULE=$(GO111MODULE) go build -o bin/lino-price-feeder   cmd/pricefeeder/main.go

build:
	make _raw_build_cmd

# lint
GOLANGCI_LINT_VERSION := v1.17.1
GOLANGCI_LINT_HASHSUM := f5fa647a12f658924d9f7d6b9628d505ab118e8e049e43272de6526053ebe08d

get_golangci_lint:
	cd scripts && bash install-golangci-lint.sh $(GOPATH)/bin $(GOLANGCI_LINT_VERSION) $(GOLANGCI_LINT_HASHSUM)

lint:
	GO111MODULE=$(GO111MODULE) golangci-lint run
	GO111MODULE=$(GO111MODULE) go mod verify
	GO111MODULE=$(GO111MODULE) go mod tidy

lint-fix:
	@echo "--> Running linter auto fix"
	GO111MODULE=$(GO111MODULE) golangci-lint run --fix
	GO111MODULE=$(GO111MODULE) find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -d -s
	GO111MODULE=$(GO111MODULE) go mod verify
	GO111MODULE=$(GO111MODULE) go mod tidy

.PHONY: lint lint-fix


.PHONY: all get_tools install build test