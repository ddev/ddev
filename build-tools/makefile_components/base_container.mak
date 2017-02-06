# Container section of standard makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.


DOTFILE_IMAGE = $(subst /,_,$(IMAGE))-$(VERSION)

container: linux .container-$(DOTFILE_IMAGE) container-name

.container-$(DOTFILE_IMAGE): linux Dockerfile.in
	@sed -e 's|UPSTREAM_REPO|$(UPSTREAM_REPO)|g' Dockerfile.in > .dockerfile
	docker build -t $(DOCKER_REPO):$(VERSION) $(DOCKER_ARGS) -f .dockerfile .
	@docker images -q $(DOCKER_REPO):$(VERSION) > $@
	@echo $(DOCKER_REPO):$(VERSION) >.docker_image

container-name:
	@echo "container: $(DOCKER_REPO):$(VERSION)"
