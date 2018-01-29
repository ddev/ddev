# Base Build portion of makefile
##### PLEASE DO NOT CHANGE THIS FILE #####
##### If one of these sections does not meet your needs, consider copying its
##### contents into ../Makefile and commenting out the include and adding a
##### comment about what you did and why.


.PHONY: all build test push clean container-clean bin-clean version

SHELL := /bin/bash

PWD := $(shell pwd)

BUILD_IMAGE ?= drud/golang-build-container:v0.5.2

all: VERSION.txt build

build: linux

linux:
#	@echo "No code to build to build in "$@" - try 'make container'

VERSION.txt:
	@echo $(VERSION) >VERSION.txt

version:
	@echo VERSION:$(VERSION)

clean: container-clean bin-clean

container-clean:
	rm -rf .container-* .dockerfile* .push-* linux darwin container VERSION.txt .docker_image

bin-clean:
	rm -rf .go $(GOTMP) bin .tmp
