# Docker repo for a push
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-gitpod-workspace-base

VERSION := $(shell git describe --tags --always --dirty)

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
BUILD_ARCHS=linux/amd64,linux/arm64

TARGET_DIR := ./base-dir # Set the target directory where you want the contents
REPO_URL := https://github.com/gitpod-io/workspace-images.git # Repository URL
DIR_TO_CHECKOUT := base # Directory to checkout
TMP_DIR := /tmp/checkout-dir # Temporary directory for sparse checkout

.PHONY: fetch-directory
fetch-directory:
	mkdir -p $(TMP_DIR)
	git clone --no-checkout $(REPO_URL) $(TMP_DIR)
	cd $(TMP_DIR) && git config core.sparseCheckout true
	echo "$(DIR_TO_CHECKOUT)" >> $(TMP_DIR)/.git/info/sparse-checkout
	cd $(TMP_DIR) && git checkout
	rsync -av --progress $(TMP_DIR)/$(DIR_TO_CHECKOUT)/ $(TARGET_DIR)/
	rm -rf $(TMP_DIR)

include ../containers_shared.mk

container: fetch-directory
push: fetch-directory

test: container
	bash -c "test/containertest.sh $(DOCKER_REPO):$(VERSION)"
