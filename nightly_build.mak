# This makefile is structured to allow building a complete ddev, with clean/fresh containers at current HEAD.

# Build with a technique like this:
# export VERSION=nightly.$(date +%Y%m%d%H%M%S); make -f nightly_build.mak clean && make -f nightly_build.mak --print-directory VERSION=$VERSION DdevVersion=$VERSION DBTag=$VERSION WebTag=$VERSION DBATag=$VERSION RouterTag=$VERSION  NGINX_LOCAL_UPSTREAM_FPM7_REPO_TAG=$VERSION NGINX_LOCAL_UPSTREAM_FPM7_REPO_TAG=$VERSION UPSTREAM_PHP_REPO_TAG=$VERSION


SHELL := /bin/bash

# These dirs must be built in this order (nginx-php-fpm depends on php7)
CONTAINER_DIRS = nginx-proxy docker.php7 docker.nginx-php-fpm docker.nginx-php-fpm-local mysql-docker-local docker.phpmyadmin

BASEDIR=./containers/

.PHONY: $(CONTAINER_DIRS) all build test clean container build submodules

# Build container dirs then build binaries
all: container test

container: submodules $(CONTAINER_DIRS)

clean:
	for item in $(CONTAINER_DIRS); do \
		echo $$item && $(MAKE) -C $(addprefix $(BASEDIR),$$item) --no-print-directory clean; \
	done
	$(MAKE) clean


$(CONTAINER_DIRS):
	$(MAKE) -C $(addprefix $(BASEDIR),$@) --print-directory test

submodules:
	git fetch --all && git submodule update --init && git submodule update --remote

test:
	$(MAKE) && $(MAKE) TESTARGS="" test
