# Makefile for a standard repo with associated container

##### These variables need to be adjusted in most repositories #####

# This repo's root import path (under GOPATH).
# PKG := github.com/drud/repo_name

# Docker repo for a push
DOCKER_REPO ?= drud/ddev-dbserver

VERSION := $(shell git describe --tags --always --dirty)

build: container

# The format here is majorversion:pinnedimageversion
# This allows to pin the version (as currently required with 8.0.19
MARIADB_VERSIONS_amd64=5.5:5.5 10.0:10.0 10.1:10.1 10.2:10.2 10.3:10.3 10.4:10.4 10.5:10.5
MYSQL_VERSIONS_amd64=5.5:5.5 5.6:5.6 5.7:5.7 8.0:8.0.19
MARIADB_VERSIONS_arm64=10.2:10.2 10.3:10.3 10.4:10.4 10.5:10.5
# MySQL images on Docker Hub currently are amd64 only
MYSQL_VERSIONS_arm64=

include ../containers_shared.mak

container: mariadb_containers mysql_containers

mariadb_containers:
	set -eu -o pipefail; \
	for item in $(MARIADB_VERSIONS_amd64); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MariaDB $${baseversion} with image $${fullversion} ==========\n"; \
		docker buildx build -o type=docker $(DOCKER_ARGS) --build-arg="DBVERSION=$${baseversion}" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${baseversion}:$(VERSION)" . ; \
	done

mysql_containers:
	set -eu -o pipefail ; \
	for item in $(MYSQL_VERSIONS_amd64) ; do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MySQL $${baseversion} with image $${fullversion} ==========\n", ${baseversion}, ${fullversion}; \
		docker buildx build -o type=docker $(DOCKER_ARGS) --build-arg="DBVERSION=$$baseversion" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mysql" -t "drud/ddev-dbserver-mysql-$${baseversion%%%.*}:$(VERSION)" . ; \
	done

# Below has mariadb_containers and mysql_containers as a prerequisite because those build the amd64 images.
# So, the multi-arch approach here is a bit different from the ones in the other Makefiles.
multi-arch: mariadb_containers mysql_containers
	set -eu -o pipefail ; \
	for item in $(MARIADB_VERSIONS_arm64); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		printf "\n\n========== Building MariaDB $${baseversion} with image $${fullversion} ==========\n"; \
		docker buildx build --platform=linux/arm64 $(DOCKER_ARGS) --build-arg="DBVERSION=$${baseversion}" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${baseversion}:$(VERSION)" . ; \
	done	

push:
	set -euo pipefail ;\
	for item in $(MARIADB_VERSIONS_amd64); do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		export archs=$$(if [[ "$(MARIADB_VERSIONS_arm64)" == *"$$item"* ]]; then echo "linux/amd64,linux/arm64"; else echo "linux/amd64"; fi); \
		docker buildx build --push --platform $${archs} $(DOCKER_ARGS)  --build-arg="DBVERSION=$${baseversion}" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mariadb" -t "drud/ddev-dbserver-mariadb-$${baseversion}:$(VERSION)" .; \
	done
	set -euo pipefail ;\
	for item in $(MYSQL_VERSIONS_amd64) ; do \
		export baseversion=$$(echo $$item | awk -F ':' '{print $$1}'); \
		export fullversion=$$(echo $$item | awk -F ':' '{print $$2}'); \
		docker buildx build --push --platform linux/amd64 $(DOCKER_ARGS) --build-arg="DBVERSION=$$baseversion" --build-arg="PARENTIMAGEVERSION=$$fullversion" --build-arg="DBTYPE=mysql" -t "drud/ddev-dbserver-mysql-$${baseversion%%%.*}:$(VERSION)" .; \
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
	@rm -rf VERSION.txt
