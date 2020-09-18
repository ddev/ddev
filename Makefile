# Makefile for a standard golang repo with associated container

# Circleci doesn't seem to provide a decent way to add to path, just adding here, for case where
# linux build and linuxbrew is installed.
export PATH := $(EXTRA_PATH):$(PATH)

DOCKERMOUNTFLAG := :cached

BUILD_BASE_DIR ?= $(PWD)

GOTMP=.gotmp
SHELL = /bin/bash
GOFILES = $(shell find $(SRC_DIRS) -name "*.go")

# Expands SRC_DIRS into the common golang ./dir/... format for "all below"
SRC_AND_UNDER = $(patsubst %,./%/...,$(SRC_DIRS))

GOLANGCI_LINT_ARGS ?= --out-format=line-number --disable-all --enable=gofmt --enable=govet --enable=golint --enable=errcheck --enable=staticcheck --enable=ineffassign --enable=varcheck --enable=deadcode

WINDOWS_SUDO_VERSION=v0.0.1
WINNFSD_VERSION=2.4.0
NSSM_VERSION=2.24-101-g897c7ad
MKCERT_VERSION=v1.4.1

GOTESTSUM_FORMAT ?= short-verbose
TESTTMP=/tmp/testresults
DOWNLOADTMP=$(HOME)/tmp

TESTTOOL ?= $(shell if command -v gotestsum >/dev/null ; then echo "gotestsum --format $(GOTESTSUM_FORMAT) --junitfile '$(TESTTMP)/$(@).xml'  --"; else echo "go test"; fi)
##### These variables need to be adjusted in most repositories #####

# This repo's root import path (under GOPATH).
PKG := github.com/drud/ddev

# Docker repo for a push
#DOCKER_REPO ?= drud/drupal-deploy

# Upstream repo used in the Dockerfile
#UPSTREAM_REPO ?= drud/site-deploy:latest

# Top-level directories to build
SRC_DIRS := cmd pkg

# Version variables to replace in build
VERSION_VARIABLES ?= DdevVersion SegmentKey

# These variables will be used as the default unless overridden by the make
DdevVersion ?= $(VERSION)
# WebTag ?= $(VERSION)  # WebTag is normally specified in version.go, sometimes overridden (night-build.mak)
# DBTag ?=  $(VERSION)  # DBTag is normally specified in version.go, sometimes overridden (night-build.mak)
# RouterTag ?= $(VERSION) #RouterTag is normally specified in version.go, sometimes overridden (night-build.mak)
# DBATag ?= $(VERSION) #DBATag is normally specified in version.go, sometimes overridden (night-build.mak)

# Optional to docker build
#DOCKER_ARGS =

# VERSION can be set by
  # Default: git tag
  # make command line: make VERSION=0.9.0
# It can also be explicitly set in the Makefile as commented out below.

# This version-strategy uses git tags to set the version string
# VERSION can be overridden on make commandline: make VERSION=0.9.1 push
VERSION := $(shell git describe --tags --always --dirty)
# Some things insist on having the version without the leading 'v', so provide a
# $(NO_V_VERSION) without it.
# no_v_version removes the front v, for Chocolatey mostly
NO_V_VERSION=$(shell echo $(VERSION) | awk  -F"-" '{ OFS="-"; sub(/^./, "", $$1); printf $$0; }')
GITHUB_ORG := drud

BUILD_OS = $(shell go env GOHOSTOS)
BUILD_ARCH = $(shell go env GOHOSTARCH)
VERSION_LDFLAGS=$(foreach v,$(VERSION_VARIABLES),-X '$(PKG)/pkg/version.$(v)=$($(v))')
LDFLAGS=-extldflags -static $(VERSION_LDFLAGS)
BUILD_IMAGE ?= drud/golang-build-container:v1.15.2
DOCKERBUILDCMD=docker run -t --rm -u $(shell id -u):$(shell id -g)                    \
          	    -v "$(PWD):/workdir$(DOCKERMOUNTFLAG)"                              \
          	    -v "$(PWD)/$(GOTMP)/bin:/go/bin" \
          	    -e GOPATH="/workdir/$(GOTMP)" \
          	    -e GOCACHE="/workdir/$(GOTMP)/.cache" \
          	    -e GOFLAGS="$(USEMODVENDOR)" \
          	    -w /workdir              \
          	    $(BUILD_IMAGE)
