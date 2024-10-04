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
  # Configure the correct repository for mariadb
  set -x
  log-stderr.sh --timeout 30 mariadb_repo_setup --mariadb-server-version="mariadb-${MARIADB_VERSION}" --skip-maxscale --skip-tools --skip-key-import || exit 2
  rm -f /etc/apt/sources.list.d/mariadb.list.old_*
  # --skip-key-import flag doesn't download the existing key again and omits "apt-get -qq update",
  # so we can run "apt-get -qq update" manually only for mariadb repo to make it faster
  apt-get -qq update -o Dir::Etc::sourcelist="sources.list.d/mariadb.list" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || exit 3
  # Install the mariadb-client and MySQL symlinks
  # MariaDB 11.x moved MySQL symlinks into a separate package
  apt-get install -y mariadb-client mariadb-client-compat || exit 4
else
  echo "This script is not intended to run with mariadb:${MARIADB_VERSION}" && exit 5
fi
