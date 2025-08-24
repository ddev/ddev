
VERSION := $(shell git describe --tags --always --dirty)

DEFAULT_IMAGES = ddev-ssh-agent

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS?=linux/amd64,linux/arm64

include ../containers_shared.mk

DOCKER_REPO ?= $(DOCKER_ORG)/ddev-ssh-agent

test: container
	test/test.sh $(DOCKER_REPO):$(VERSION)
