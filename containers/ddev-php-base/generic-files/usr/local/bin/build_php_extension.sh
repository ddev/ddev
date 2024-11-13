#!/usr/bin/env bash

set -eu -o pipefail

if [ "$#" -ne 4 ]; then
  echo "Usage: $0 <PHP_VERSION> <EXTENSION_NAME> <EXTENSION_VERSION> <EXTENSION_FILE>"
  exit 1
fi

# 8.0
PHP_VERSION=$1
# xdebug
EXTENSION_NAME=$2
# 3.2.2
EXTENSION_VERSION=$3
# /usr/lib/php/20210902/xdebug.so
# The dates from /usr/lib/php/YYYYMMDD/ represent PHP API versions https://unix.stackexchange.com/a/591771
EXTENSION_FILE=$4

# install pecl
if ! command -v pecl >/dev/null 2>&1 || [ "$(dpkg -l | grep "php${PHP_VERSION}-dev")" = "" ]; then
  echo "Installing pecl to build php${PHP_VERSION}-${EXTENSION_NAME}"
  apt-get -qq update -o Dir::Etc::sourcelist="sources.list.d/php.list" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || true
  apt-get -qq update -o Dir::Etc::sourcelist="sources.list.d/debian.sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || true
  DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends --no-install-suggests -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -y build-essential php-pear "php${PHP_VERSION}-dev" || exit $?
fi

if [ -f "${EXTENSION_FILE}" ]; then
  echo "Moving existing ${EXTENSION_FILE} to ${EXTENSION_FILE}.bak"
  mv "${EXTENSION_FILE}" "${EXTENSION_FILE}.bak"
fi

# PECL does not allow to install multiple versions of extension at the same time,
# use `rm -f /usr/share/php/.registry/.channel.pecl.php.net/xdebug.reg` to make it forget about another version.
(pecl channel-update pecl.php.net && \
  echo "Building php${PHP_VERSION}-${EXTENSION_NAME}..." && \
  pecl -d php_suffix="${PHP_VERSION}" install -f "${EXTENSION_NAME}-${EXTENSION_VERSION}" >/dev/null && \
  rm -f /usr/share/php/.registry/.channel.pecl.php.net/xdebug.reg) || true

if [ ! -f "${EXTENSION_FILE}" ]; then
  echo "Failed to build ${EXTENSION_FILE}"

  if [ -f "${EXTENSION_FILE}.bak" ]; then
    echo "Restoring previously existing file ${EXTENSION_FILE}"
    mv "${EXTENSION_FILE}.bak" "${EXTENSION_FILE}"
  fi

  exit 2
fi

echo "Done building php${PHP_VERSION}-${EXTENSION_NAME} to ${EXTENSION_FILE}"
exit 0
