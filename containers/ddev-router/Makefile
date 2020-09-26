# Docker repo for a push
DOCKER_REPO ?= drud/ddev-router

# This version-strategy uses git tags to set the version string
# VERSION can be overridden on make commandline: make VERSION=0.9.1 push
VERSION := $(shell git describe --tags --always --dirty)
#
# This version-strategy uses a manual value to set the version string
#VERSION := 1.2.3

# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS=linux/amd64,linux/arm64

include ../containers_shared.mak

test: container
	bash -c test/containertest.sh

multi-arch:
	docker buildx build --platform $(BUILD_ARCHS) --progress=$(PROGRESS) -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) .; \
	echo "created multi-arch builds $(BUILD_ARCHS) for $(DOCKER_REPO)";
