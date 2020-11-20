DOCKER_REPO ?= drud/ddev-dbserver

VERSION := $(shell git describe --tags --always --dirty)

.PHONY: build container mariadb_containers mysql_containers multi-arch push test clean

include database-versions

CURRENT_ARCH=$(shell ../get_arch.sh)

# This Makefile explicitly does not include containers/containers_shared.mak,
# So has to explicitly declare anything it might need from there (like SHELL)
SHELL = /bin/bash

container: mariadb_containers mysql_containers

mariadb_containers:
	set -eu -o pipefail; \
	for item in $(subst $\",,$(MARIADB_VERSIONS_$(CURRENT_ARCH))); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MariaDB $${baseversion} with image $${fullversion} ==========\n"; \
		docker buildx build -o type=docker $(DOCKER_ARGS) --build-arg="DBVERSION=$${baseversion}" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${baseversion}:$(VERSION)" . ; \
	done

mysql_containers:
	set -eu -o pipefail ; \
	for item in $(subst $\",,$(MYSQL_VERSIONS_$(CURRENT_ARCH))); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MySQL $${baseversion} with image $${fullversion} ==========\n", ${baseversion}, ${fullversion}; \
		docker buildx build -o type=docker $(DOCKER_ARGS) --build-arg="DBVERSION=$$baseversion" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mysql" -t "drud/ddev-dbserver-mysql-$${baseversion%%%.*}:$(VERSION)" . ; \
	done

# Below has mariadb_containers and mysql_containers as a prerequisite because those build the amd64 images.
# So, the multi-arch approach here is a bit different from the ones in the other Makefiles.
multi-arch: mariadb_containers mysql_containers
	set -eu -o pipefail ; \
	for item in $(subst $\",,$(MARIADB_VERSIONS_arm64)); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MariaDB $${baseversion} with image $${fullversion} ==========\n"; \
		docker buildx build --platform=linux/arm64 $(DOCKER_ARGS) --build-arg="DBVERSION=$${baseversion}" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${baseversion}:$(VERSION)" . ; \
	done	

push:
	set -euo pipefail ;\
	for item in $(subst $\",,$(MARIADB_VERSIONS_amd64)); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		export archs=$$(if [[ "$(subst $\",,$(MARIADB_VERSIONS_arm64))" == *"$$item"* ]]; then echo "linux/amd64,linux/arm64"; else echo "linux/amd64"; fi); \
		docker buildx build --push --platform $${archs} $(DOCKER_ARGS)  --build-arg="DBVERSION=$${baseversion}" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${baseversion}:$(VERSION)" .; \
	done
	set -euo pipefail ;\
	for item in $(subst $\",,$(MYSQL_VERSIONS_amd64)); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		docker buildx build --push --platform linux/amd64 $(DOCKER_ARGS) --build-arg="DBVERSION=$$baseversion" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mysql" -t "drud/ddev-dbserver-mysql-$${baseversion%%%.*}:$(VERSION)" .; \
	done

test_mariadb: mariadb_containers
	bash ./test/test_dbserver.sh $(VERSION) mariadb # bash execution just for windows make, only mariadb tests

test_mysql: mysql_containers
	bash ./test/test_dbserver.sh $(VERSION) mysql # bash execution just for windows make, only mysql tests

test: container
	bash ./test/test_dbserver.sh $(VERSION) # bash execution just for windows make

clean:
	for item in $(subst $\",,$(MARIADB_VERSIONS_$(CURRENT_ARCH))); do \
		if docker image inspect $(DOCKER_REPO)-mariadb-${item}:$(VERSION) >/dev/null 2>&1; then docker rmi -f $(DOCKER_REPO)-mariadb-$${item}:$(VERSION); fi; \
	done
	for item in $(subst $\",,$(MYSQL_VERSIONS_$(CURRENT_ARCH))); do \
		if docker image inspect $(DOCKER_REPO)-mysql-${item}:$(VERSION) >/dev/null 2>&1; then docker rmi -f $(DOCKER_REPO)-mysql-$${item}:$(VERSION); fi; \
	done
	@rm -rf VERSION.txt
