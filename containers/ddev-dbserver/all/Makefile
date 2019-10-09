# Makefile for a standard repo with associated container

##### These variables need to be adjusted in most repositories #####

# This repo's root import path (under GOPATH).
# PKG := github.com/drud/repo_name

# Docker repo for a push
DOCKER_REPO ?= drud/ddev-dbserver
SHELL=/bin/bash

# Upstream repo used in the Dockerfile
# UPSTREAM_REPO ?= oraclelinux:7.3

# Top-level directories to build
#SRC_DIRS := files drudapi secrets utils

# Optional to docker build
# DOCKER_ARGS = --build-arg MYSQL_PACKAGE_VERSION=5.7.17-1


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
#include ../../../build-tools/makefile_components/base_build_go.mak
#include ../../build-tools/makefile_components/base_build_python-docker.mak
#include ../../build-tools/makefile_components/base_container.mak
#include ../../build-tools/makefile_components/base_push.mak
#include build-tools/makefile_components/base_test_go.mak
#include ../../build-tools/makefile_components/base_test_python.mak

build: container

MARIADB_VERSIONS=5.5 10.0 10.1 10.2 10.3 10.4

#container: mariadb5.5 mariadb10.0 mariadb10.1 mariadb10.2 mariadb10.3 mariadb10.4

#mariadb5.5:
#	docker build $(DOCKER_ARGS) --build-arg="DBTYPE=mariadb" --build-arg "DBVERSION=5.5" -t "drud/ddev-dbserver-mariadb-5.5:$(VERSION)" . ;
#mariadb10.0:
#	docker build $(DOCKER_ARGS) --build-arg "DBVERSION=10.0" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-10.0:$(VERSION)" . ;
#mariadb10.1:
#	docker build $(DOCKER_ARGS) --build-arg "DBVERSION=10.1" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-10.1:$(VERSION)" . ;
#mariadb10.2:
#	docker build $(DOCKER_ARGS) --build-arg "DBVERSION=10.2" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-10.2:$(VERSION)" . ;
#mariadb10.3:
#	docker build $(DOCKER_ARGS) --build-arg "DBVERSION=10.3" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-10.3:$(VERSION)" . ;
#mariadb10.4:
#	docker build $(DOCKER_ARGS) --build-arg "DBVERSION=10.4" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-10.4:$(VERSION)" . ;

MARIADB_VERSIONS=5.5 10.0 10.1 10.2 10.3 10.4
container:
	for item in $(MARIADB_VERSIONS); do \
		docker build $(DOCKER_ARGS) --build-arg "DBVERSION=$${item}" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${item}:$(VERSION)" . ; \
	done
#	for item in 5.5 5.6 5.7 ; do \
#		docker build --build-arg "DBVERSION=$$item" --build-arg="DBTYPE=mysql" -t "drud/ddev-dbserver-mysql-$${item%%%.*}:$(VERSION)" . ; \
#	done

push: container
	for item in $(MARIADB_VERSIONS); do \
		docker push "drud/ddev-dbserver-mariadb-$${item%%%.*}:$(VERSION)"; \
	done

test: container
	set -x ; for item in $(MARIADB_VERSIONS); do \
		echo $$item && ../test/testserver.sh $(DOCKER_REPO)-mariadb-$$item:$(VERSION) $$item ;\
	done

clean:
	@if docker image inspect $(DOCKER_REPO):$(VERSION) >/dev/null 2>&1; then docker rmi -f $(DOCKER_REPO):$(VERSION); fi
	@rm -rf .container-* .dockerfile* .push-* linux darwin windows container VERSION.txt .docker_image
