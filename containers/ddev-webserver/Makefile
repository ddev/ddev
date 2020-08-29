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

# Each section of the Makefile is included from standard components below.
# If you need to override one, import its contents below and comment out the
# include. That way the base components can easily be updated as our general needs
# change.
#include ../build-tools/makefile_components/base_build_go.mak
#include ../../build-tools/makefile_components/base_build_python-docker.mak
include ../../build-tools/makefile_components/base_container.mak
include ../../build-tools/makefile_components/base_container_arm64.mak
#include ../../build-tools/makefile_components/base_push.mak
#include build-tools/makefile_components/base_test_go.mak
#include build-tools/makefile_components/base_test_python.mak

# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

DOCKER_BUILDKIT=1

container build: images

images: $(DEFAULT_IMAGES)

push: images
	for item in $(DEFAULT_IMAGES); do \
		docker push $(DOCKER_ORG)/$$item:$(VERSION); \
		echo "pushed $(DOCKER_ORG)/$$item"; \
	done

ddev-webserver:
	DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker buildx build --progress=$(PROGRESS) --platform linux/amd64,linux/arm64 --label com.ddev.buildhost=${shell hostname} --target=$@  -t $(DOCKER_ORG)/$@:$(VERSION) $(DOCKER_ARGS) .

test: images
	for item in $(DEFAULT_IMAGES); do \
		if [ -x tests/$$item/test.sh ]; then tests/$$item/test.sh $(DOCKER_ORG)/$$item:$(VERSION); fi; \
	done
