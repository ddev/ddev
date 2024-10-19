DOCKER_ORG ?= ddev
export DOCKER_ORG

SHELL = /bin/bash

SANITIZED_DOCKER_REPO = $(subst /,_,$(DOCKER_REPO))

DOTFILE_IMAGE = $(subst /,_,$(IMAGE))-$(VERSION)

.PHONY: container push

container: container-name
	docker build -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) --label "build-info=$(DOCKER_REPO):$(VERSION) commit=$(shell git describe --tags --always)" .

container-name:
	@echo "container: $(DOCKER_REPO):$(VERSION)"

push:
	set -eu -o pipefail; \
	for item in $(DEFAULT_IMAGES); do \
		docker buildx use multi-arch-builder >/dev/null 2>&1 || docker buildx create --name multi-arch-builder --use; \
		tags="-t $(DOCKER_ORG)/$${item}:$(VERSION)"; \
		if [[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$$ ]] ; then \
			tags="$${tags} -t $(DOCKER_ORG)/$${item}:latest"; \
		fi; \
		docker buildx build --push --platform $(BUILD_ARCHS) \
			--target=$${item} \
			$${tags} \
			--label "build-info=$(DOCKER_ORG)/$${item}:$(VERSION) commit=$(shell git describe --tags --always) built $$(date) by $$(id -un) on $$(hostname)" \
			--label "maintainer=DDEV <support@ddev.com>" \
			$(DOCKER_ARGS) . ; \
	done
