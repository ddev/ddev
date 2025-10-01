# Makefile for a standard repo with associated container

##### These variables need to be adjusted in most repositories #####

# This repo's root import path (under GOPATH).
# PKG := github.com/ddev/repo_name

VERSION := $(shell git describe --tags --always --dirty)
DEFAULT_IMAGES = test-ssh-server

# Tests always run against amd64 (build host). Once tests have passed, a multi-arch build
# will be generated and pushed (the amd64 build will be cached automatically to prevent it from building twice).
# arm64 doesn't make sense here (yet) since gotechnies/alpine-ssh is amd64-only
BUILD_ARCHS?=linux/amd64,linux/arm64

include ../containers_shared.mk

# Docker repo for a push
DOCKER_REPO ?= $(DOCKER_ORG)/test-ssh-server

test: container
	true
