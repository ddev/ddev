#!/usr/bin/env bash

# This script is used to install a matching mariadb-client
# in ddev-webserver
# It should be called with the appropriate mariadb version as an argument
# This script is intended to be run with root privileges (normally in a docker build phase)
# Can be tested with "ddev exec sudo DDEV_DATABASE=mariadb:11.4 mariadb-client-install.sh"

set -eu -o pipefail

DDEV_DATABASE_FAMILY=${DDEV_DATABASE%:*}
if [ "${DDEV_DATABASE_FAMILY}" != "mariadb" ]; then
  echo "This script is to be used only with a project using mariadb" >&2
  exit 1
fi
MARIADB_VERSION=${DDEV_DATABASE#*:}

# Use MariaDB 10.11 client for server versions below 11.x, because 11.4+ clients
# enforce SSL verification that older servers don't support, causing connection failures.
if [ "${MARIADB_VERSION%%.*}" -lt 11 ]; then
  MARIADB_VERSION="10.11"
fi

sed -i "s|^URIs:.*|URIs: https://archive.mariadb.org/mariadb-${MARIADB_VERSION}/repo/debian|" /etc/apt/sources.list.d/mariadb-archive.sources

# Select the appropriate Debian suite based on MariaDB version availability.
# Note: Versions below 10.11 require libssl1.1 and will not work on current Debian
# without creating a FrankenDebian system (https://wiki.debian.org/DontBreakDebian).
# The logic for these older versions is provided as-is for those who may need it.
# Search for CHANGE_MARIADB_CLIENT to update related code.
if [ "${MARIADB_VERSION}" = "11.8" ]; then
  sed -i "s/^Suites:.*/Suites: trixie/" /etc/apt/sources.list.d/mariadb-archive.sources
elif [ "${MARIADB_VERSION}" = "10.11" ] || [ "${MARIADB_VERSION}" = "11.4" ]; then
  sed -i "s/^Suites:.*/Suites: bookworm/" /etc/apt/sources.list.d/mariadb-archive.sources
elif [ "${MARIADB_VERSION}" = "10.5" ] || [ "${MARIADB_VERSION}" = "10.6" ] || [ "${MARIADB_VERSION}" = "10.7" ] || [ "${MARIADB_VERSION}" = "10.8" ]; then
  sed -i "s/^Suites:.*/Suites: bullseye/" /etc/apt/sources.list.d/mariadb-archive.sources
elif [ "${MARIADB_VERSION}" = "10.1" ] || [ "${MARIADB_VERSION}" = "10.2" ] || [ "${MARIADB_VERSION}" = "10.3" ] || [ "${MARIADB_VERSION}" = "10.4" ]; then
  sed -i "s/^Suites:.*/Suites: stretch/" /etc/apt/sources.list.d/mariadb-archive.sources
fi

# Run "apt-get update" manually only for mariadb and debian repos to make it faster
apt-get update -o Acquire::Retries=5 -o Dir::Etc::sourcelist="sources.list.d/mariadb-archive.sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0"
apt-get update -o Acquire::Retries=5 -o Dir::Etc::sourcelist="sources.list.d/debian.sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0"

# Install the mariadb-client
export DEBIAN_FRONTEND=noninteractive
apt-get install --allow-downgrades --no-install-recommends --no-install-suggests -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -y mariadb-client
