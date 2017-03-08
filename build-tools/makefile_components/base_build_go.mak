# Base Build portion of makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.


.PHONY: all build test push clean container-clean bin-clean version static govendor gofmt govet golint

SHELL := /bin/bash

GOFILES = $(shell find $(SRC_DIRS) -name "*.go")

BUILD_IMAGE ?= drud/golang-build-container:0.1.0

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
	@mkdir -p bin/$@ .go/std/$@ .go/bin .go/src/$(PKG)
	docker run                                                            \
	    -t                                                                \
	    -u $(shell id -u):$(shell id -g)                                             \
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

static: govendor gofmt govet lint

govendor:
	@echo -n "Using govendor to check for missing dependencies and unused dependencies: "
	docker run                                                            \
		-t                                                                \
	    -u $(shell id -u):$(shell id -g)                                             \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'OUT=$$(govendor list +missing +unused); if [ -n "$$OUT" ]; then echo "$$OUT"; exit 1; fi'

gofmt:
	@echo "Checking gofmt: "
	docker run                                                            \
		-t                                                                \
	    -u $(shell id -u):$(shell id -g)                                             \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'export OUT=$$(gofmt -l $(SRC_DIRS))  && if [ -n "$$OUT" ]; then echo "These files need gofmt -w: $$OUT"; exit 1; fi'

govet:
	@echo -n "Checking go vet: "
	docker run                                                            \
		-t                                                                \
		-u $(shell id -u):$(shell id -g)                                             \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'go vet $(SRC_AND_UNDER)'

golint:
	@echo -n "Checking golint: "
	docker run                                                            \
		-t                                                                \
	    -u $(shell id -u):$(shell id -g)                                             \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'export OUT=$$(golint $(SRC_AND_UNDER)) && if [ -n "$$OUT" ]; then echo "Golint problems discovered: $$OUT"; exit 1; fi'


version:
	@echo VERSION:$(VERSION)

clean: container-clean bin-clean

container-clean:
	rm -rf .container-* .dockerfile* .push-* linux darwin container VERSION.txt .docker_image

bin-clean:
	rm -rf .go bin .tmp

# print-ANYVAR prints the expanded variable
print-%: ; @echo $* = $($*)
