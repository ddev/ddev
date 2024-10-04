# Makefile for a standard golang repo with associated container

# Circleci doesn't seem to provide a decent way to add to path, just adding here, for case where
# linux build and linuxbrew is installed.
export PATH := $(EXTRA_PATH):$(PATH)

BUILD_BASE_DIR ?= $(PWD)

GOTMP=.gotmp
SHELL = /bin/bash
PWD = $(shell pwd)
GOFILES = $(shell find $(SRC_DIRS) -type f ! -path "*/testdata/*")
GORACE = "halt_on_error=1"
CGO_ENABLED = 0
.PHONY: darwin_amd64 darwin_arm64 darwin_amd64_notarized darwin_arm64_notarized darwin_arm64_signed darwin_amd64_signed linux_amd64 linux_arm64 linux_arm windows_amd64 windows_arm64 setup

# Expands SRC_DIRS into the common golang ./dir/... format for "all below"
SRC_AND_UNDER = $(patsubst %,./%/...,$(SRC_DIRS))

GOLANGCI_LINT_ARGS ?= --out-format=line-number --disable-all --enable=gofmt --enable=govet --enable=revive --enable=errcheck --enable=staticcheck --enable=ineffassign

WINDOWS_GSUDO_VERSION=v0.7.3
WINNFSD_VERSION=2.4.0
NSSM_VERSION=2.24-101-g897c7ad

TESTTMP=/tmp/testresults

# This repo's root import path (under GOPATH).
PKG := github.com/ddev/ddev

# Top-level directories to build
SRC_DIRS := cmd pkg

# Version variables to replace in build
VERSION_VARIABLES ?= DdevVersion AmplitudeAPIKey

# These variables will be used as the default unless overridden by the make
DdevVersion ?= $(VERSION)
# WebTag ?= $(VERSION)  # WebTag is normally specified in version_constants.go, sometimes overridden (night-build.mak)
# DBTag ?=  $(VERSION)  # DBTag is normally specified in version_constants.go, sometimes overridden (night-build.mak)
# RouterTag ?= $(VERSION) #RouterTag is normally specified in version_constants.go, sometimes overridden (night-build.mak)
# DBATag ?= $(VERSION) #DBATag is normally specified in version_constants.go, sometimes overridden (night-build.mak)

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
NO_V_VERSION=$(shell echo $(VERSION) | awk -F"-" '{ OFS="-"; sub(/^./, "", $$1); printf $$0; }')
GITHUB_ORG := ddev

BUILD_OS = $(shell go env GOHOSTOS)
BUILD_ARCH = $(shell go env GOHOSTARCH)
VERSION_LDFLAGS=$(foreach v,$(VERSION_VARIABLES),-X '$(PKG)/pkg/versionconstants.$(v)=$($(v))')
LDFLAGS=-extldflags -static $(VERSION_LDFLAGS)
DEFAULT_BUILD=$(shell go env GOHOSTOS)_$(shell go env GOHOSTARCH)

build: $(DEFAULT_BUILD)


# Provide shorthand targets
linux_amd64: $(GOTMP)/bin/linux_amd64/ddev
linux_arm64: $(GOTMP)/bin/linux_arm64/ddev
linux_arm: $(GOTMP)/bin/linux_arm/ddev
darwin_amd64: $(GOTMP)/bin/darwin_amd64/ddev
darwin_arm64: $(GOTMP)/bin/darwin_arm64/ddev
windows_amd64: windows_amd64_install
windows_arm64: windows_arm64_install
completions: $(GOTMP)/bin/completions.tar.gz

TARGETS=$(GOTMP)/bin/linux_amd64/ddev $(GOTMP)/bin/linux_arm64/ddev $(GOTMP)/bin/linux_arm/ddev $(GOTMP)/bin/darwin_amd64/ddev $(GOTMP)/bin/darwin_arm64/ddev $(GOTMP)/bin/windows_amd64/ddev.exe $(GOTMP)/bin/windows_arm64/ddev.exe
$(TARGETS): mkcert $(GOFILES)
	@echo "building $@ from $(SRC_AND_UNDER) GORACE=$(GORACE) CGO_ENABLED=$(CGO_ENABLED)";
	@#echo "LDFLAGS=$(LDFLAGS)";
	@rm -f $@
	@export TARGET=$(word 3, $(subst /, ,$@)) && \
	export CGO_ENABLED=$(CGO_ENABLED) GOOS="$${TARGET%_*}" GOARCH="$${TARGET#*_}" GOPATH="$(PWD)/$(GOTMP)" GOCACHE="$(PWD)/$(GOTMP)/.cache" && \
	mkdir -p $(GOTMP)/{.cache,pkg,src,bin/$$TARGET} && \
	chmod 777 $(GOTMP)/{.cache,pkg,src,bin/$$TARGET} && \
	go build -o $(GOTMP)/bin/$$TARGET -installsuffix static $(BUILDARGS) -ldflags " $(LDFLAGS) " $(SRC_AND_UNDER)
	$( shell if [ -d $(GOTMP) ]; then chmod -R u+w $(GOTMP); fi )
	@echo $(VERSION) >VERSION.txt

