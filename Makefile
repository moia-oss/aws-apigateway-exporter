include mk-templates/go.mk
include mk-templates/docker.mk

SHELL := /usr/bin/env bash

SERVICE               = aws-apigateway-exporter
BUILD_DIR             = bin
PACKAGES              = $(shell go list -mod=vendor ./...)
LINUX_BINARIES        = $(shell go list -mod=vendor ./... | grep -v -e vendor | awk -F/ '{print "bin/linux_amd64/" $$NF}')
DARWIN_BINARIES       = $(shell go list -mod=vendor ./... | grep -v -e vendor |awk -F/ '{print "bin/darwin_amd64/" $$NF}')
LINT_TARGETS          = $(shell go list -mod=vendor -f '{{.Dir}}' ./... | sed -e"s|${CURDIR}/\(.*\)\$$|\1/...|g" | grep -v ^node_modules/ )
SYSTEM                = $(shell uname -s | tr A-Z a-z)_$(shell uname -m | sed "s/x86_64/amd64/")
DEPENDENCIES          = $(shell find ./vendor -type f)
DOCKER_REGISTRY       = moia
VERSION               = $(shell git describe --always --tags)
DOCKER_IMAGE_TAG      = $(shell git describe --always --tags)
BUILD_TIME            = $(shell date +%FT%T%z)
export GO111MODULE    = on

$(BUILD_DIR)/linux_amd64/%: %/*.go $(DEPENDENCIES)
	env GOOS=linux $(GO) build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o $@ ./$(notdir $@)

$(BUILD_DIR)/darwin_amd64/%: %/*.go $(DEPENDENCIES)
	env GOOS=darwin $(GO) build -ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)" -o $@ ./$(notdir $@)

.PHONY: build
build: $(LINUX_BINARIES) $(DARWIN_BINARIES)

.PHONY: build-linux
build-linux: $(LINUX_BINARIES)

golangci-lint:
	@curl -sSLf \
		https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(shell echo $(SYSTEM) | tr '_' '-').tar.gz \
		| tar xzOf - golangci-lint-$(GOLANGCI_LINT_VERSION)-$(shell echo $(SYSTEM) | tr '_' '-')/golangci-lint > golangci-lint && chmod +x golangci-lint

.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)
