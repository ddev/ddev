# Docker repo for a push
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-gitpod-base
DEFAULT_IMAGES = ddev-gitpod-base

VERSION := $(shell git describe --tags --always --dirty)

# The upstream is amd64. In https://github.com/ddev/ddev/pull/6613
# there was an attempt to build both arm64 and amd64 and perhaps we
# should return to that one day if needed.
BUILD_ARCHS=linux/amd64

include ../containers_shared.mk

test: container
	bash -c "test/containertest.sh $(DOCKER_REPO):$(VERSION)"