$(GOTMP)/bin/completions.tar.gz: build
	$(GOTMP)/bin/$(BUILD_OS)_$(BUILD_ARCH)/ddev_gen_autocomplete
	tar -C $(GOTMP)/bin/completions -czf $(GOTMP)/bin/completions.tar.gz .

mkcert: $(GOTMP)/bin/darwin_arm64/mkcert $(GOTMP)/bin/darwin_amd64/mkcert $(GOTMP)/bin/linux_arm64/mkcert $(GOTMP)/bin/linux_amd64/mkcert

# Download mkcert to it can be added to tarball installations
$(GOTMP)/bin/darwin_arm64/mkcert $(GOTMP)/bin/darwin_amd64/mkcert $(GOTMP)/bin/linux_arm64/mkcert $(GOTMP)/bin/linux_amd64/mkcert:
	@export TARGET=$(word 3, $(subst /, ,$@)) && \
	export GOOS="$${TARGET%_*}" GOARCH="$${TARGET#*_}" MKCERT_VERSION=v1.4.4 && \
	mkdir -p $(GOTMP)/bin/$${GOOS}_$${GOARCH} && \
	curl --fail -JL -s -o $(GOTMP)/bin/$${GOOS}_$${GOARCH}/mkcert "https://github.com/FiloSottile/mkcert/releases/download/$${MKCERT_VERSION}/mkcert-$${MKCERT_VERSION}-$${GOOS}-$${GOARCH}" && chmod +x $(GOTMP)/bin/$${GOOS}_$${GOARCH}/mkcert

TEST_TIMEOUT=4h
BUILD_ARCH = $(shell go env GOARCH)

DDEVNAME=ddev
SHASUM=shasum -a 256
ifeq ($(BUILD_OS),windows)
	DDEVNAME=ddev.exe
	SHASUM=sha256sum
	TEST_TIMEOUT=6h
endif

DDEV_PATH=$(PWD)/$(GOTMP)/bin/$(BUILD_OS)_$(BUILD_ARCH)
DDEV_BINARY_FULLPATH=$(DDEV_PATH)/$(DDEVNAME)

# Override test section with tests specific to ddev
test: testpkg testcmd

testcmd: $(DEFAULT_BUILD) setup
	@echo LDFLAGS=$(LDFLAGS)
	@echo DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH)
	export PATH="$(DDEV_PATH):$$PATH" DDEV_NO_INSTRUMENTATION=true CGO_ENABLED=$(CGO_ENABLED) DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH); go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags " $(LDFLAGS) " ./cmd/... $(TESTARGS)

testpkg: testnotddevapp testddevapp

testddevapp: $(DEFAULT_BUILD) setup
	export PATH="$(DDEV_PATH):$$PATH" DDEV_NO_INSTRUMENTATION=true CGO_ENABLED=$(CGO_ENABLED) DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH); go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags " $(LDFLAGS) " ./pkg/ddevapp $(TESTARGS)

testnotddevapp: $(DEFAULT_BUILD) setup
	export PATH="$(DDEV_PATH):$$PATH" DDEV_NO_INSTRUMENTATION=true CGO_ENABLED=$(CGO_ENABLED) DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH); go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags " $(LDFLAGS) " $(shell find ./pkg -maxdepth 1 -type d ! -name ddevapp ! -name pkg) $(TESTARGS)

testfullsitesetup: $(DEFAULT_BUILD) setup
	export PATH="$(DDEV_PATH):$$PATH" DDEV_NO_INSTRUMENTATION=true CGO_ENABLED=$(CGO_ENABLED) DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH); go test $(USEMODVENDOR) -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags " $(LDFLAGS) " ./pkg/ddevapp -run TestDdevFullSiteSetup $(TESTARGS)

setup:
	@mkdir -p $(GOTMP)/{bin/linux_arm64,bin/linux_amd64,bin/darwin_arm64,bin/darwin_amd64,bin/windows_amd64,bin/windows_arm64,src,pkg/mod/cache,.cache}
	@mkdir -p $(TESTTMP)

