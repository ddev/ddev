# Base Build portion of makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.

# The //workdir prevents docker and friends from trying to convert the thing to a non-unix path.
# The $(S) in fromt of paths is a slash used only for Docker Toolbox, for the same reason.
DOCKERBUILDCMD=docker run -t --rm -u $(shell id -u):$(shell id -g)                    \
          	    -v "$(S)$(PWD):/workdir$(DOCKERMOUNTFLAG)"                              \
          	    -v "$(S)$(PWD)/$(GOTMP)/bin:$(S)/go/bin" \
          	    -e CGO_ENABLED=0                  \
          	    -e GOOS=$@						  \
          	    -e GOPATH="//workdir/$(GOTMP)" \
          	    -e GOCACHE="//workdir/$(GOTMP)/.cache" \
          	    -e GOFLAGS="$(USEMODVENDOR)" \
          	    -w //workdir              \
          	    $(BUILD_IMAGE)

DOCKERTESTCMD=docker run -t --rm -u $(shell id -u):$(shell id -g)                    \
          	    -v "$(S)$(PWD):/workdir$(DOCKERMOUNTFLAG)"                              \
          	    -e GOPATH="//workdir/$(GOTMP)" \
          	    -e GOCACHE="//workdir/$(GOTMP)/.cache" \
          	    -e GOLANGCI_LINT_CACHE="//workdir/$(GOTMP)/.golanci-lint-cache" \
          	    -e GOFLAGS="$(USEMODVENDOR)" \
          	    -w //workdir              \
          	    $(BUILD_IMAGE)

.PHONY: all build test push clean container-clean bin-clean version static gofmt govet golint golangci-lint container pull
GOTMP=.gotmp

SHELL = /bin/bash

GOFILES = $(shell find $(SRC_DIRS) -name "*.go")

BUILD_OS = $(shell go env GOHOSTOS)

BUILD_IMAGE ?= drud/golang-build-container:v1.14.2

BUILD_BASE_DIR ?= $(PWD)

# Expands SRC_DIRS into the common golang ./dir/... format for "all below"
SRC_AND_UNDER = $(patsubst %,./%/...,$(SRC_DIRS))

GOMETALINTER_ARGS ?= --vendored-linters --disable-all --enable=gofmt --enable=vet --enable=vetshadow --enable=golint --enable=errcheck --enable=staticcheck --enable=ineffassign --enable=varcheck --enable=deadcode --deadline=2m

GOLANGCI_LINT_ARGS ?= --out-format=line-number --disable-all --enable=gofmt --enable=govet --enable=golint --enable=errcheck --enable=staticcheck --enable=ineffassign --enable=varcheck --enable=deadcode

COMMIT := $(shell git describe --tags --always --dirty)
BUILDINFO = $(shell echo Built $$(date) $(BUILD_IMAGE) )

VERSION_VARIABLES += VERSION COMMIT BUILDINFO

VERSION_LDFLAGS := $(foreach v,$(VERSION_VARIABLES),-X "$(PKG)/pkg/version.$(v)=$($(v))")

LDFLAGS := -extldflags -static $(VERSION_LDFLAGS)
DOCKERMOUNTFLAG := :delegated

# In go 1.11 -mod=vendor is not autodetected; it probably will be in 1.12
# See https://github.com/golang/go/issues/27227
USEMODVENDOR := $(shell if [ -d vendor ]; then echo "-mod=vendor"; fi)

ifeq ($(BUILD_OS),windows)
    # On Windows docker toolbox, volume mounts oddly need a // at the beginning for things to work out, so
    # add that extra slash only on Windows.
    DOCKERMOUNTFLAG=
endif

