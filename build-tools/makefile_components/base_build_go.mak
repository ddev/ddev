# Base Build portion of makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.


.PHONY: all build test push clean container-clean bin-clean version static vendorcheck gofmt govet golint

SHELL := /bin/bash

GOFILES = $(shell find $(SRC_DIRS) -name "*.go")

BUILD_IMAGE ?= drud/golang-build-container

BUILD_BASE_DIR ?= $$PWD

SRC_AND_UNDER = $(patsubst %,./%/...,$(SRC_DIRS))

build: linux darwin

linux darwin: $(GOFILES)
	@echo "building $@ from $(GOFILES)"
	@rm -f VERSION.txt
	@mkdir -p bin/$@
	@docker run                                                            \
	    -t                                                                \
	    -u root:root                                             \
	    -v $(BUILD_BASE_DIR)/build-tools:/build-tools		\
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/bin/$@:/go/bin                                     \
	    -v $$(pwd)/bin/$@:/go/bin/$@                      \
	    -v $$(pwd)/.go/std/$@:/usr/local/go/pkg/$@_amd64_static  \
	    -e GOOS=$@	\
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
	    /bin/sh -c "                                                       \
	        OS=$@                                                 \
	        VERSION=$(VERSION)                                             \
	        PKG=$(PKG)                                                     \
	        /build-tools/build-scripts/build_go.sh $(SRC_DIRS)                                              \
	    "
	@touch $@
	@echo $(VERSION) >VERSION.txt

static: vendorcheck gofmt govet lint

vendorcheck:
	echo -n "Checking vendorcheck for missing dependencies and unused dependencies: "
	@docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
                 	    -v $(BUILD_BASE_DIR)/build-tools:/build-tools		\
                 	    -v $$(pwd)/.go:/go                                                 \
                 	    -v $$(pwd):/go/src/$(PKG)                                          \
                 	    -w /go/src/$(PKG)                                                  \
                 	    $(BUILD_IMAGE)                                                     \
                 	    bash -c 'OUT=$$(vendorcheck ./... && govendor list +unused); if [ -n "$$OUT" ]; then echo "$$OUT"; exit 1; fi'

gofmt:
	echo -n "Checking gofmt: "
	@docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
                 	    -v $(BUILD_BASE_DIR)/build-tools:/build-tools		\
                 	    -v $$(pwd)/.go:/go                                                 \
                 	    -v $$(pwd):/go/src/$(PKG)                                          \
                 	    -w /go/src/$(PKG)                                                  \
                 	    $(BUILD_IMAGE)                                                     \
                 	    bash -c 'gofmt -l $(SRC_DIRS)'

govet:
	echo -n "Checking go vet: "
	docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
                 	    -v $(BUILD_BASE_DIR)/build-tools:/build-tools		\
                 	    -v $$(pwd)/.go:/go                                                 \
                 	    -v $$(pwd):/go/src/$(PKG)                                          \
                 	    -w /go/src/$(PKG)                                                  \
                 	    $(BUILD_IMAGE)                                                     \
                 	    bash -c 'go vet $(SRC_AND_UNDER)'

golint:
	echo -n "Checking golint: "
	docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
                 	    -v $(BUILD_BASE_DIR)/build-tools:/build-tools		\
                 	    -v $$(pwd)/.go:/go                                                 \
                 	    -v $$(pwd):/go/src/$(PKG)                                          \
                 	    -w /go/src/$(PKG)                                                  \
                 	    $(BUILD_IMAGE)                                                     \
                 	    bash -c 'golint $(SRC_AND_UNDER)'


version:
	@echo VERSION:$(VERSION)

clean: container-clean bin-clean

container-clean:
	rm -rf .container-* .dockerfile* .push-* linux darwin container VERSION.txt .docker_image

bin-clean:
	rm -rf .go bin .tmp
