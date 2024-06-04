SYSTEM                := $(shell uname -s | tr A-Z a-z)_$(shell uname -m | sed "s/x86_64/amd64/")
# The current version of the jsonnet-bundler
# See: https://github.com/jsonnet-bundler/jsonnet-bundler/releases
JB_VERSION := 0.4.0

.PHONY: bin/jb
bin/jb: bin/jb-$(JB_VERSION)

# Downloads the current jsonnet-bundler executable into the bin directory and
# makes it executable
bin/jb-$(JB_VERSION):
	mkdir -p bin
	curl -sSLf \
		https://github.com/jsonnet-bundler/jsonnet-bundler/releases/download/v$(JB_VERSION)/jb-$(shell echo $(SYSTEM) | tr '_' '-') \
		-o $@ && chmod +x $@ && ln -s $@ bin/jb
