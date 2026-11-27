#!/usr/bin/env bash
# This script builds ddev artifacts and their sha256 hashes.
# First arg is the artifact directory

set -o errexit
set -o pipefail
set -o nounset

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


# mkcert comes from `make mkcert`, already in each .gotmp/bin platform directory.
# The GUI app is built in other jobs, so it is added only where present:
# ddev-gui next to ddev, and DDEV.app.zip from darwin_universal.

# archive <os>-<arch> <dir> <files...>
archive() {
  local name=$1 dir=$2
  shift 2
  local files=("$@")
  local extra=()
  for f in ddev-gui ddev-gui.exe; do
    if [ -f "$BASE_DIR/$dir/$f" ]; then files+=("$f"); fi
  done
  if [[ $name == macos-* ]] && [ -f "$BASE_DIR/.gotmp/bin/darwin_universal/DDEV.app.zip" ]; then
    extra=(-C "$BASE_DIR/.gotmp/bin/darwin_universal" DDEV.app.zip)
  fi
  tar -czf "$ARTIFACTS/ddev_$name.$VERSION.tar.gz" -C "$BASE_DIR/$dir" "${files[@]}" ${extra[@]+"${extra[@]}"}
}

archive macos-amd64 .gotmp/bin/darwin_amd64 ddev ddev-hostname mkcert
archive macos-arm64 .gotmp/bin/darwin_arm64 ddev ddev-hostname mkcert
archive linux-amd64 .gotmp/bin/linux_amd64 ddev ddev-hostname mkcert
archive linux-arm64 .gotmp/bin/linux_arm64 ddev ddev-hostname mkcert
# mkcert.exe is already here and signed
archive windows-amd64 .gotmp/bin/windows_amd64 ddev.exe ddev-hostname.exe mkcert.exe
archive windows-arm64 .gotmp/bin/windows_arm64 ddev.exe ddev-hostname.exe mkcert.exe
