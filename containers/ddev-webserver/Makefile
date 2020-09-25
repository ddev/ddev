# Makefile for a standard golang repo with associated container

##### These variables need to be adjusted in most repositories #####

# Docker repo for a push
DOCKER_ORG ?= drud
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-webserver

DEFAULT_IMAGES = ddev-webserver

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

# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

DOCKER_BUILDKIT=1

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS=linux/amd64,linux/arm64,linux/arm

container build: images

images: $(DEFAULT_IMAGES)

push:
	for item in $(DEFAULT_IMAGES); do \
		docker buildx build --push --platform $(BUILD_ARCHS) --progress=$(PROGRESS) --label com.ddev.buildhost=${shell hostname} --target=$$item  -t $(DOCKER_ORG)/$$item:$(VERSION) $(DOCKER_ARGS) .; \
		echo "pushed $(DOCKER_ORG)/$$item"; \
	done

multi-arch:
	for item in $(DEFAULT_IMAGES); do \
		docker buildx build --platform $(BUILD_ARCHS) --progress=$(PROGRESS) --label com.ddev.buildhost=${shell hostname} --target=$$item  -t $(DOCKER_ORG)/$$item:$(VERSION) $(DOCKER_ARGS) .; \
		echo "created multi-arch builds $(BUILD_ARCHS) for $(DOCKER_ORG)/$$item"; \
	done

ddev-webserver:
	docker buildx build --platform linux/amd64 -o type=docker --progress=$(PROGRESS) --label com.ddev.buildhost=${shell hostname} --target=$@  -t $(DOCKER_ORG)/$@:$(VERSION) $(DOCKER_ARGS) .

test: images
	for item in $(DEFAULT_IMAGES); do \
		if [ -x tests/$$item/test.sh ]; then tests/$$item/test.sh $(DOCKER_ORG)/$$item:$(VERSION); fi; \
	done	
