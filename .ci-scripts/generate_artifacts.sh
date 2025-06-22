#!/usr/bin/env bash
# This script builds ddev artifacts and their sha256 hashes.
# First arg is the artifact directory

set -o errexit
set -o pipefail
set -o nounset

MKCERT_VERSION=v1.4.4
BUILD_IMAGE_TARBALLS=${BUILD_IMAGE_TARBALLS:-false}

ARTIFACTS=${1:-/artifacts}
BASE_DIR=$PWD

mkdir -p $ARTIFACTS || (sudo mkdir -p $ARTIFACTS && sudo chmod 777 $ARTIFACTS)
export VERSION=$(git describe --tags --always --dirty)

# 2022-03-10: The image tarballs were for ddev/quicksprint, which is currently in retirement
# If the version does not have a dash in it, it's not prerelease,
# so build image tarballs
#if [ "${VERSION}" = "${VERSION%%-*}" ]; then
#  BUILD_IMAGE_TARBALLS=true
#fi

BUILTPATH=.gotmp/bin/$(go env GOOS)_$(go env GOARCH)

if [ "${BUILD_IMAGE_TARBALLS}" = "true" ]; then
  ${BUILTPATH}/ddev poweroff
  # Make sure we have all our docker images, and save them in a tarball
  $BUILTPATH/ddev version | awk '/(ddev|postgres|traefik)\// {print $2;}' >/tmp/images.txt
  for arch in amd64 arm64; do
    for item in $(cat /tmp/images.txt); do
      docker pull --platform=linux/$arch $item
    done
    echo "Generating ddev_docker_images.${arch}.${VERSION}.tar"
    docker save -o $ARTIFACTS/ddev_docker_images.${arch}.${VERSION}.tar $(cat /tmp/images.txt)
    echo "Generating ddev_docker_images.${arch}.${VERSION}.tar.xz"
    xz $ARTIFACTS/ddev_docker_images.${arch}.$VERSION.tar
  done
  # Untag the pulled images in case they're the wrong platform for
  # where we're executing this.
  for item in $(cat /tmp/images.txt); do
    docker rmi $item
  done
fi


echo "Using github as source for mkcert binaries for tarballs"

# Generate macOS-amd64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/darwin_amd64 >/dev/null
curl --fail -JL -s -o mkcert "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-darwin-amd64" && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_macos-amd64.$VERSION.tar.gz ddev ddev_hostname mkcert
popd >/dev/null

# Generate macOS-arm64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/darwin_arm64 >/dev/null
curl --fail -JL -s -o mkcert "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-darwin-arm64" && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_macos-arm64.$VERSION.tar.gz ddev ddev_hostname mkcert
popd >/dev/null

# Generate linux-amd64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/linux_amd64 >/dev/null
curl --fail -JL -s -o mkcert "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-amd64" && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_linux-amd64.$VERSION.tar.gz ddev ddev_hostname ddev_hostname.exe mkcert
popd >/dev/null

# Generate linux-arm64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/linux_arm64 >/dev/null
curl --fail -JL -s -o mkcert "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-arm64" && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_linux-arm64.$VERSION.tar.gz ddev ddev_hostname ../windows_arm64/ddev_hostname.exe mkcert
popd >/dev/null

# generate windows-amd64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/windows_amd64 >/dev/null
curl --fail -JL -s -o mkcert.exe "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-windows-amd64.exe"
tar -czf $ARTIFACTS/ddev_windows-amd64.$VERSION.tar.gz ddev.exe ddev_hostname.exe mkcert.exe
popd >/dev/null

# generate windows-arm64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/windows_arm64 >/dev/null
curl --fail -JL -s -o mkcert.exe "https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-windows-arm64.exe"
tar -czf $ARTIFACTS/ddev_windows-arm64.$VERSION.tar.gz ddev.exe ddev_hostname.exe mkcert.exe
popd >/dev/null


