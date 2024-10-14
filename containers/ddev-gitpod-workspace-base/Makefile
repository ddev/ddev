# Docker repo for a push
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-gitpod-workspace-base

VERSION := $(shell git describe --tags --always --dirty)

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS=linux/amd64,linux/arm64

.PHONY: fetch-directory

build: fetch-directory container
push: fetch-directory

# Build a Dockerfile from the upstream dazzle-built workspace-images
TARGET_DIR :=./
REPO_URL :=https://github.com/gitpod-io/workspace-images.git
TMP_DIR :=/tmp/checkout-dir
GO_VERSION=1.23.2
fetch-directory:
	rm -rf $(TMP_DIR) && mkdir -p $(TMP_DIR)
	git clone $(REPO_URL) $(TMP_DIR)
	set -x
	cp $(TMP_DIR)/base/* .
	echo "ENV GO_VERSION=$(GO_VERSION)" >> $(TARGET_DIR)/Dockerfile
	for item in dep-cacert-update lang-go tool-docker; do grep -v 'FROM ${base}' $(TMP_DIR)/chunks/$$item/Dockerfile | awk '{ gsub(/amd64/, "$$(dpkg --print-architecture)"); print}' >> $(TARGET_DIR)/Dockerfile; done
	rm -rf $(TMP_DIR)
	sed -i.bak '1s|^FROM buildpack-deps:jammy@sha256:[^ ]*|FROM buildpack-deps:jammy|' $(TARGET_DIR)/Dockerfile && rm -f $(TARGET_DIR)/Dockerfile.bak

include ../containers_shared.mk

test: container
	bash -c "test/containertest.sh $(DOCKER_REPO):$(VERSION)"
