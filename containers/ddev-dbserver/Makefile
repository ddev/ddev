DOCKER_REPO ?= drud/ddev-dbserver

VERSION := $(shell git describe --tags --always --dirty)

.PHONY: build clean test

CURRENT_ARCH=$(shell arch)

# This Makefile explicitly does not include containers/containers_shared.mak,
# So has to explicitly declare anything it might need from there (like SHELL)
SHELL = /bin/bash

BUILD_TARGETS=mariadb_5.5_amd64 mariadb_10.0_amd64 mariadb_10.1_amd64 mariadb_10.2_both mariadb_10.3_both mariadb_10.4_both mariadb_10.5_both mysql_5.5_amd64 mysql_5.6_amd64 mysql_5.7_amd64 mysql_8.0_amd64
TEST_TARGETS=$(shell if [ "$(CURRENT_ARCH)" = "amd64" ] ; then \
	echo "mariadb_5.5_test mariadb_10.0_test mariadb_10.1_test mariadb_10.2_test mariadb_10.3_test mariadb_10.4_test mariadb_10.5_test mysql_5.5_test mysql_5.6_test mysql_5.7_test mysql_8.0_test"; \
	else \
	  echo "mariadb_10.2_test mariadb_10.3_test mariadb_10.4_test mariadb_10.5_test"; \
  	fi )

build: $(BUILD_TARGETS)

mariadb_5.5: mariadb_5.5_amd64
mariadb_10.0: mariadb_10.0_amd64
mariadb_10.1: mariadb_10.1_amd64
mariadb_10.2: mariadb_10.2_both
mariadb_10.3: mariadb_10.3_both
mariadb_10.4: mariadb_10.4_both
mariadb_10.5: mariadb_10.5_both

mysql_5.5: mysql_5.5_amd64
mysql_5.6: mysql_5.6_amd64
mysql_5.7: mysql_5.7_amd64
mysql_8.0: mysql_8.0_amd64


# Examples:
# make mariadb_10.3_both VERSION=someversion PUSH=true
# make mysql_8.0_amd64 VERSION=someversion
$(BUILD_TARGETS):
	@echo "building $@";
	export DB_TYPE=$(word 1, $(subst _, ,$@)) && \
	export DB_MAJOR_VERSION=$(word 2, $(subst _, ,$@)) && \
	export ARCH=$(word 3, $(subst _, ,$@)) && \
	if [ $${ARCH} = "both" ]; then ARCHS="linux/amd64,linux/arm64"; else ARCHS="linux/amd64"; fi && \
	cmd="./build_image.sh --db-type=$${DB_TYPE} --db-major-version $${DB_MAJOR_VERSION} --archs=$${ARCHS} --tag=$(VERSION)" && \
	if [ ! -z $${PUSH} ]; then cmd="$$cmd --push"; fi && \
	echo $${cmd} && \
	$${cmd}

test: build $(TEST_TARGETS)

# make mariadb_10.2_test mysql_8.0_test
# make test # for all
$(TEST_TARGETS):
	@export DB_TYPE=$(word 1, $(subst _, ,$@)) && \
	export DB_MAJOR_VERSION=$(word 2, $(subst _, ,$@)) && \
	export ARCH=$(CURRENT_ARCH) && \
	./test/test_dbserver.sh $${DB_TYPE} $${DB_MAJOR_VERSION} $(VERSION)

clean:
	@rm -rf VERSION.txt
