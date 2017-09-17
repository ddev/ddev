# Push section of Makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.

push: .push-$(DOTFILE_IMAGE) push-name
.push-$(DOTFILE_IMAGE): .container-$(DOTFILE_IMAGE)
	@gcloud docker -- push $(DOCKER_REPO):$(VERSION)
	@docker images -q $(DOCKER_REPO):$(VERSION) > $@

push-name:
	@echo "pushed: $(DOCKER_REPO):$(VERSION)"