# On Docker Toolbox we can't use paths like C:\xxx\xxx, must use /C/xxx/xxx
# However, on Docker-for-Windows/Appveyor we must use C:\xxx
# On all other (macos/linux) $PWD will already be OK.
ifneq ($(shell if [ "$(BUILD_OS)" = "windows" ] && [ -z "$(DOCKER_TOOLBOX_INSTALL_PATH)" ]; then echo true; fi),)
	PWD=$(shell cmd //c "echo %cd%")
endif
ifneq ($(DOCKER_TOOLBOX_INSTALL_PATH),)
	PWD=$(shell pwd)
    S=/
endif

build: $(BUILD_OS)

pull:
	@if [[ "$(docker images -q $(BUILD_IMAGE)  2> /dev/null)" == "" ]]; then docker pull $(BUILD_IMAGE) >/dev/null 2>&1; fi


linux darwin windows: pull $(GOFILES)
	@echo "building $@ from $(SRC_AND_UNDER)"
	@echo $(shell if [ "$(BUILD_OS)" = "windows" ]; then echo "windows build: BUILD_OS=$(BUILD_OS)  DOCKER_TOOLBOX_INSTALL_PATH=$(DOCKER_TOOLBOX_INSTALL_PATH) PWD=$(PWD) S=$(S)"; fi )
	@mkdir -p $(GOTMP)/{.cache,pkg,src,bin}
	@$(DOCKERBUILDCMD) \
        go install -installsuffix static -ldflags ' $(LDFLAGS) ' $(SRC_AND_UNDER) && touch $@
	$( shell if [ -d $(GOTMP) ]; then chmod -R u+w $(GOTMP); fi )
	@echo $(VERSION) >VERSION.txt

gofmt:
	@echo "Checking gofmt: "
	@$(DOCKERTESTCMD) \
		bash -c 'export OUT=$$(gofmt -l $(SRC_DIRS))  && if [ -n "$$OUT" ]; then echo "These files need gofmt -w: $$OUT"; exit 1; fi'

govet:
	@echo "Checking go vet: "
	$(DOCKERTESTCMD) \
		bash -c 'go vet $(SRC_AND_UNDER)'

golint:
	@echo "Checking golint: "
	@$(DOCKERTESTCMD) \
		bash -c 'export OUT=$$(golint $(SRC_AND_UNDER)) && if [ -n "$$OUT" ]; then echo "Golint problems discovered: $$OUT"; exit 1; fi'

errcheck:
	@echo "Checking errcheck: "
	@$(DOCKERTESTCMD) \
		errcheck $(SRC_AND_UNDER)

staticcheck:
	@echo "Checking staticcheck: "
	@$(DOCKERTESTCMD) \
		staticcheck $(SRC_AND_UNDER)

varcheck:
	@echo "Checking unused globals and struct members: "
	@$(DOCKERTESTCMD) \
		bash -c "varcheck $(SRC_AND_UNDER) && structcheck $(SRC_AND_UNDER)"

misspell:
	@echo "Checking for misspellings: "
	@$(DOCKERTESTCMD) \
		misspell $(SRC_DIRS)

gometalinter:
	@echo "gometalinter: "
	@$(DOCKERTESTCMD) \
		time gometalinter $(GOMETALINTER_ARGS) $(SRC_AND_UNDER)

golangci-lint:
	@echo "golangci-lint: "
	@$(DOCKERTESTCMD) \
		time bash -c "golangci-lint run $(GOLANGCI_LINT_ARGS) $(SRC_AND_UNDER)"

version:
	@echo VERSION:$(VERSION)

clean: container-clean bin-clean

container-clean:
	@if docker image inspect $(DOCKER_REPO):$(VERSION) >/dev/null 2>&1; then docker rmi -f $(DOCKER_REPO):$(VERSION); fi
	@rm -rf .container-* .dockerfile* .push-* linux darwin windows container VERSION.txt .docker_image

bin-clean:
	@rm -rf bin
	$(shell if [ -d $(GOTMP) ]; then chmod -R u+w $(GOTMP) && rm -rf $(GOTMP); fi )

# print-ANYVAR prints the expanded variable
print-%: ; @echo $* = $($*)
