# Makefile for a standard golang repo with associated container

##### These variables need to be adjusted in most repositories #####

# Docker repo for a push
DOCKER_ORG ?= drud
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-webserver

DEFAULT_IMAGES = ddev-webserver

.PHONY: prep

# Top-level directories to build
#SRC_DIRS := filexs drudapi secrets utils

# Optional to docker build
# DOCKER_ARGS =

# VERSION can be set by
  # Default: git tag
  # make command line: make VERSION=0.9.0
# It can also be explicitly set in the Makefile as commented out below.

# This version-strategy uses git tags to set the version string
# VERSION can be overridden on make commandline: make VERSION=0.9.1 push
VERSION := $(shell git describe --tags --always --dirty)
#
# This version-strategy uses a manual value to set the version string
#VERSION := 1.2.3

# Each section of the Makefile is included from standard components below.
# If you need to override one, import its contents below and comment out the
# include. That way the base components can easily be updated as our general needs
# change.
#include ../build-tools/makefile_components/base_build_go.mak
#include ../../build-tools/makefile_components/base_build_python-docker.mak
#include ../../build-tools/makefile_components/base_container.mak
#include ../../build-tools/makefile_components/base_push.mak
#include build-tools/makefile_components/base_test_go.mak
#include build-tools/makefile_components/base_test_python.mak

# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

DOCKER_BUILDKIT=1

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS=linux/amd64,linux/arm64

container build: images

images: $(DEFAULT_IMAGES)

push: prep
	for item in $(DEFAULT_IMAGES); do \
		docker buildx build --push --platform $(BUILD_ARCHS) --progress=$(PROGRESS) --label com.ddev.buildhost=${shell hostname} --target=$$item  -t $(DOCKER_ORG)/$$item:$(VERSION) $(DOCKER_ARGS) .; \
		echo "pushed $(DOCKER_ORG)/$$item"; \
	done

multi-arch: prep
	for item in $(DEFAULT_IMAGES); do \
		docker buildx build --platform $(BUILD_ARCHS) --progress=$(PROGRESS) --label com.ddev.buildhost=${shell hostname} --target=$$item  -t $(DOCKER_ORG)/$$item:$(VERSION) $(DOCKER_ARGS) .; \
		echo "created multi-arch builds $(BUILD_ARCHS) for $(DOCKER_ORG)/$$item"; \
	done

ddev-webserver: prep
	docker buildx build --platform linux/amd64 -o type=docker --progress=$(PROGRESS) --label com.ddev.buildhost=${shell hostname} --target=$@  -t $(DOCKER_ORG)/$@:$(VERSION) $(DOCKER_ARGS) .

test: images
	for item in $(DEFAULT_IMAGES); do \
		if [ -x tests/$$item/test.sh ]; then tests/$$item/test.sh $(DOCKER_ORG)/$$item:$(VERSION); fi; \
	done

prep:
	# We need this to get arm64 qemu to work https://github.com/docker/buildx/issues/138#issuecomment-569240559
	docker run --rm --privileged docker/binfmt:66f9012c56a8316f9244ffd7622d7c21c1f6f28d
	if ! docker buildx inspect ddev-builder-multi --bootstrap >/dev/null; then docker buildx create --name ddev-builder-multi; fi
	docker buildx use ddev-builder-multi
