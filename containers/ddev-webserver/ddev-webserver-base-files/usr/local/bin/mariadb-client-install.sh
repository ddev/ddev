#!/bin/bash

# This script is used to install a matching mariadb-client
# in ddev-webserver
# It should be called with the appropriate mariadb version as an argument
# This script is intended to be run with root privileges (normally in a docker build phase)

set -eu -o pipefail

DDEV_DATABASE_FAMILY=${DDEV_DATABASE%:*}
if [ "${DDEV_DATABASE_FAMILY}" != "mariadb" ]; then
  echo "This script is to be used only with a project using mariadb" && exit 1
fi
MARIADB_VERSION=${DDEV_DATABASE#*:}

# Search for CHANGE_MARIADB_CLIENT to update related code.
# Add MariaDB versions that can have their own client here:
if [ "${MARIADB_VERSION}" = "11.4" ]; then
  set -x
  # Configure the correct repository for mariadb
  DEFAULT_MARIADB_VERSION="10.11"
  sed -i "s|${DEFAULT_MARIADB_VERSION}|${MARIADB_VERSION}|g" /etc/apt/sources.list.d/mariadb.list
  # update only mariadb.list to make it faster
  timeout 30 apt-get update -o Dir::Etc::sourcelist="sources.list.d/mariadb.list" \
    -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || exit 2
  # Install the mariadb-client and mysql symlinks
  # MariaDB 11.x moved MySQL symlinks into separate packages
  apt-get install -y mariadb-client mariadb-client-compat || exit 3
else
  echo "This script is not intended to run with mariadb:${MARIADB_VERSION}" && exit 4
fi
