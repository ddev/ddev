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

# The format here is majorversion:pinnedimageversion
# This allows to pin the version (as currently required with 8.0.19
MARIADB_VERSIONS=5.5:5.5 10.0:10.0 10.1:10.1 10.2:10.2 10.3:10.3 10.4:10.4 10.5:10.5
MYSQL_VERSIONS=5.5:5.5 5.6:5.6 5.7:5.7 8.0:8.0.19

container: mariadb_containers mysql_containers

mariadb_containers:
	for item in $(MARIADB_VERSIONS); do \
		set -euo pipefail ; \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MariaDB $${baseversion} with image $${fullversion} ==========\n"; \
		docker build $(DOCKER_ARGS) --build-arg="DBVERSION=$${baseversion}" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${baseversion}:$(VERSION)" . ; \
	done

mysql_containers:
	for item in $(MYSQL_VERSIONS) ; do \
		set -euo pipefail ; \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MySQL $${baseversion} with image $${fullversion} ==========\n", ${baseversion}, ${fullversion}; \
		docker build $(DOCKER_ARGS) --build-arg="DBVERSION=$$baseversion" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mysql" -t "drud/ddev-dbserver-mysql-$${baseversion%%%.*}:$(VERSION)" . ; \
	done

push: container
	for item in $(MARIADB_VERSIONS); do \
		set -euo pipefail ;\
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		docker push "drud/ddev-dbserver-mariadb-$${baseversion%%%.*}:$(VERSION)"; \
	done
	for item in $(MYSQL_VERSIONS) ; do \
		set -euo pipefail ;\
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		docker push "drud/ddev-dbserver-mysql-$${baseversion%%%.*}:$(VERSION)"; \
	done


test: container
	bash ./test/test_dbserver.sh $(VERSION) # bash execution just for windows make

clean:
	for item in $(MARIADB_VERSIONS); do \
		if docker image inspect $(DOCKER_REPO)-mariadb-${item}:$(VERSION) >/dev/null 2>&1; then docker rmi -f $(DOCKER_REPO)-mariadb-$${item}:$(VERSION); fi; \
	done
	for item in $(MYSQL_VERSIONS); do \
		if docker image inspect $(DOCKER_REPO)-mysql-${item}:$(VERSION) >/dev/null 2>&1; then docker rmi -f $(DOCKER_REPO)-mysql-$${item}:$(VERSION); fi; \
	done
	@rm -rf .container-* .dockerfile* .push-* linux darwin windows container VERSION.txt .docker_image

