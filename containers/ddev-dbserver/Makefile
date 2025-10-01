DOCKER_ORG ?= ddev
export DOCKER_ORG
DOCKER_REPO ?= $(DOCKER_ORG)/ddev-dbserver
export DOCKER_REPO

VERSION := $(shell git describe --tags --always --dirty)

.PHONY: build clean test

CURRENT_ARCH=$(shell ../get_arch.sh)

# This Makefile explicitly does not include containers/containers_shared.mk,
# So has to explicitly declare anything it might need from there (like SHELL)
SHELL = /bin/bash

BUILD_TARGETS=mysql_8.4_both mysql_8.0_both mysql_5.7_both mysql_5.6_amd64 mysql_5.5_amd64 mariadb_11.8_both mariadb_11.4_both mariadb_10.11_both mariadb_10.8_both mariadb_10.7_both mariadb_10.6_both mariadb_10.5_both mariadb_10.4_both mariadb_10.3_both mariadb_10.2_both mariadb_10.1_both mariadb_10.0_amd64 mariadb_5.5_amd64
SINGLE_ARCH_TARGETS=mysql_8.4_amd64 mysql_8.4_arm64 mysql_8.0_amd64 mysql_8.0_arm64 mysql_5.7_amd64 mysql_5.7_arm64 mariadb_11.8_amd64 mariadb_11.8_arm64 mariadb_11.4_amd64 mariadb_11.4_arm64 mariadb_10.11_amd64 mariadb_10.11_arm64 mariadb_10.8_amd64 mariadb_10.8_arm64 mariadb_10.7_amd64 mariadb_10.7_arm64 mariadb_10.6_amd64 mariadb_10.6_arm64 mariadb_10.5_amd64 mariadb_10.5_arm64 mariadb_10.4_amd64 mariadb_10.4_arm64 mariadb_10.3_amd64 mariadb_10.3_arm64 mariadb_10.2_amd64 mariadb_10.2_arm64 mariadb_10.1_amd64 mariadb_10.1_arm64
TEST_TARGETS=$(shell if [ "$(CURRENT_ARCH)" = "amd64" ] ; then \
  echo "mysql_8.4_test mysql_8.0_test mysql_5.7_test mysql_5.6_test mysql_5.5_test mariadb_11.8_test mariadb_11.4_test mariadb_10.11_test mariadb_10.8_test mariadb_10.7_test mariadb_10.6_test mariadb_10.5_test mariadb_10.4_test mariadb_10.3_test mariadb_10.2_test mariadb_10.1_test mariadb_10.0_test mariadb_5.5_test"; \
else \
  echo "mysql_8.4_test mysql_8.0_test mysql_5.7_test mariadb_11.8_test mariadb_11.4_test mariadb_10.11_test mariadb_10.8_test mariadb_10.7_test mariadb_10.6_test mariadb_10.5_test mariadb_10.4_test mariadb_10.3_test mariadb_10.2_test mariadb_10.1_test"; \
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
mariadb_11.8: mariadb_11.8_both

mysql_5.5: mysql_5.5_amd64
mysql_5.6: mysql_5.6_amd64
mysql_5.7: mysql_5.7_both
mysql_8.0: mysql_8.0_both
mysql_8.4: mysql_8.4_both

# Examples:
# make <dbtype>_<dbmajor>_[both|amd64]_<pin>  # pin is optional, often needed for mysql 8.0
# make mariadb_10.3_both VERSION=someversion PUSH=true
# make mysql_8.0_amd64_8.0.36 VERSION=someversion
$(BUILD_TARGETS) $(SINGLE_ARCH_TARGETS):
	@echo "building $@";
	export DB_TYPE=$(word 1, $(subst _, ,$@)) && \
	export DB_MAJOR_VERSION=$(word 2, $(subst _, ,$@)) && \
	export ARCH=$(word 3, $(subst _, ,$@)) && \
	export DB_PINNED_VERSION=$(word 4, $(subst _, ,$@)) && \
	export DOCKER_ORG=${DOCKER_ORG:ddev} && \
	case $${ARCH} in \
	  both) \
	    ARCHS="linux/amd64,linux/arm64"; \
	    ;; \
	  amd64 | arm64) \
	    ARCHS="linux/$${ARCH}"; \
	    ;; \
	esac; \
	cmd="./build_image.sh --db-type=$${DB_TYPE} --db-major-version=$${DB_MAJOR_VERSION} --archs=$${ARCHS} --tag=$(VERSION) --docker-org=$(DOCKER_ORG) --docker-args=$(DOCKER_ARGS)" && \
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
	printf "\n\n\n==== Testing $${DB_TYPE} $${DB_MAJOR_VERSION} $(VERSION) " && \
	./test/test_dbserver.sh $${DB_TYPE} $${DB_MAJOR_VERSION} $(VERSION)

clean:
	@rm -rf VERSION.txt
