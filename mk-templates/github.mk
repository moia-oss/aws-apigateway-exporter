# this makefile can be used to create a Github Release in a repository, with all binaries for
# linux and darwin as seperate applications
GITHUB_OWNER          = moia-dev
GITHUB_REPOSITORY     = $(shell basename `git rev-parse --show-toplevel`)

ifdef GIT_VERSION
	VERSION = ${GIT_VERSION}
else
	VERSION = $(shell git describe --always --tags --dirty)
endif

# we need to do some magic here, because importing this will not work when we are not
# in this folder, e.g. from ../ the include will fail-- make is not smart.
#
# the last word of the MAKEFILE_LIST is the current makefile, so we can take that
# and append it to the include directory so that it will always be accurate
#
# not that you cannot use make -f with this approach, and must run the make targets
# in the same directory as the Makefile
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
include $(SELF_DIR)/common.mk

.PHONY: release-dependencies
release-dependencies:
	go get -u github.com/aktau/github-release

.PHONY: release
release: guard-VERSION release-dependencies
	$(if $(GITHUB_TOKEN),,$(eval GITHUB_TOKEN=$(call ssm-get,/Github/ApiToken)))
	github-release info --user $(GITHUB_OWNER) --repo $(GITHUB_REPOSITORY) -s $(GITHUB_TOKEN)
	github-release release --user $(GITHUB_OWNER) --repo $(GITHUB_REPOSITORY) --tag $(VERSION) --name $(VERSION) -s $(GITHUB_TOKEN)
	for f in bin/linux_amd64/*; do github-release upload --user $(GITHUB_OWNER) --repo $(GITHUB_REPOSITORY) -s $(GITHUB_TOKEN) --tag $(VERSION) --name `basename $${f}`_linux_amd64 --file $${f}; done
	for f in bin/darwin_amd64/*; do github-release upload --user $(GITHUB_OWNER) --repo $(GITHUB_REPOSITORY) -s $(GITHUB_TOKEN) --tag $(VERSION) --name `basename $${f}`_darwin_amd64 --file $${f}; done
	github-release edit --user $(GITHUB_OWNER) --repo $(GITHUB_REPOSITORY) -s $(GITHUB_TOKEN) --tag $(VERSION) --name $(VERSION) --description $(VERSION)
