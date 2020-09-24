# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

container:
	docker buildx build --platform linux/amd64 --progress=$(PROGRESS) -o type=docker -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) .

push:
	docker buildx build --push --platform $(BUILD_ARCHS) --progress=$(PROGRESS) -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) .
