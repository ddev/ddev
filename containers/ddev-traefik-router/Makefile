VERSION := $(shell git describe --tags --always --dirty)
DEFAULT_IMAGES = ddev-traefik-router

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS?=linux/amd64,linux/arm64

include ../containers_shared.mk

DOCKER_REPO ?= $(DOCKER_ORG)/ddev-traefik-router

test: container
	bash -c "test/containertest.sh $(DOCKER_REPO):$(VERSION)"

multi-arch:
	docker buildx build --platform $(BUILD_ARCHS) -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) .; \
	echo "created multi-arch builds $(BUILD_ARCHS) for $(DOCKER_REPO)";
