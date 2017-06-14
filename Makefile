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
WebTag ?= v0.6.1
DBImg ?= drud/mysql-docker-local-57
DBTag ?= v0.4.1
RouterImage ?= drud/ddev-router
RouterTag ?= v0.4.2
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

# Run tests with -short by default, for faster run times.
TESTARGS ?= -short

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
DDEV_BINARY_FULLPATH=$(shell pwd)/bin/$(TESTOS)/ddev

# Override test section with tests specific to ddev
test: testpkg testcmd

testcmd: build setup
	PATH=$$PWD/bin/$(TESTOS):$$PATH CGO_ENABLED=0 DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH) go test -p 1 -timeout 20m -v -installsuffix static -ldflags '$(LDFLAGS)' ./cmd/... $(TESTARGS)

testpkg:
	PATH=$$PWD/bin/$(TESTOS):$$PATH CGO_ENABLED=0 DDEV_BINARY_FULLPATH=$(DDEV_BINARY_FULLPATH) DRUD_DEBUG=true go test  -timeout 20m -v -installsuffix static -ldflags '$(LDFLAGS)' ./pkg/... $(TESTARGS)

setup:
	@mkdir -p bin/darwin bin/linux
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/linux
	@if [ ! -L $$PWD/bin/darwin/ddev ] ; then ln -s $$PWD/bin/darwin/darwin_amd64/ddev $$PWD/bin/darwin/ddev; fi

# Required static analysis targets used in circleci - these cause fail if they don't work
staticrequired: gofmt govet golint errcheck staticcheck codecoroner

