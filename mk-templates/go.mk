# This overrides the default go command withenv Variables which are usually
# used in moia-dev repositories
SYSTEM                := $(shell uname -s | tr A-Z a-z)_$(shell uname -m | sed "s/x86_64/amd64/")
GO_PREFIX             := CGO_ENABLED=0 GOFLAGS=-mod=vendor GOPRIVATE=github.com/moia-dev
GO                    := $(GO_PREFIX) go
# This collects every path, which contains go files in the current project
LINT_TARGETS          ?= $(shell $(GO) list -f '{{.Dir}}' ./... | sed -e"s|${CURDIR}/\(.*\)\$$|\1/...|g" | grep -v node_modules/ )
# The current version of golangci-lint. 
# See: https://github.com/golangci/golangci-lint/releases
GOLANGCI_LINT_VERSION := 1.30.0

# Executes the linter on all our go files inside of the project
.PHONY: lint
lint: bin/golangci-lint-$(GOLANGCI_LINT_VERSION)
	$(GO_PREFIX) ./bin/golangci-lint-$(GOLANGCI_LINT_VERSION) --timeout 120s run $(LINT_TARGETS)

.PHONY: create-golint-config
create-golint-config: .golangci.yml

.golangci.yml:
	cp moia-mk-templates/assets/golangci.yml $@

# Downloads the current golangci-lint executable into the bin directory and
# makes it executable
bin/golangci-lint-$(GOLANGCI_LINT_VERSION):
	mkdir -p bin
	curl -sSLf \
		https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(shell echo $(SYSTEM) | tr '_' '-').tar.gz \
		| tar xzOf - golangci-lint-$(GOLANGCI_LINT_VERSION)-$(shell echo $(SYSTEM) | tr '_' '-')/golangci-lint > bin/golangci-lint-$(GOLANGCI_LINT_VERSION) && chmod +x bin/golangci-lint-$(GOLANGCI_LINT_VERSION)

# Creates and pushes a branch with all golang specific dependabot updates
PHONY: dependabot-batch
dependabot-batch:
	$(eval NOW := $(shell date +%s))
	git checkout -b update-deps-$(NOW) && for i in $(shell git branch -a | grep "/origin/dependabot/go_modules") ; do git cherry-pick $$i ; done && go mod vendor && go mod tidy && git add vendor go.mod go.sum && git commit -m "vendor dependencies" && git push --set-upstream origin update-deps-$(NOW)