# Required static analysis targets used in circleci - these cause fail if they don't work
staticrequired: setup golangci-lint markdownlint mkdocs pyspelling

# Best to install markdownlint-cli locally with "npm install -g markdownlint-cli"
markdownlint:
	@echo "markdownlint: "
	@CMD="markdownlint *.md docs/content 2>&1"; \
	set -eu -o pipefail; \
	if command -v markdownlint >/dev/null 2>&1 ; then \
		$$CMD; \
	else \
		echo "Skipping markdownlint as not installed"; \
	fi

# Best to install mkdocs locally with "sudo pip3 install -r docs/mkdocs-pip-requirements"
mkdocs:
	@echo "mkdocs: "
	@CMD="mkdocs -q build -d /tmp/mkdocsbuild"; \
	if command -v mkdocs >/dev/null 2>&1; then \
		$$CMD ; \
	else \
		echo "Not running mkdocs because it's not installed"; \
	fi

# To see what the docs build will be, you can use `make mkdocs-serve`
# It works best with mkdocs installed, `pip3 install mkdocs`,
# see https://www.mkdocs.org/user-guide/installation/
# But it will also work using docker.
MKDOCS_TAG := 1.5.2
ifeq ($(BUILD_ARCH),arm64)
    MKDOCS_TAG := arm64v8-$(MKDOCS_TAG)
endif
mkdocs-serve:
	set -x; \
	if command -v mkdocs >/dev/null ; then \
  		mkdocs serve; \
	else \
		docker run -it -p 8000:8000 -v "${PWD}:/docs" -e "ADD_MODULES=mkdocs-material mkdocs-redirects mkdocs-minify-plugin mdx_truly_sane_lists mkdocs-git-revision-date-localized-plugin" -e "LIVE_RELOAD_SUPPORT=true" -e "FAST_MODE=true" -e "DOCS_DIRECTORY=./docs" "polinux/mkdocs:$(MKDOCS_TAG)"; \
	fi; \
	set +x

# Install markdown-link-check locally with "npm install -g markdown-link-check"
markdown-link-check:
	@echo "markdown-link-check: "
	if command -v markdown-link-check >/dev/null 2>&1; then \
  		find docs *.md -name "*.md" -exec markdown-link-check -q -c markdown-link-check.json {} \; 2>&1  ;\
	else \
		echo "Not running markdown-link-check because it's not installed"; \
	fi

# Best to install pyspelling locally with "sudo -H pip3 install pyspelling pymdown-extensions". Also requries aspell, `sudo apt-get install aspell"
pyspelling:
	@echo "pyspelling: "
	@CMD="pyspelling --config .spellcheck.yml"; \
	set -eu -o pipefail; \
	if command -v pyspelling >/dev/null 2>&1 ; then \
		$$CMD; \
	else \
		echo "Not running pyspelling because it's not installed"; \
	fi

# Install textlint locally with `npm install -g textlint textlint-filter-rule-comments textlint-rule-no-todo textlint-rule-stop-words textlint-rule-terminology`
textlint:
	@echo "textlint: "
	@CMD="textlint {README.md,version-history.md,docs/**}"; \
	set -eu -o pipefail; \
	if command -v textlint >/dev/null 2>&1 ; then \
		$$CMD; \
	else \
		echo "textlint is not installed"; \
	fi

darwin_amd64_signed: $(GOTMP)/bin/darwin_amd64/ddev
	@if [ -z "$(DDEV_MACOS_SIGNING_PASSWORD)" ] ; then echo "Skipping signing ddev for macOS, no DDEV_MACOS_SIGNING_PASSWORD provided"; else echo "Signing $< ..."; \
		set -o errexit -o pipefail; \
		curl -s https://raw.githubusercontent.com/ddev/signing_tools/master/macos_sign.sh | bash -s -  --signing-password="$(DDEV_MACOS_SIGNING_PASSWORD)" --cert-file=certfiles/ddev_developer_id_cert.p12 --cert-name="Developer ID Application: Localdev Foundation (9HQ298V2BW)" --target-binary="$<" ; \
	fi
darwin_arm64_signed: $(GOTMP)/bin/darwin_arm64/ddev
	@if [ -z "$(DDEV_MACOS_SIGNING_PASSWORD)" ] ; then echo "Skipping signing ddev for macOS, no DDEV_MACOS_SIGNING_PASSWORD provided"; else echo "Signing $< ..."; \
		set -o errexit -o pipefail; \
		codesign --remove-signature "$(GOTMP)/bin/darwin_arm64/ddev" || true; \
		curl -s https://raw.githubusercontent.com/ddev/signing_tools/master/macos_sign.sh | bash -s -  --signing-password="$(DDEV_MACOS_SIGNING_PASSWORD)" --cert-file=certfiles/ddev_developer_id_cert.p12 --cert-name="Developer ID Application: Localdev Foundation (9HQ298V2BW)" --target-binary="$<" ; \
	fi

