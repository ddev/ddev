#!/bin/bash

# This script is used to install a matching `mysql` client (with `mysqldump`)
# in ddev-webserver
# It should be called with the appropriate mysql version as an argument
# This script is intended to be run with root privileges (normally in a docker build phase)

set -eu -o pipefail

DDEV_DATABASE_FAMILY=${DDEV_DATABASE%:*}
if [ "${DDEV_DATABASE_FAMILY}" != "mysql" ]; then
  echo "This script is to be used only with a project using mysql" && exit 1
fi
ARCH=$(dpkg --print-architecture)
MYSQL_VERSION=${DDEV_DATABASE#*:}
# For MySQL 5.6 and 5.5 we can't build the client, but the 5.7 client is probably as good as it gets
if [ "${MYSQL_VERSION}" = "5.6" ] || [ "${MYSQL_VERSION}" = "5.5" ]; then
  MYSQL_VERSION="5.7"
fi

TARBALL_VERSION=v0.2.3
TARBALL_URL=https://github.com/ddev/mysql-client-build/releases/download/${TARBALL_VERSION}/mysql-${MYSQL_VERSION}-${ARCH}.tar.gz

# Install the related mysql client if available
set -x
cd /tmp && log-stderr.sh --timeout 30 curl -L -o /tmp/mysql.tgz --fail -s ${TARBALL_URL}
tar -zxf /tmp/mysql.tgz -C /usr/local/bin && rm -f /tmp/mysql.tgz

# Remove any existing mariadb installs
apt-get remove -y mariadb-client-core mariadb-client || true
apt-get autoremove -y || true
