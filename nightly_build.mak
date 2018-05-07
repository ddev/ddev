# This makefile is structured to allow building a complete ddev, with clean/fresh containers at current HEAD.

SHELL := /bin/bash

# These dirs must be built in this order (nginx-php-fpm depends on php7)
CONTAINER_DIRS = ddev-router nginx-php-fpm-local mysql-local phpmyadmin

BASEDIR=./containers/

.PHONY: $(CONTAINER_DIRS) all build test clean container build

# Build container dirs then build binaries
all: container test

container: $(CONTAINER_DIRS)

clean:
	for item in $(CONTAINER_DIRS); do \
		echo $$item && $(MAKE) -C $(addprefix $(BASEDIR),$$item) --no-print-directory clean; \
	done
	$(MAKE) clean


$(CONTAINER_DIRS):
	$(MAKE) -C $(addprefix $(BASEDIR),$@) --print-directory test

test:
	$(MAKE) && $(MAKE) TESTARGS="" test
