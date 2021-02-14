DOCKER_REPO ?= drud/ddev-dbserver

VERSION := $(shell git describe --tags --always --dirty)

.PHONY: build container mariadb_containers mysql_containers multi-arch push test clean

include database-versions

CURRENT_ARCH=$(shell ../get_arch.sh)

# This Makefile explicitly does not include containers/containers_shared.mak,
# So has to explicitly declare anything it might need from there (like SHELL)
SHELL = /bin/bash

container: mariadb_containers mysql_containers

mariadb_containers: mariadb_5.5 mariadb_10.0 mariadb_10.1 mariadb_10.2 mariadb_10.3 mariadb_10.4 mariadb_10.5
mariadb_5.5: mariadb_5.5_amd64
mariadb_10.0: mariadb_10.0_amd64
mariadb_10.1: mariadb_10.1_amd64
mariadb_10.2: mariadb_10.2_both
mariadb_10.3: mariadb_10.3_both
mariadb_10.4: mariadb_10.4_both
mariadb_10.5: mariadb_10.5_both

mysql_containers: mysql_5.5 mysql_5.6 mysql_5.7 mysql_8.0
mysql_5.5: mysql_5.5_amd64
mysql_5.6: mysql_5.6_amd64
mysql_5.7: mysql_5.7_amd64
mysql_8.0: mysql_8.0_amd64

TARGETS=mariadb_5.5_amd64 mariadb_10.0_amd64 mariadb_10.1_amd64 mariadb_10.2_both mariadb_10.3_both .3_amd64 mariadb_10.4_both mariadb_10.5_both mysql_5.5_amd64 mysql_5.6_amd64 mysql_5.7_amd64 mysql_8.0_amd64

$(TARGETS):
	@echo "building $@";
	@export DB_TYPE=$(word 1, $(subst _, ,$@)) && \
	export DB_MAJOR_VERSION=$(word 2, $(subst _, ,$@)) && \
	export ARCH=$(word 3, $(subst _, ,$@)) && \
	if [ $${ARCH} = "both" ]; then ARCHS="linux/arm64,linux/amd64"; else ARCHS="linux/amd64"; fi && \
	cmd="./build_image.sh --db-major-version $${DB_MAJOR_VERSION} --archs=$${ARCHS} --tag=$(VERSION)" && \
	if [ ! -z $${PUSH} ]; then cmd="$$cmd --push"; fi && \
	echo $${cmd} && \
	$${cmd}


test_mariadb:
	bash ./test/test_dbserver.sh $(VERSION) mariadb # bash execution just for windows make, only mariadb tests

test_mysql:
	bash ./test/test_dbserver.sh $(VERSION) mysql # bash execution just for windows make, only mysql tests

test:
	bash ./test/test_dbserver.sh $(VERSION) # bash execution just for windows make

clean:
	@rm -rf VERSION.txt
