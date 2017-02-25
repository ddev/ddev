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

# Expands SRC_DIRS into the common golang ./dir/... format for "all below"
SRC_AND_UNDER = $(patsubst %,./%/...,$(SRC_DIRS))


VERSION_VARIABLES += VERSION

VERSION_LDFLAGS := $(foreach v,$(VERSION_VARIABLES),-X $(PKG)/pkg/version.$(v)=$($(v)))

LDFLAGS := -extldflags -static $(VERSION_LDFLAGS)

build: linux darwin

linux darwin: $(GOFILES)
	@echo "building $@ from $(SRC_AND_UNDER)"
	@rm -f VERSION.txt
	@mkdir -p bin/$@
	@docker run                                                            \
	    -t                                                                \
	    -u root:root                                             \
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/bin/$@:/go/bin                                     \
	    -v $$(pwd)/bin/$@:/go/bin/$@                      \
	    -v $$(pwd)/.go/std/$@:/usr/local/go/pkg/$@_amd64_static  \
	    -e CGO_ENABLED=0                  \
	    -w /go/src/$(PKG)                 \
	    $(BUILD_IMAGE)                    \
	    /bin/sh -c '                      \
	        GOOS=$@                       \
	        go install -installsuffix 'static' -ldflags "$(LDFLAGS)" $(SRC_AND_UNDER)  \
	    '
	@touch $@
	@echo $(VERSION) >VERSION.txt

static: vendorcheck gofmt govet lint

vendorcheck:
	@echo -n "Checking vendorcheck for missing dependencies and unused dependencies: "
	@docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
                 	    -v $$(pwd)/.go:/go                                                 \
                 	    -v $$(pwd):/go/src/$(PKG)                                          \
                 	    -w /go/src/$(PKG)                                                  \
                 	    $(BUILD_IMAGE)                                                     \
                 	    bash -c 'OUT=$$(vendorcheck ./... && govendor list +unused); if [ -n "$$OUT" ]; then echo "$$OUT"; exit 1; fi'

gofmt:
	@echo -n "Checking gofmt: "
	docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
                 	    -v $$(pwd)/.go:/go                                                 \
                 	    -v $$(pwd):/go/src/$(PKG)                                          \
                 	    -w /go/src/$(PKG)                                                  \
                 	    $(BUILD_IMAGE)                                                     \
                 	    bash -c 'gofmt -l $(SRC_DIRS)'

govet:
	@echo -n "Checking go vet: "
	docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
                 	    -v $$(pwd)/.go:/go                                                 \
                 	    -v $$(pwd):/go/src/$(PKG)                                          \
                 	    -w /go/src/$(PKG)                                                  \
                 	    $(BUILD_IMAGE)                                                     \
                 	    bash -c 'go vet $(SRC_AND_UNDER)'

golint:
	@echo -n "Checking golint: "
	docker run                                                            \
                 	    -t                                                                \
                 	    -u root:root                                             \
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

# print-ANYVAR prints the expanded variable
print-%: ; @echo $* = $($*)
