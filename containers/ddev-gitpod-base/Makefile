# Docker repo for a push
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-gitpod-base

VERSION := $(shell git describe --tags --always --dirty)

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS=linux/amd64,linux/arm64

include ../containers_shared.mk

test: container
	bash -c "test/containertest.sh $(DOCKER_REPO):$(VERSION)"
