# Makefile for a standard repo with associated image

include ../containers_shared.mk

##### These variables need to be adjusted in most repositories #####
SHELL=/bin/bash

DEFAULT_IMAGES = ddev-php-base ddev-php-prod
BUILD_ARCHS?=linux/amd64,linux/arm64
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-php-base

.PHONY: images

# VERSION can be set by
  # Default: git tag
  # make command line: make VERSION=0.9.0
# It can also be explicitly set in the Makefile as commented out below.

# This version-strategy uses git tags to set the version string
# VERSION can be overridden on make commandline: make VERSION=0.9.1 push
VERSION := $(shell git describe --tags --always --dirty)
BUILDINFO = $(shell echo hash=$$(git rev-parse --short HEAD) Built $$(date) by $${USER} on $$(hostname) $(BUILD_IMAGE) )

# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

#
# This version-strategy uses a manual value to set the version string
#VERSION := 1.2.3

build: images

images: $(DEFAULT_IMAGES)

$(DEFAULT_IMAGES):
	set -eu -o pipefail; \
	docker buildx build --label com.ddev.buildhost=${shell hostname} --target=$@  -t $(DOCKER_ORG)/$@:$(VERSION) $(DOCKER_ARGS) .

test: $(DEFAULT_IMAGES)
	docker buildx build --label com.ddev.buildhost=${shell hostname} --label com.ddev.buildinfo="$(BUILDINFO)" -t $(DOCKER_ORG)/$<:$(VERSION) $(DOCKER_ARGS) .
	for item in $(DEFAULT_IMAGES); do \
		if [ -x tests/$$item/test.sh ]; then tests/$$item/test.sh $(DOCKER_ORG)/$$item:$(VERSION); fi; \
	done

version:
	@echo VERSION:$(VERSION)
