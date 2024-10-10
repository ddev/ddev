DOCKER_REPO ?= ddev/ddev-dbserver

VERSION := $(shell git describe --tags --always --dirty)

.PHONY: build clean test

CURRENT_ARCH=$(shell ../get_arch.sh)

# This Makefile explicitly does not include containers/containers_shared.mak,
# So has to explicitly declare anything it might need from there (like SHELL)
SHELL = /bin/bash

BUILD_TARGETS=mariadb_5.5_amd64 mariadb_10.0_amd64 mariadb_10.1_both mariadb_10.2_both mariadb_10.3_both mariadb_10.4_both mariadb_10.5_both mariadb_10.6_both mariadb_10.7_both mariadb_10.8_both mariadb_10.11_both mariadb_11.4_both mysql_5.5_amd64 mysql_5.6_amd64 mysql_5.7_both mysql_8.0_both mysql_8.4_both
TEST_TARGETS=$(shell if [ "$(CURRENT_ARCH)" = "amd64" ] ; then \
	echo "mariadb_5.5_test mariadb_10.0_test mariadb_10.1_test mariadb_10.2_test mariadb_10.3_test mariadb_10.4_test mariadb_10.5_test mariadb_10.6_test mariadb_10.7_test mariadb_10.8_test mariadb_10.11_test mariadb_11.4_test mysql_5.5_test mysql_5.6_test mysql_5.7_test mysql_8.0_test mysql_8.4_test"; \
	else \
	  echo "mariadb_10.2_test mariadb_10.3_test mariadb_10.4_test mariadb_10.5_test mariadb_10.6_test mariadb_10.7_test mariadb_10.8_test mariadb_10.11_test mariadb_11.4_test mysql_5.7_test mysql_8.0_test"; \
  	fi )

container: build
build: $(BUILD_TARGETS)

mariadb_5.5: mariadb_5.5_amd64
mariadb_10.0: mariadb_10.0_amd64
mariadb_10.1: mariadb_10.1_both
mariadb_10.2: mariadb_10.2_both
mariadb_10.3: mariadb_10.3_both
mariadb_10.4: mariadb_10.4_both
mariadb_10.5: mariadb_10.5_both
mariadb_10.6: mariadb_10.6_both
mariadb_10.7: mariadb_10.7_both
mariadb_10.8: mariadb_10.8_both
mariadb_10.11: mariadb_10.11_both
mariadb_11.4: mariadb_11.4_both

mysql_5.5: mysql_5.5_amd64
mysql_5.6: mysql_5.6_amd64
mysql_5.7: mysql_5.7_both
mysql_8.0: mysql_8.0_both
mysql_8.4: mysql_8.4_both

#TODO: Review this comment and see if we can get rid of it,
# And do we need to add the 8.0.36 again?
# Mysql 8.0 often must be pinned because xtrabackup is not ready for latest 8.0
# So check whether xtrabackup is available for latest 8.0 before changing pin


# Examples:
# make <dbtype>_<dbmajor>_[both|amd64]_<pin>  # pin is optional, often needed for mysql 8.0
# make mariadb_10.3_both VERSION=someversion PUSH=true
# make mysql_8.0_amd64_8.0.36 VERSION=someversion
$(BUILD_TARGETS):
	@echo "building $@";
	export DB_TYPE=$(word 1, $(subst _, ,$@)) && \
	export DB_MAJOR_VERSION=$(word 2, $(subst _, ,$@)) && \
	export ARCH=$(word 3, $(subst _, ,$@)) && \
	export DB_PINNED_VERSION=$(word 4, $(subst _, ,$@)) && \
	case $${ARCH} in \
	  both) \
	    ARCHS="linux/amd64,linux/arm64"; \
	    ;; \
	  amd64 | arm64) \
	    ARCHS="linux/$${ARCH}"; \
	    ;; \
	esac; \
	cmd="./build_image.sh --db-type=$${DB_TYPE} --db-major-version=$${DB_MAJOR_VERSION} --archs=$${ARCHS} --tag=$(VERSION) --docker-args=$(DOCKER_ARGS)" && \
	if [ ! -z $${PUSH} ]; then cmd="$$cmd --push"; fi && \
	if [ ! -z $${DB_PINNED_VERSION} ]; then cmd="$$cmd --db-pinned-version=$${DB_PINNED_VERSION}"; fi && \
	echo $${cmd} && \
	$${cmd}

test: build $(TEST_TARGETS)

push:
	@echo "To push all images, use make PUSH=true VERSION=<tag>, to push a specific image, use something like make mariadb_10.3 PUSH=true VERSION=<tag>" && exit 1

# make mariadb_10.2_test mysql_8.0_test VERSION=20210213_db_image_no_sudo
# make test VERSION=20210213_db_image_no_sudo # for all
$(TEST_TARGETS):
	@export DB_TYPE=$(word 1, $(subst _, ,$@)) && \
	export DB_MAJOR_VERSION=$(word 2, $(subst _, ,$@)) && \
	export ARCH=$(CURRENT_ARCH) && \
	printf "\n\n\n==== Testing $${DB_TYPE} $${DB_MAJOR_VERSION} $(VERSION)" && \
	./test/test_dbserver.sh $${DB_TYPE} $${DB_MAJOR_VERSION} $(VERSION)

clean:
	@rm -rf VERSION.txt
