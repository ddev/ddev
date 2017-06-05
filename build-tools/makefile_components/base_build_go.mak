# Base Build portion of makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.


.PHONY: all build test push clean container-clean bin-clean version static govendor gofmt govet golint
GOTMP=.gotmp

SHELL := /bin/bash

GOFILES = $(shell find $(SRC_DIRS) -name "*.go")

BUILD_IMAGE ?= drud/golang-build-container:v0.4.2

BUILD_BASE_DIR ?= $$PWD

# Expands SRC_DIRS into the common golang ./dir/... format for "all below"
SRC_AND_UNDER = $(patsubst %,./%/...,$(SRC_DIRS))

GOMETALINTER_ARGS ?= --vendored-linters --disable=gocyclo --disable=gotype --disable=goconst --disable=gas --deadline=2m


COMMIT := $(shell git describe --tags --always --dirty)
BUILDINFO = $(shell echo Built $$(date) $$USER@$$(hostname) $(BUILD_IMAGE) )

VERSION_VARIABLES += VERSION COMMIT BUILDINFO

VERSION_LDFLAGS := $(foreach v,$(VERSION_VARIABLES),-X "$(PKG)/pkg/version.$(v)=$($(v))")

LDFLAGS := -extldflags -static $(VERSION_LDFLAGS)

build: linux darwin

linux darwin windows: $(GOFILES)
	@echo "building $@ from $(SRC_AND_UNDER)"
	@rm -f VERSION.txt
	@mkdir -p bin/$@ $(GOTMP)/{std/$@,bin,src/$(PKG)}

	@docker run -t --rm -u $(shell id -u):$(shell id -g)                    \
	    -v $$(pwd)/$(GOTMP):/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/bin/$@:/go/bin                                     \
	    -v $$(pwd)/bin/$@:/go/bin/$@                      \
	    -v $$(pwd)/$(GOTMP)/std/$@:/usr/local/go/pkg/$@_amd64_static  \
	    -e CGO_ENABLED=0                  \
	    -e GOOS=$@						  \
	    -w /go/src/$(PKG)                 \
	    $(BUILD_IMAGE)                    \
        go install -installsuffix static -ldflags ' $(LDFLAGS) ' $(SRC_AND_UNDER)
	@touch $@
	@echo $(VERSION) >VERSION.txt

static: govendor gofmt govet lint

govendor:
	@echo -n "Using govendor to check for missing dependencies and unused dependencies: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                    \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'OUT=$$(govendor list +missing +unused); if [ -n "$$OUT" ]; then echo "$$OUT"; exit 1; fi'

gofmt:
	@echo "Checking gofmt: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                    \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'export OUT=$$(gofmt -l $(SRC_DIRS))  && if [ -n "$$OUT" ]; then echo "These files need gofmt -w: $$OUT"; exit 1; fi'

govet:
	@echo "Checking go vet: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                         \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'go vet $(SRC_AND_UNDER)'

golint:
	@echo "Checking golint: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                   \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		bash -c 'export OUT=$$(golint $(SRC_AND_UNDER)) && if [ -n "$$OUT" ]; then echo "Golint problems discovered: $$OUT"; exit 1; fi'

errcheck:
	@echo "Checking errcheck: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                   \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		errcheck $(SRC_AND_UNDER)

staticcheck:
	@echo "Checking staticcheck: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                         \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		staticcheck $(SRC_AND_UNDER)

unused:
	@echo "Checking unused variables and functions: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                         \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		unused $(SRC_AND_UNDER)

codecoroner:
	@echo "Checking codecoroner for unused functions: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                         \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE) \
		bash -c 'OUT=$$(codecoroner -tests -ignore vendor funcs $(SRC_AND_UNDER)); if [ -n "$$OUT" ]; then echo "$$OUT"; exit 1; fi'                                             \


varcheck:
	@echo "Checking unused globals and struct members: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                         \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		varcheck $(SRC_AND_UNDER) && structcheck $(SRC_AND_UNDER)

misspell:
	@echo "Checking for misspellings: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                         \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		misspell $(SRC_DIRS)

gometalinter:
	@echo "gometalinter: "
	@docker run -t --rm -u $(shell id -u):$(shell id -g)                         \
		-v $$(pwd)/$(GOTMP):/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-w /go/src/$(PKG)                                                  \
		$(BUILD_IMAGE)                                                     \
		gometalinter $(GOMETALINTER_ARGS) $(SRC_AND_UNDER)

version:
	@echo VERSION:$(VERSION)

clean: container-clean bin-clean

container-clean:
	rm -rf .container-* .dockerfile* .push-* linux darwin windows container VERSION.txt .docker_image

bin-clean:
	rm -rf $(GOTMP) bin .tmp

# print-ANYVAR prints the expanded variable
print-%: ; @echo $* = $($*)