DEFAULT_BUILD=$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)

build: $(DEFAULT_BUILD)

pullbuildimage:
	docker pull $(BUILD_IMAGE) || true

linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64 windows_arm: pullbuildimage $(GOFILES)
	@echo "building $@ from $(SRC_AND_UNDER)"
	@echo $(shell if [ "$(BUILD_OS)" = "windows" ]; then echo "windows build: BUILD_OS=$(BUILD_OS)  DOCKER_TOOLBOX_INSTALL_PATH=$(DOCKER_TOOLBOX_INSTALL_PATH) PWD=$(PWD) S="; fi )
	@mkdir -p $(GOTMP)/{.cache,pkg,src,bin}
	@echo "LDFLAGS=$(LDFLAGS)"
	$(DOCKERBUILDCMD) \
        bash -c "GOOS=$(word 1, $(subst _, ,$@)) GOARCH=$(word 2, $(subst _, ,$@)) go install -installsuffix static -ldflags \" $(LDFLAGS) \" $(SRC_AND_UNDER) && touch $@"
	$( shell if [ -d $(GOTMP) ]; then chmod -R u+w $(GOTMP); fi )
	@echo $(VERSION) >VERSION.txt


TEST_TIMEOUT=150m
BUILD_ARCH = $(shell go env GOARCH)
ifeq ($(BUILD_OS),linux)
    DDEV_BINARY_FULLPATH=$(PWD)/$(GOTMP)/bin/ddev
endif

ifeq ($(BUILD_OS),windows_amd64)
    DDEV_BINARY_FULLPATH=$(shell pwd)/$(GOTMP)/bin/$(BUILD_OS)_$(BUILD_ARCH)/ddev.exe
endif

ifeq ($(BUILD_OS),darwin_amd64)
    DDEV_BINARY_FULLPATH=$(PWD)/$(GOTMP)/bin/$(BUILD_OS)_$(BUILD_ARCH)/ddev
endif

# Override test section with tests specific to ddev
test: testpkg testcmd

testcmd: $(DEFAULT_BUILD) setup
	echo LDFLAGS=$(LDFLAGS)
	DDEV_NO_INSTRUMENTATION=true CGO_ENABLED=0 DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH) go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags " $(LDFLAGS) " ./cmd/... $(TESTARGS)

testpkg: $(DEFAULT_BUILD) setup
	echo LDFLAGS=$(LDFLAGS)
	DDEV_NO_INSTRUMENTATION=true CGO_ENABLED=0 DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH) go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags " $(LDFLAGS) " ./pkg/... $(TESTARGS)

setup:
	@mkdir -p $(GOTMP)/{src,pkg/mod/cache,.cache}
	@mkdir -p $(TESTTMP)
	@mkdir -p $(DOWNLOADTMP)

# packr2 target currently builds packr2 caches in cmd/ddev/cmd and pkg/ddevapp
# using the golang-build-container's packr2 command
packr2:
	docker run -t --rm -u $(shell id -u):$(shell id -g)    \
          	    -v "$(S)$(PWD):/workdir$(DOCKERMOUNTFLAG)"  \
          	    -v "$(S)$(PWD)/$(GOTMP)/bin:$(S)/go/bin" \
          	    -e GOCACHE="//workdir/$(GOTMP)/.cache" \
          	    -w //workdir/cmd/ddev/cmd       \
          	    $(BUILD_IMAGE) packr2
	docker run -t --rm -u $(shell id -u):$(shell id -g)    \
          	    -v "$(S)$(PWD):/workdir$(DOCKERMOUNTFLAG)"  \
          	    -v "$(S)$(PWD)/$(GOTMP)/bin:$(S)/go/bin" \
          	    -e GOCACHE="//workdir/$(GOTMP)/.cache" \
          	    -w //workdir/pkg/ddevapp       \
          	    $(BUILD_IMAGE) packr2

