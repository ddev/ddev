# This makefile is structured to allow building a complete ddev, with clean/fresh containers at current HEAD.

# Build with a technique like this:
# VERSION=nightly.$(date +%Y%m%d%H%M%S) make -f nightly_build.mak clean && make -f nightly_build.mak --print-directory VERSION=$VERSION DdevVersion=$VERSION DBTag=$VERSION WebTag=$VERSION RouterTag=$VERSION UPSTREAM_PHP_REPO_TAG=$VERSION

# TODO:
#   * Build the dependencies first?
#   * Consider building it with make VERSION=201704051700 NGINX_LOCAL_UPSTREAM_FPM7_REPO_TAG=201704051700 NGINX_LOCAL_UPSTREAM_FPM7_REPO_TAG=201704051700
#   * Test should validate that the correct upstream got used.
#   # Then build ddev (with those tags)
#   * Set up a nightly build for it. https://circleci.com/docs/1.0/nightly-builds/ and https://circleci.com/docs/api/v1-reference/#new-build

SHELL := /bin/bash

# These dirs must be built in this order (nginx-php-fpm depends on php7)
CONTAINER_DIRS = docker.php7 docker.nginx-php-fpm docker.nginx-php-fpm-local mysql-docker-local nginx-proxy
ALL_DIRS = $(CONTAINER_DIRS) $(BINARY_DIRS)

BASEDIR=./containers/

.PHONY: $(CONTAINER_DIRS) all build test clean container build submodules

# Build container dirs then build binaries
all: submodules container test

container: $(CONTAINER_DIRS)

clean:
	for item in $(CONTAINER_DIRS); do \
		echo $$item && $(MAKE) -C $(addprefix $(BASEDIR),$$item) --no-print-directory clean; \
	done
	$(MAKE) clean


$(CONTAINER_DIRS):
	git --git-dir=$(addprefix $(BASEDIR),$@)/.git fetch && git --git-dir=$(addprefix $(BASEDIR),$@)/.git checkout  origin/master
	$(MAKE) -C $(addprefix $(BASEDIR),$@) --print-directory test

submodules:
	git submodule update --init

test:
	$(MAKE) && $(MAKE) test


