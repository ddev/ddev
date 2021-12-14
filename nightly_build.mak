# This makefile is structured to allow building a complete ddev, with clean/fresh containers at current HEAD.

SHELL := /bin/bash

VERSION := $(shell git describe --tags --always --dirty)

VERSION_VARIABLES=DdevVersion WebTag DBTag RouterTag DBATag
WebTag = $(VERSION)
DBTag =  $(VERSION)
RouterTag = $(VERSION)
DBATag = $(VERSION)

# List of containers to be built in containers/ directory
CONTAINER_DIRS = $(shell pushd containers >/dev/null && \ls && popd >/dev/null )

BASEDIR=./containers/

.PHONY: $(CONTAINER_DIRS) all build test clean container build

# Build container dirs then build binaries
all: container test

container: $(CONTAINER_DIRS)

clean:
	for item in $(CONTAINER_DIRS); do \
		echo $$item && $(MAKE) -C $(addprefix $(BASEDIR),$$item) --no-print-directory clean; \
	done
	$(MAKE) clean


$(CONTAINER_DIRS):
	$(MAKE) -C $(addprefix $(BASEDIR),$@) --print-directory test

test:
	$(MAKE) && $(MAKE) VERSION_VARIABLES="$(VERSION_VARIABLES)" WebTag="$(VERSION)" DBTag="$(VERSION)" RouterTag="$(VERSION)" DBATag="$(VERSION)" TESTARGS="" test