darwin_amd64_notarized: darwin_amd64_signed
	@if [ -z "$(DDEV_MACOS_APP_PASSWORD)" ]; then echo "Skipping notarizing ddev for macOS, no DDEV_MACOS_APP_PASSWORD provided"; else \
		set -o errexit -o pipefail; \
		echo "Notarizing $(GOTMP)/bin/darwin_amd64/ddev ..." ; \
		curl -sSL -f https://raw.githubusercontent.com/ddev/signing_tools/master/macos_notarize.sh | bash -s -  --app-specific-password=$(DDEV_MACOS_APP_PASSWORD) --apple-id=notarizer@localdev.foundation --primary-bundle-id=com.ddev.ddev --target-binary="$(GOTMP)/bin/darwin_amd64/ddev" ; \
	fi
darwin_arm64_notarized: darwin_arm64_signed
	@if [ -z "$(DDEV_MACOS_APP_PASSWORD)" ]; then echo "Skipping notarizing ddev for macOS, no DDEV_MACOS_APP_PASSWORD provided"; else \
		set -o errexit -o pipefail; \
		echo "Notarizing $(GOTMP)/bin/darwin_arm64/ddev ..." ; \
		curl -sSL -f https://raw.githubusercontent.com/ddev/signing_tools/master/macos_notarize.sh | bash -s - --app-specific-password=$(DDEV_MACOS_APP_PASSWORD) --apple-id=notarizer@localdev.foundation --primary-bundle-id=com.ddev.ddev --target-binary="$(GOTMP)/bin/darwin_arm64/ddev" ; \
	fi

windows_amd64_install: $(GOTMP)/bin/windows_amd64/ddev_windows_amd64_installer.exe
windows_arm64_install: $(GOTMP)/bin/windows_arm64/ddev_windows_arm64_installer.exe

windows_sign_binaries: $(GOTMP)/bin/windows_amd64/ddev.exe $(GOTMP)/bin/windows_amd64/mkcert.exe $(GOTMP)/bin/windows_arm64/ddev.exe $(GOTMP)/bin/windows_arm64/mkcert.exe
	ls -l .gotmp/bin/windows_amd64
	@if [ "$(DDEV_WINDOWS_SIGN)" != "true" ] ; then echo "Skipping signing amd64 ddev.exe, DDEV_WINDOWS_SIGN not set"; else echo "Signing windows amd64 binaries..." && signtool sign -fd SHA256 ".gotmp/bin/windows_amd64/ddev.exe" ".gotmp/bin/windows_amd64/mkcert.exe" ".gotmp/bin/windows_amd64/ddev_gen_autocomplete.exe"; fi
	ls -l .gotmp/bin/windows_arm64
	@if [ "$(DDEV_WINDOWS_SIGN)" != "true" ] ; then echo "Skipping signing arm64 ddev.exe, DDEV_WINDOWS_SIGN not set"; else echo "Signing windows arm64 binaries..." && signtool sign -fd SHA256 ".gotmp/bin/windows_arm64/ddev.exe" ".gotmp/bin/windows_arm64/mkcert.exe" ".gotmp/bin/windows_arm64/ddev_gen_autocomplete.exe"; fi

$(GOTMP)/bin/windows_amd64/ddev_windows_amd64_installer.exe: windows_sign_binaries $(GOTMP)/bin/windows_amd64/sudo_license.txt $(GOTMP)/bin/windows_amd64/mkcert_license.txt winpkg/ddev.nsi
	@makensis -DTARGET_ARCH=amd64 -DVERSION=$(VERSION) winpkg/ddev.nsi  # brew install makensis, apt-get install nsis, or install on Windows
	@if [ "$(DDEV_WINDOWS_SIGN)" != "true" ] ; then echo "Skipping signing amd64 $@, DDEV_WINDOWS_SIGN not set"; else echo "Signing windows installer amd64 binary..." && signtool sign -fd SHA256 "$@"; fi
	$(SHASUM) $@ >$@.sha256.txt

$(GOTMP)/bin/windows_arm64/ddev_windows_arm64_installer.exe: windows_sign_binaries $(GOTMP)/bin/windows_arm64/sudo_license.txt $(GOTMP)/bin/windows_arm64/mkcert_license.txt winpkg/ddev.nsi
	@makensis -DTARGET_ARCH=arm64 -DVERSION=$(VERSION) winpkg/ddev.nsi  # brew install makensis, apt-get install nsis, or install on Windows
	@if [ "$(DDEV_WINDOWS_SIGN)" != "true" ] ; then echo "Skipping signing arm64 $@, DDEV_WINDOWS_SIGN not set"; else echo "Signing windows installer arm64 binary..." && signtool sign -fd SHA256 "$@"; fi
	$(SHASUM) $@ >$@.sha256.txt

