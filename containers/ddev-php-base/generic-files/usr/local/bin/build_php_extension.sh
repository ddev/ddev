#!/usr/bin/env bash

set -eu -o pipefail

if [ "$#" -ne 4 ] && [ "$#" -ne 5 ] && [ "$#" -ne 6 ]; then
  echo "Usage: $0 <PHP_VERSION> <EXTENSION_NAME> <EXTENSION_VERSION> <EXTENSION_FILE> [<BUILD_PACKAGES>] [<CONFIGURE_OPTIONS>]"
  exit 1
fi

# 8.0
PHP_VERSION=${1/php/}
# xdebug
EXTENSION_NAME=$2
# 3.2.2, latest
EXTENSION_VERSION=$3
# /usr/lib/php/20210902/xdebug.so
# The dates from /usr/lib/php/YYYYMMDD/ represent PHP API versions https://unix.stackexchange.com/a/591771
EXTENSION_FILE=$4
# Optional build packages
# Required for some extensions
BUILD_PACKAGES="${5:-}"
# Optional configureoptions:
# How to use --configureoptions https://stackoverflow.com/a/72981491/8097891
# See https://salsa.debian.org/php-team/pecl for package.xml files with <configureoption>
# Pass "-i" for interactive mode
CONFIGURE_OPTIONS="${6:-}"

# install pecl
if ! command -v pecl >/dev/null 2>&1 || [ "$(dpkg -l | grep "php${PHP_VERSION}-dev")" = "" ] || [ "${BUILD_PACKAGES}" != "" ]; then
  echo "Installing pecl to build php${PHP_VERSION}-${EXTENSION_NAME}"
  timeout "${START_SCRIPT_TIMEOUT:-30}" apt-get update -o Dir::Etc::sourcelist="sources.list.d/php.list" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || true
  timeout "${START_SCRIPT_TIMEOUT:-30}" apt-get update -o Dir::Etc::sourcelist="sources.list.d/debian.sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" || true
  DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends --no-install-suggests -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -y build-essential php-pear "php${PHP_VERSION}-dev" ${BUILD_PACKAGES} || exit $?
fi

if [ -f "${EXTENSION_FILE}" ]; then
  echo "Moving existing ${EXTENSION_FILE} to ${EXTENSION_FILE}.bak"
  mv "${EXTENSION_FILE}" "${EXTENSION_FILE}.bak"
fi

if [ "${EXTENSION_VERSION}" = "latest" ]; then
  PECL_EXTENSION="${EXTENSION_NAME}"
else
  PECL_EXTENSION="${EXTENSION_NAME}-${EXTENSION_VERSION}"
fi

timeout "${START_SCRIPT_TIMEOUT:-30}" pecl channel-update pecl.php.net || true

echo "Building php${PHP_VERSION}-${EXTENSION_NAME}..."

if [ "${CONFIGURE_OPTIONS}" = "-i" ]; then
  echo "pecl -d php_suffix=\"${PHP_VERSION}\" install -f \"${PECL_EXTENSION}\""
  pecl -d php_suffix="${PHP_VERSION}" install -f "${PECL_EXTENSION}" || true
elif [ "${CONFIGURE_OPTIONS}" != "" ]; then
  echo "pecl -d php_suffix=\"${PHP_VERSION}\" install --configureoptions=\"${CONFIGURE_OPTIONS}\" -f \"${PECL_EXTENSION}\""
  pecl -d php_suffix="${PHP_VERSION}" install --configureoptions="${CONFIGURE_OPTIONS}" -f "${PECL_EXTENSION}" || true
else
  echo "yes '' | pecl -d php_suffix=\"${PHP_VERSION}\" install -f \"${PECL_EXTENSION}\""
  (yes '' | pecl -d php_suffix="${PHP_VERSION}" install -f "${PECL_EXTENSION}") || true
fi

# PECL does not allow to install multiple versions of extension at the same time,
# use `rm -f /usr/share/php/.registry/.channel.pecl.php.net/extension.reg` to make it forget about another version.
rm -f "/usr/share/php/.registry/.channel.pecl.php.net/${EXTENSION_NAME}.reg"

if [ ! -f "${EXTENSION_FILE}" ]; then
  echo "Failed to build ${EXTENSION_FILE}"

  if [ -f "${EXTENSION_FILE}.bak" ]; then
    echo "Restoring previously existing file ${EXTENSION_FILE}"
    mv "${EXTENSION_FILE}.bak" "${EXTENSION_FILE}"
  fi

  exit 2
