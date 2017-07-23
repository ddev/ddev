# Makefile for a standard golang repo with associated container

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
VERSION_VARIABLES = DdevVersion WebImg WebTag DBImg DBTag RouterImage RouterTag DBAImg DBATag

# These variables will be used as the default unless overridden by the make
DdevVersion ?= $(VERSION)
WebImg ?= drud/nginx-php-fpm7-local
WebTag ?= v0.7.3
DBImg ?= drud/mysql-local-57
DBTag ?= v0.6.2
RouterImage ?= drud/ddev-router
RouterTag ?= v0.4.3
DBAImg ?= drud/phpmyadmin
DBATag ?= v0.2.0

# Optional to docker build
#DOCKER_ARGS =

# VERSION can be set by
  # Default: git tag
  # make command line: make VERSION=0.9.0
# It can also be explicitly set in the Makefile as commented out below.

# This version-strategy uses git tags to set the version string
# VERSION can be overridden on make commandline: make VERSION=0.9.1 push
VERSION := $(shell git describe --tags --always --dirty)

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

.PHONY: test testcmd testpkg build setup staticrequired

TESTOS = $(shell uname -s | tr '[:upper:]' '[:lower:]')

TEST_TIMEOUT=20m
BUILD_ARCH = $(shell go env GOARCH)
ifeq ($(BUILD_OS),linux)
    DDEV_BINARY_FULLPATH=$(PWD)/bin/$(BUILD_OS)/ddev
endif

ifeq ($(BUILD_OS),windows)
    DDEV_BINARY_FULLPATH=$(PWD)/bin/$(BUILD_OS)/$(BUILD_OS)_$(BUILD_ARCH)/ddev.exe
    TEST_TIMEOUT=40m
endif

ifeq ($(BUILD_OS),darwin)
    DDEV_BINARY_FULLPATH=$(PWD)/bin/$(BUILD_OS)/$(BUILD_OS)_$(BUILD_ARCH)/ddev
endif


# Override test section with tests specific to ddev
test: testpkg testcmd

testcmd: $(BUILD_OS) setup
	CGO_ENABLED=0 DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH) go test -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags '$(LDFLAGS)' ./cmd/... $(TESTARGS)

testpkg:
	CGO_ENABLED=0 go test -p 1 -timeout $(TEST_TIMEOUT) -v -installsuffix static -ldflags '$(LDFLAGS)' ./pkg/... $(TESTARGS)

setup:
	@mkdir -p bin/darwin bin/linux
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/linux

# Required static analysis targets used in circleci - these cause fail if they don't work
staticrequired: gofmt govet golint errcheck staticcheck codecoroner