# Required static analysis targets used in circleci - these cause fail if they don't work
staticrequired: setup golangci-lint markdownlint mkdocs

markdownlint:
	@echo "markdownlint: "
	@sleep 1 && $(DOCKERTESTCMD) \
		bash -c "markdownlint *.md docs 2>&1"
mkdocs:
	@echo "mkdocs: "
	@sleep 1 && $(DOCKERTESTCMD) \
		bash -c "mkdocs build -d /tmp/mkdocsbuld >/dev/null 2>&1"

darwin_signed: darwin_amd64
	@if [ -z "$(DDEV_MACOS_SIGNING_PASSWORD)" ] ; then echo "Skipping signing ddev for macOS, no DDEV_MACOS_SIGNING_PASSWORD provided"; else echo "Signing macOS ddev..."; \
		set -o errexit -o pipefail; \
		curl -s https://raw.githubusercontent.com/drud/signing_tools/master/macos_sign.sh | bash -s -  --signing-password="$(DDEV_MACOS_SIGNING_PASSWORD)" --cert-file=certfiles/ddev_developer_id_cert.p12 --cert-name="Developer ID Application: DRUD Technology, LLC (3BAN66AG5M)" --target-binary="$(GOTMP)/bin/darwin_amd64/ddev" ; \
	fi

darwin_notarized: darwin_signed
	@if [ -z "$(DDEV_MACOS_APP_PASSWORD)" ]; then echo "Skipping notarizing ddev for macOS, no DDEV_MACOS_APP_PASSWORD provided"; else \
		set -o errexit -o pipefail; \
		echo "Notarizing macOS ddev..." ; \
		curl -s https://raw.githubusercontent.com/drud/signing_tools/master/macos_notarize.sh | bash -s -  --app-specific-password=${DDEV_MACOS_APP_PASSWORD} --apple-id=accounts@drud.com --primary-bundle-id=com.ddev.ddev --target-binary="$(PWD)/$(GOTMP)/bin/darwin_amd64/ddev" ; \
	fi

$(GOTMP)/bin/windows_amd64/ddev.exe: windows_amd64

windows_install: $(GOTMP)/bin/windows_amd64/ddev_windows_installer.$(VERSION).exe

$(GOTMP)/bin/windows_amd64/ddev_windows_installer.$(VERSION).exe:  $(GOTMP)/bin/windows_amd64/ddev.exe $(GOTMP)/bin/windows_amd64/sudo.exe $(GOTMP)/bin/windows_amd64/sudo_license.txt $(GOTMP)/bin/windows_amd64/nssm.exe $(GOTMP)/bin/windows_amd64/winnfsd.exe $(GOTMP)/bin/windows_amd64/winnfsd_license.txt $(GOTMP)/bin/windows_amd64/mkcert.exe $(GOTMP)/bin/windows_amd64/mkcert_license.txt winpkg/ddev.nsi
	@if [ -z "$(DDEV_WINDOWS_SIGNING_PASSWORD)" ] ; then echo "Skipping signing ddev.exe, no DDEV_WINDOWS_SIGNING_PASSWORD provided"; else echo "Signing windows ddev.exe..."&& osslsigncode sign -pkcs12 certfiles/drud_cs.p12  -n "DDEV-Local Binary" -i https://ddev.com -in $< -out $< -t http://timestamp.digicert.com -pass $(DDEV_WINDOWS_SIGNING_PASSWORD); fi

	@makensis -DVERSION=$(VERSION) winpkg/ddev.nsi  # brew install makensis, apt-get install nsis, or install on Windows
	@if [ -z "$(DDEV_WINDOWS_SIGNING_PASSWORD)" ] ; then echo "Skipping signing ddev_windows_installer, no DDEV_WINDOWS_SIGNING_PASSWORD provided"; else echo "Signing windows installer binary..."&& osslsigncode sign -pkcs12 certfiles/drud_cs.p12  -n "DDEV-Local Installer" -i https://ddev.com -in $@ -out $@ -t http://timestamp.digicert.com -pass $(DDEV_WINDOWS_SIGNING_PASSWORD); fi
	shasum -a 256 $@ >$@.sha256.txt

