#!/usr/bin/env bash

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

# Configure the correct repository for mariadb
set -x
log-stderr.sh --timeout "${START_SCRIPT_TIMEOUT:-30}" mariadb_repo_setup --mariadb-server-version="mariadb-${MARIADB_VERSION}" --skip-maxscale --skip-tools --os-type=debian --os-version=bookworm || exit $?
rm -f /etc/apt/sources.list.d/mariadb.list.old_*
# --skip-key-import flag doesn't download the existing key again and omits "apt-get update",
# so we can run "apt-get update" manually only for mariadb and debian repos to make it faster
timeout "${START_SCRIPT_TIMEOUT:-30}" apt-get update -o Dir::Etc::sourcelist="sources.list.d/mariadb.list" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || exit $?
timeout "${START_SCRIPT_TIMEOUT:-30}" apt-get update -o Dir::Etc::sourcelist="sources.list.d/debian.sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || exit $?
# Install the mariadb-client
export DEBIAN_FRONTEND=noninteractive
if apt-cache search mariadb-client-compat 2>/dev/null | grep -q mariadb-client-compat; then
  # MariaDB 11.x moved MySQL symlinks into a separate package
  log-stderr.sh apt-get install --no-install-recommends --no-install-suggests -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -y mariadb-client mariadb-client-compat || exit $?
else
  log-stderr.sh apt-get install --no-install-recommends --no-install-suggests -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -y mariadb-client || exit $?
fi
