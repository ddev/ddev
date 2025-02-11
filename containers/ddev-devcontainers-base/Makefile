# Docker repo for a push
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-devcontainers-base
DEFAULT_IMAGES = ddev-devcontainers-base

VERSION := $(shell git describe --tags --always --dirty)

BUILD_ARCHS=linux/amd64,linux/arm64

include ../containers_shared.mk

test: container
	bash -c "test/containertest.sh $(DOCKER_REPO):$(VERSION)"
