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
# Add all required MariaDB versions here
if [ "${MARIADB_VERSION}" != "11.4" ]; then
  echo "This script is not intended to run with mariadb:${MARIADB_VERSION}" && exit 0
fi

# Configure the correct repository for mariadb
set -x
timeout 30 mariadb_repo_setup --mariadb-server-version="mariadb-${MARIADB_VERSION}"
rm -f /etc/apt/sources.list.d/mariadb.list.old_*

# Install the mariadb-client and mysql symlinks
# MariaDB 11.x moved MySQL symlinks into separate packages
apt-get install -y mariadb-client mariadb-client-compat
