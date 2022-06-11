#!/bin/bash

# This builds the windows installer for ddev
# $1 must be the TARGET_DIRNAME, the directory into which we'll put the installer
# $2 must be the VERSION

set -x
set -eu -o pipefail

if [ $# != 2 ]; then echo "Need 2 args, TARGET_DIRNAME and VERSION" && exit 2; fi
TARGET_DIRNAME=$1
VERSION=$2

MKCERT_VERSION=v1.4.6

# Get mkcert and license
curl --fail -sSL -o ${TARGET_DIRNAME}/mkcert.exe https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-windows-amd64.exe
curl --fail -sSL -o ${TARGET_DIRNAME}/mkcert_license.txt -O https://raw.githubusercontent.com/drud/mkcert/master/LICENSE

# Get sudo license
curl --fail -sSL -o "${TARGET_DIRNAME}/sudo_license.txt" "https://raw.githubusercontent.com/gerardog/gsudo/master/LICENSE.txt"

# Build installer with makensis
makensis -DVERSION=${VERSION} -DTARGET=${TARGET_DIRNAME} winpkg/ddev.nsi