no_v_version:
	@echo $(NO_V_VERSION)

chocolatey: $(GOTMP)/bin/windows_amd64/ddev_windows_installer.$(VERSION).exe
	rm -rf $(GOTMP)/bin/windows_amd64/chocolatey && cp -r winpkg/chocolatey $(GOTMP)/bin/windows_amd64/chocolatey
	perl -pi -e 's/REPLACE_DDEV_VERSION/$(NO_V_VERSION)/g' $(GOTMP)/bin/windows_amd64/chocolatey/*.nuspec
	perl -pi -e 's/REPLACE_DDEV_VERSION/$(VERSION)/g' $(GOTMP)/bin/windows_amd64/chocolatey/tools/*.ps1
	perl -pi -e 's/REPLACE_GITHUB_ORG/$(GITHUB_ORG)/g' $(GOTMP)/bin/windows_amd64/chocolatey/*.nuspec $(GOTMP)/bin/windows_amd64/chocolatey/tools/*.ps1 #GITHUB_ORG is for testing, for example when the binaries are on rfay acct
	perl -pi -e "s/REPLACE_INSTALLER_CHECKSUM/$$(cat $(GOTMP)/bin/windows_amd64/ddev_windows_installer.$(VERSION).exe.sha256.txt | awk '{ print $$1; }')/g" $(GOTMP)/bin/windows_amd64/chocolatey/tools/*
	docker run --rm -v $(PWD)/$(GOTMP)/bin/windows_amd64/chocolatey:/tmp/chocolatey -w /tmp/chocolatey linuturk/mono-choco pack ddev.nuspec
	@echo "chocolatey package is in $(GOTMP)/bin/windows_amd64/chocolatey"

$(GOTMP)/bin/windows_amd64/mkcert.exe $(GOTMP)/bin/windows_amd64/mkcert_license.txt:
	curl --fail -sSL -o $(GOTMP)/bin/windows_amd64/mkcert.exe  https://github.com/FiloSottile/mkcert/releases/download/$(MKCERT_VERSION)/mkcert-$(MKCERT_VERSION)-windows-amd64.exe
	curl --fail -sSL -o $(GOTMP)/bin/windows_amd64/mkcert_license.txt -O https://raw.githubusercontent.com/FiloSottile/mkcert/master/LICENSE

$(GOTMP)/bin/windows_amd64/sudo.exe $(GOTMP)/bin/windows_amd64/sudo_license.txt:
	curl  -sSL --create-dirs -o $(DOWNLOADTMP)/sudo.zip  https://github.com/mattn/sudo/releases/download/$(WINDOWS_SUDO_VERSION)/sudo-x86_64.zip
	unzip -o -d $(GOTMP)/bin/windows_amd64 $(DOWNLOADTMP)/sudo.zip
	curl --fail -sSL -o $(GOTMP)/bin/windows_amd64/sudo_license.txt https://raw.githubusercontent.com/mattn/sudo/master/LICENSE

$(GOTMP)/bin/windows_amd64/nssm.exe $(GOTMP)/bin/windows_amd64/winnfsd_license.txt $(GOTMP)/bin/windows_amd64/winnfsd.exe :
	curl --fail -sSL -o $(GOTMP)/bin/windows_amd64/winnfsd.exe  https://github.com/winnfsd/winnfsd/releases/download/$(WINNFSD_VERSION)/WinNFSd.exe
	curl --fail -sSL -o $(GOTMP)/bin/windows_amd64/nssm.exe https://github.com/drud/nssm/releases/download/$(NSSM_VERSION)/nssm.exe
	curl --fail -sSL -o $(GOTMP)/bin/windows_amd64/winnfsd_license.txt https://www.gnu.org/licenses/gpl.txt

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
