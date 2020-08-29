# Container section of standard makefile (arm64)
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.

SANITIZED_DOCKER_REPO = $(subst /,_,$(DOCKER_REPO))

DOTFILE_IMAGE = $(subst /,_,$(IMAGE))-$(VERSION)

container-arm64: .container-arm64-$(DOTFILE_IMAGE) container-arm64-name

# In CI environments, use the plain Docker build progress to not overload the CI logs
PROGRESS := $(if $(CI),plain,auto)

.container-arm64-$(DOTFILE_IMAGE): $(wildcard Dockerfile Dockerfile.in) container-arm64-name
    # UPSTREAM_REPO in the Dockerfile.in will be changed to the value from Makefile; this is deprecated.
    # There's no reason not to just use Dockerfile now.
	@if [ -f Dockerfile.in ]; then sed -e 's|UPSTREAM_REPO|$(UPSTREAM_REPO)|g' Dockerfile.in > .dockerfile; else cp Dockerfile .dockerfile; fi
	# Add information about the commit into .docker_image, to be added to the build.
	@echo "$(DOCKER_REPO):$(VERSION) commit=$(shell git describe --tags --always)"  >.docker_image
	# Add the .docker_image into the build so it's easy to figure out where a docker image came from.
	@echo "ADD .docker_image /$(SANITIZED_DOCKER_REPO)_VERSION_INFO.txt" >>.dockerfile
	docker buildx build --progress=$(PROGRESS) --platform linux/arm64 -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) -f .dockerfile .
	@docker images -q $(DOCKER_REPO):$(VERSION) >$@

container-arm64-name:
	@echo "container: $(DOCKER_REPO):$(VERSION)"
