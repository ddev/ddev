# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

SANITIZED_DOCKER_REPO = $(subst /,_,$(DOCKER_REPO))

DOTFILE_IMAGE = $(subst /,_,$(IMAGE))-$(VERSION)

container: container-name
	docker buildx build --platform linux/amd64 --progress=$(PROGRESS) -o type=docker -t $(DOCKER_REPO):$(VERSION) -t $(DOCKER_ARGS) --label "build-info=$(DOCKER_REPO):$(VERSION) commit=$(shell git describe --tags --always)" .

container-name:
	@echo "container: $(DOCKER_REPO):$(VERSION)"

push:
	docker buildx build --push --platform $(BUILD_ARCHS) --progress=$(PROGRESS) -t $(DOCKER_REPO):$(VERSION) --label "build-info=$(DOCKER_REPO):$(VERSION) commit=$(shell git describe --tags --always) built $$(date) by $$(id -un) on $$(hostname)" --label "maintainer=DDEV <rfay@ddev.com>" $(DOCKER_ARGS) .
