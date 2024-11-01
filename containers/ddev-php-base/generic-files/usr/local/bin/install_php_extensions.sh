#!/bin/bash

set -eu -o pipefail

if [ -z "${1:-}" ] || [ -z "${2:-}" ]; then
    echo "Usage: $0 <PHP_VERSION> <ARCH>"
    exit 1
fi

if ! command -v yq >/dev/null ; then
  echo "yq is required, please install it" && exit 2
fi

# Assign PHP version and architecture
PHP_VERSION=$1
ARCH=$2

# Retrieve and format the list of packages for the specified PHP version and architecture
# Uses `yq` to get the proper list of extensions for the PHP version and architecture.
# The awk transforms the list from something like "cli common fpm" to
# somethign like "phpX.X-cli phpX.X-common phpX.X-fpm"
pkgs=$(yq ".${PHP_VERSION//.}.${ARCH} | join(\" \")" /etc/php-packages.yaml | awk -v v="$PHP_VERSION" 'BEGIN {RS=" ";} {printf "%s-%s ", v, $0}')

# Echo the packages to be installed for logging
echo "Installing packages for PHP ${PHP_VERSION} on ${ARCH}: $pkgs"

apt-get -qq install --no-install-recommends --no-install-suggests -y $pkgs || exit $?
