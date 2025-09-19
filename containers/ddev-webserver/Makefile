
DEFAULT_IMAGES = ddev-webserver ddev-webserver-prod

VERSION := $(shell git describe --tags --always --dirty)

DOCKER_BUILDKIT=1

BUILD_ARCHS?=linux/amd64,linux/arm64

build: images

include ../containers_shared.mk

DOCKER_REPO ?= $(DOCKER_ORG)/ddev-webserver

images: $(DEFAULT_IMAGES)

$(DEFAULT_IMAGES):
	 docker build --label com.ddev.buildhost=${shell hostname} --target=$@  -t $(DOCKER_ORG)/$@:$(VERSION) $(DOCKER_ARGS) .


test: images
	set -eu -o pipefail; \
	for item in $(DEFAULT_IMAGES); do \
		if [ -x tests/$$item/test.sh ]; then tests/$$item/test.sh $(DOCKER_ORG)/$$item:$(VERSION); fi; \
	done
