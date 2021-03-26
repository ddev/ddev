DOCKER_ORG ?= drud
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-webserver

DEFAULT_IMAGES = ddev-webserver ddev-webserver-prod

VERSION := $(shell git describe --tags --always --dirty)

DOCKER_BUILDKIT=1

BUILD_ARCHS=linux/amd64,linux/arm64

include ../containers_shared.mak

container build: images

images: $(DEFAULT_IMAGES)

push:
	set -eu -o pipefail; \
	for item in $(DEFAULT_IMAGES); do \
		docker buildx build --push --platform $(BUILD_ARCHS) --label com.ddev.buildhost=${shell hostname} --target=$$item  -t $(DOCKER_ORG)/$$item:$(VERSION) $(DOCKER_ARGS) .; \
		echo "pushed $(DOCKER_ORG)/$$item"; \
	done

multi-arch:
	set -eu -o pipefail; \
	for item in $(DEFAULT_IMAGES); do \
		docker buildx build --platform $(BUILD_ARCHS) --label com.ddev.buildhost=${shell hostname} --target=$$item  -t $(DOCKER_ORG)/$$item:$(VERSION) $(DOCKER_ARGS) .; \
		echo "created multi-arch builds $(BUILD_ARCHS) for $(DOCKER_ORG)/$$item"; \
	done

ddev-webserver ddev-webserver-dev-base ddev-webserver-prod-base ddev-webserver-prod:
	docker buildx build -o type=docker --label com.ddev.buildhost=${shell hostname} --target=$@  -t $(DOCKER_ORG)/$@:$(VERSION) $(DOCKER_ARGS) .


test: images
	set -eu -o pipefail; \
	for item in $(DEFAULT_IMAGES); do \
		if [ -x tests/$$item/test.sh ]; then tests/$$item/test.sh $(DOCKER_ORG)/$$item:$(VERSION); fi; \
	done	
