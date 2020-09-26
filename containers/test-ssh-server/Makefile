# Makefile for a standard repo with associated container

##### These variables need to be adjusted in most repositories #####

# This repo's root import path (under GOPATH).
# PKG := github.com/drud/repo_name

# Docker repo for a push
DOCKER_REPO ?= drud/test-ssh-server

VERSION := $(shell git describe --tags --always --dirty)

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
# arm64 doesn't make sense here (yet) since gotechnies/alpine-ssh is amd64-only
BUILD_ARCHS=linux/amd64,linux/arm64

include ../containers_shared.mak

# Additional targets can be added here
# Also, existing targets can be overridden by copying and customizing them.
multi-arch:
	docker buildx build --platform $(BUILD_ARCHS) -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) .; \
	echo "created multi-arch builds $(BUILD_ARCHS) for $(DOCKER_REPO)";

test: container
	true