no_v_version:
	@echo $(NO_V_VERSION)

chocolatey: $(GOTMP)/bin/windows_amd64/ddev_windows_amd64_installer.exe
	rm -rf $(GOTMP)/bin/windows_amd64/chocolatey && cp -r winpkg/chocolatey $(GOTMP)/bin/windows_amd64/chocolatey
	perl -pi -e 's/REPLACE_DDEV_VERSION/$(NO_V_VERSION)/g' $(GOTMP)/bin/windows_amd64/chocolatey/*.nuspec
	perl -pi -e 's/REPLACE_DDEV_VERSION/$(VERSION)/g' $(GOTMP)/bin/windows_amd64/chocolatey/tools/*.ps1
	perl -pi -e 's/REPLACE_GITHUB_ORG/$(REPOSITORY_OWNER)/g' $(GOTMP)/bin/windows_amd64/chocolatey/*.nuspec $(GOTMP)/bin/windows_amd64/chocolatey/tools/*.ps1 #GITHUB_ORG is for testing, for example when the binaries are on rfay acct
	perl -pi -e "s/REPLACE_INSTALLER_CHECKSUM/$$(cat $(GOTMP)/bin/windows_amd64/ddev_windows_amd64installer.exe.sha256.txt | awk '{ print $$1; }')/g" $(GOTMP)/bin/windows_amd64/chocolatey/tools/*
	if [[ "$(NO_V_VERSION)" =~ -g[0-9a-f]+ ]]; then \
		echo "Skipping chocolatey build on interim version"; \
	else \
		docker run --rm -v "/$(PWD)/$(GOTMP)/bin/windows_amd64/chocolatey:/tmp/chocolatey" -w "//tmp/chocolatey" linuturk/mono-choco pack ddev.nuspec; \
		echo "chocolatey package is in $(GOTMP)/bin/windows_amd64/chocolatey"; \
	fi

$(GOTMP)/bin/windows_amd64/mkcert.exe $(GOTMP)/bin/windows_amd64/mkcert_license.txt:
	curl --fail -JL -s -o $(GOTMP)/bin/windows_amd64/mkcert.exe "https://dl.filippo.io/mkcert/latest?for=windows/amd64"
	curl --fail -sSL -o $(GOTMP)/bin/windows_amd64/mkcert_license.txt -O https://raw.githubusercontent.com/FiloSottile/mkcert/master/LICENSE

$(GOTMP)/bin/windows_arm64/mkcert.exe $(GOTMP)/bin/windows_arm64/mkcert_license.txt:
	curl --fail -JL -s -o $(GOTMP)/bin/windows_arm64/mkcert.exe "https://dl.filippo.io/mkcert/latest?for=windows/arm64"
	curl --fail -sSL -o $(GOTMP)/bin/windows_arm64/mkcert_license.txt -O https://raw.githubusercontent.com/FiloSottile/mkcert/master/LICENSE

$(GOTMP)/bin/windows_amd64/sudo_license.txt:
	set -x
	curl --fail -sSL -o "$(GOTMP)/bin/windows_amd64/sudo_license.txt" "https://raw.githubusercontent.com/gerardog/gsudo/master/LICENSE.txt"

$(GOTMP)/bin/windows_arm64/sudo_license.txt:
	set -x
	curl --fail -sSL -o "$(GOTMP)/bin/windows_arm64/sudo_license.txt" "https://raw.githubusercontent.com/gerardog/gsudo/master/LICENSE.txt"

# Best to install golangci-lint locally with "curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.31.0"
golangci-lint:
	@echo "golangci-lint: "
	@CMD="golangci-lint run $(GOLANGCI_LINT_ARGS) $(SRC_AND_UNDER)"; \
	set -eu -o pipefail; \
	if command -v golangci-lint >/dev/null 2>&1; then \
		$$CMD; \
	else \
		echo "Skipping golanci-lint as not installed"; \
	fi

version:
	@echo VERSION:$(VERSION)

clean: bin-clean

bin-clean:
	@rm -rf bin
	$(shell if [ -d $(GOTMP) ]; then chmod -R u+w $(GOTMP) && rm -rf $(GOTMP); fi )

# print-ANYVAR prints the expanded variable
print-%: ; @echo $* = $($*)
