#!/bin/bash

# This script is used to install a matching `mysql` client (with `mysqldump`)
# in ddev-webserver
# It should be called with the appropriate mysql version as an argument

set -eu -o pipefail

if [ "${DDEV_DATABASE_FAMILY}" != "mysql" ]; then
  echo "This script is to be used only with a project using mysql" && exit 1
fi
ARCH=${GOARCH}
MYSQL_VERSION=${DDEV_DATABASE#*:}
TARBALL_VERSION=v0.2.1
TARBALL_URL=https://github.com/ddev/mysql-client-build/releases/download/${TARBALL_VERSION}/mysql-${MYSQL_VERSION}-${ARCH}.tar.gz

# set -x

# Remove any existing mariadb installs
sudo apt remove -y mariadb-client-core mariadb-client || true
sudo apt autoremove || true

cd /tmp && curl -L -o /tmp/mysql.tgz --fail -s ${TARBALL_URL}
sudo tar -zxf /tmp/mysql.tgz -C /usr/local/bin ./mysql ./mysqldump