fi

# Used as an example
PHP_DEFAULT_VERSION=8.3
echo "Done building php${PHP_VERSION}-${EXTENSION_NAME} to ${EXTENSION_FILE}"
echo "If this is a multistage build, add: 'COPY --from=ddev-php-extension-build ${EXTENSION_FILE} ${EXTENSION_FILE}'"
echo "If there is no /etc/php/${PHP_VERSION}/mods-available/${EXTENSION_NAME}.ini file, add: 'RUN cp /etc/php/${PHP_DEFAULT_VERSION}/mods-available/${EXTENSION_NAME}.ini /etc/php/${PHP_VERSION}/mods-available/${EXTENSION_NAME}.ini'"
echo "Some extensions have more files than ${EXTENSION_FILE} and /etc/php/${PHP_VERSION}/mods-available/${EXTENSION_NAME}.ini:"
echo "Check which files should be in php${PHP_VERSION}-${EXTENSION_NAME}: 'dpkg-query -L php${PHP_DEFAULT_VERSION}-${EXTENSION_NAME}'"
echo "If the extension needs to be enabled, add: 'RUN phpenmod -v ${PHP_VERSION} ${EXTENSION_NAME}'"
exit 0

# Examples:
#
# -------------
# Install php8.4-apcu:
# To understand what files you need in "COPY --from", use "dpkg-query -L php8.3-apcu"
# -------------
# FROM base AS ddev-php-extension-build
# ...
# RUN /usr/local/bin/build_php_extension.sh "php8.4" "apcu" "latest" "/usr/lib/php/20240924/apcu.so"
# ...
# FROM base AS ddev-php-base
# ...
# COPY --from=ddev-php-extension-build /usr/lib/php/20240924/apcu.so /usr/lib/php/20240924/apcu.so
# COPY --from=ddev-php-extension-build /usr/include/php/20240924/ext/apcu /usr/include/php/20240924/ext/apcu
# RUN cp /etc/php/8.3/mods-available/apcu.ini /etc/php/8.4/mods-available/apcu.ini
# RUN phpenmod -v 8.4 apcu
# ...
#
# -------------
# Install php8.4-memcached:
# To understand what files you need in "COPY --from", use "dpkg-query -L php8.3-memcached"
# -------------
# FROM base AS ddev-php-extension-build
# ...
# RUN /usr/local/bin/build_php_extension.sh "php8.4" "memcached" "latest" "/usr/lib/php/20240924/memcached.so" "libmemcached-dev zlib1g-dev libssl-dev"
# ...
# FROM base AS ddev-php-base
# ...
# COPY --from=ddev-php-extension-build /usr/lib/php/20240924/memcached.so /usr/lib/php/20240924/memcached.so
# RUN cp /etc/php/8.3/mods-available/memcached.ini /etc/php/8.4/mods-available/memcached.ini
# RUN phpenmod -v 8.4 memcached
# ...
#
# -------------
# Install php8.4-redis:
# To understand what files you need in "COPY --from", use "dpkg-query -L php8.3-redis"
# -------------
# FROM base AS ddev-php-extension-build
# ...
# RUN /usr/local/bin/build_php_extension.sh "php8.4" "redis" "latest" "/usr/lib/php/20240924/redis.so"
# ...
# FROM base AS ddev-php-base
# ...
# COPY --from=ddev-php-extension-build /usr/lib/php/20240924/redis.so /usr/lib/php/20240924/redis.so
# RUN cp /etc/php/8.3/mods-available/redis.ini /etc/php/8.4/mods-available/redis.ini
# RUN phpenmod -v 8.4 redis
# ...
#
# -------------
# Install php8.4-xdebug (3.4.0beta1 comes from https://pecl.php.net/package/xdebug):
# To understand what files you need in "COPY --from", use "dpkg-query -L php8.3-xdebug"
# -------------
# FROM base AS ddev-php-extension-build
# ...
# RUN /usr/local/bin/build_php_extension.sh "php8.4" "xdebug" "3.4.0beta1" "/usr/lib/php/20240924/xdebug.so"
# ...
# FROM base AS ddev-php-base
# ...
# COPY --from=ddev-php-extension-build /usr/lib/php/20240924/xdebug.so /usr/lib/php/20240924/xdebug.so
# ...
