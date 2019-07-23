# Makefile for a standard golang repo with associated container

# Circleci doesn't seem to provide a decent way to add to path, just adding here, for case where
# linux build and linuxbrew is installed.
export PATH := $(EXTRA_PATH):$(PATH)

# Not updating build-tools to get this, but this should be removed
# when build-tools is updated.
BUILD_IMAGE := drud/golang-build-container:v1.12.7

GOMETALINTER_ARGS := --vendored-linters --disable-all --enable=gofmt --enable=vet --enable vetshadow --enable=golint --enable=errcheck --enable=staticcheck --enable=ineffassign --enable=varcheck --enable=deadcode --deadline=4m
GOLANGCI_LINT_ARGS ?= --out-format=line-number --disable-all --enable=gofmt --enable=govet --enable=golint --enable=errcheck --enable=staticcheck --enable=ineffassign --enable=varcheck --enable=deadcode

WINDOWS_SUDO_VERSION=v0.0.1
WINNFSD_VERSION=2.4.0
NSSM_VERSION=2.24-101-g897c7ad
MKCERT_VERSION=v1.3.0

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
VERSION_VARIABLES ?= DdevVersion SentryDSN SegmentKey

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
# no_v_version removes the front v, keeps the special to 20 chars, uses -alpha before the rest.
NO_V_VERSION=$(shell echo $(VERSION) | awk -F"-" '{sub(/^./, "", $$1); printf $$1; if (NF >2) { printf("-alpha-%s-%s", $$2, $$3); } }')
GITHUB_ORG := drud

#
# This version-strategy uses a manual value to set the version string
#VERSION := 1.2.3

# Each section of the Makefile is included from standard components below.
# If you need to override one, import its contents below and comment out the
# include. That way the base components can easily be updated as our general needs
# change.
include build-tools/makefile_components/base_build_go.mak
#include build-tools/makefile_components/base_build_python-docker.mak
#include build-tools/makefile_components/base_container.mak
#include build-tools/makefile_components/base_push.mak
#include build-tools/makefile_components/base_test_go.mak
#include build-tools/makefile_components/base_test_python.mak

.PHONY: test testcmd testpkg build setup staticrequired windows_install

TESTOS = $(shell uname -s | tr '[:upper:]' '[:lower:]')

TEST_TIMEOUT=120m
BUILD_ARCH = $(shell go env GOARCH)
ifeq ($(BUILD_OS),linux)
    DDEV_BINARY_FULLPATH=$(PWD)/$(GOTMP)/bin/ddev
endif

ifeq ($(BUILD_OS),windows)
    DDEV_BINARY_FULLPATH=$(shell pwd)/$(GOTMP)/bin/$(BUILD_OS)_$(BUILD_ARCH)/ddev.exe
endif

ifeq ($(BUILD_OS),darwin)
    DDEV_BINARY_FULLPATH=$(PWD)/$(GOTMP)/bin/$(BUILD_OS)_$(BUILD_ARCH)/ddev
endif

# Override test section with tests specific to ddev
test: testpkg testcmd

testcmd: $(BUILD_OS) setup
	DDEV_NO_SENTRY=true CGO_ENABLED=0 DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH) go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags '$(LDFLAGS)' ./cmd/... $(TESTARGS)

testpkg: setup
	DDEV_NO_SENTRY=true CGO_ENABLED=0 go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags '$(LDFLAGS)' ./pkg/... $(TESTARGS)

setup:
	@mkdir -p $(GOTMP)/{src,pkg/mod/cache,.cache}
	@mkdir -p $(TESTTMP)
	@mkdir -p $(DOWNLOADTMP)

# packr2 target currently builds packr2 caches in the cmd/ddev/cmd directory only
# using the golang-build-container's packr2 command
packr2:
	docker run -t --rm -u $(shell id -u):$(shell id -g)    \
          	    -v "$(S)$(PWD):/workdir$(DOCKERMOUNTFLAG)"  \
          	    -v "$(S)$(PWD)/$(GOTMP)/bin:$(S)/go/bin" \
          	    -e GOCACHE="//workdir/$(GOTMP)/.cache" \
          	    -w //workdir/cmd/ddev/cmd       \
          	    $(BUILD_IMAGE) packr2

# Required static analysis targets used in circleci - these cause fail if they don't work
staticrequired: setup golangci-lint

$(GOTMP)/bin/darwin_amd64/ddev: darwin
	@if [ -z "$(DDEV_MACOS_SIGNING_PASSWORD)" ] ; then echo "Skipping signing ddev for macOS, no DDEV_MACOS_SIGNING_PASSWORD provided"; else echo "Signing macOS ddev..."; \
		security create-keychain -p "$(DDEV_MACOS_SIGNING_PASSWORD)" buildagent; \
		security list-keychains -s buildagent; \
		security unlock-keychain -p "$(DDEV_MACOS_SIGNING_PASSWORD)" buildagent; \
		security default-keychain -s buildagent; \
		security import certfiles/macos_ddev_cert.p12 -k buildagent -P "$(DDEV_MACOS_SIGNING_PASSWORD)" -T /usr/bin/codesign ; \
		security set-key-partition-list -S apple-tool:,apple: -s -k "$(DDEV_MACOS_SIGNING_PASSWORD)" buildagent ; \
		codesign --keychain buildagent -s "Apple Distribution: DRUD Technology, LLC (3BAN66AG5M)" $@ ; \
		security delete-keychain buildagent ; \
		codesign -v $@ ; \
	fi


$(GOTMP)/bin/windows_amd64/ddev.exe: windows

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

