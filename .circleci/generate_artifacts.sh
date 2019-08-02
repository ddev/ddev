#!/bin/bash
# This script builds ddev artifacts and their sha256 hashes.
# First arg is the artifact directory
# Optional second arg is whether to build xz version of ddev_docker_images.tar

set -o errexit
set -o pipefail
set -o nounset

ARTIFACTS=${1:-/artifacts}
# We only build the xz artifacts if $2 ($BUILD_XZ) is not empty.
BUILD_XZ=${2:-}
BUILD_IMAGE_TARBALLS=${3:-true}
BASE_DIR=$PWD

sudo mkdir -p $ARTIFACTS && sudo chmod 777 $ARTIFACTS
export VERSION=$(git describe --tags --always --dirty)

case "${OSTYPE}" in
darwin*)
  BUILTPATH=.gotmp/bin/darwin_amd64
  ;;
linux*)
  BUILTPATH=.gotmp/bin
  ;;
windows*)
  BUILTPATH=.gotmp/bin/windows_amd64
  ;;
esac

if [ "${BUILD_IMAGE_TARBALLS}" = "true" ]; then

    # Make sure we have all our docker images, and save them in a tarball
    $BUILTPATH/ddev version | awk '/drud\// {print $2;}' >/tmp/images.txt
    for item in $(cat /tmp/images.txt); do
      docker pull $item
    done
    echo "Generating ddev_docker_images.$VERSION.tar.gz"
    docker save -o $ARTIFACTS/ddev_docker_images.$VERSION.tar $(cat /tmp/images.txt)
    gzip --keep $ARTIFACTS/ddev_docker_images.$VERSION.tar
    if [ ! -z "$BUILD_XZ" ] ; then
        echo "Generating ddev_docker_images.$VERSION.tar.xz"
        xz $ARTIFACTS/ddev_docker_images.$VERSION.tar
    fi
fi

# Generate and place extra items like autocomplete
$BUILTPATH/ddev_gen_autocomplete

for dir in .gotmp/bin/darwin_amd64 .gotmp/bin/windows_amd64; do
  cp .gotmp/bin/ddev_bash_completion.sh $dir
done

# Generate macOS tarball/zipball
cd $BASE_DIR/.gotmp/bin/darwin_amd64
tar -czf $ARTIFACTS/ddev_macos.$VERSION.tar.gz ddev ddev_bash_completion.sh

# Generate linux tarball/zipball
cd $BASE_DIR/.gotmp/bin
tar -czf $ARTIFACTS/ddev_linux.$VERSION.tar.gz ddev ddev_bash_completion.sh

# generate windows tarball/zipball
cd $BASE_DIR/.gotmp/bin/windows_amd64
tar -czf $ARTIFACTS/ddev_windows.$VERSION.tar.gz ddev.exe ddev_bash_completion.sh
zip $ARTIFACTS/ddev_windows.$VERSION.zip ddev.exe ddev_bash_completion.sh
if [ -d chocolatey ]; then
    tar -czf $ARTIFACTS/ddev_chocolatey.$VERSION.tar.gz chocolatey
fi

cp ddev_windows_installer*.exe $ARTIFACTS

# Create macOS and Linux homebrew bottles
for os in sierra x86_64_linux ; do
    NO_V_VERSION=${VERSION#v}
    rm -rf /tmp/bottle
    BOTTLE_BASE=/tmp/bottle/ddev/$NO_V_VERSION
    mkdir -p $BOTTLE_BASE/{bin,etc/bash_completion.d}
    cp $BASE_DIR/.gotmp/bin/ddev_bash_completion.sh $BOTTLE_BASE/etc/bash_completion.d/ddev
    if [ "${os}" = "sierra" ]; then cp $BASE_DIR/.gotmp/bin/darwin_amd64/ddev $BOTTLE_BASE/bin ; fi
    if [ "${os}" = "x86_64_linux" ]; then cp $BASE_DIR/.gotmp/bin/ddev $BOTTLE_BASE/bin ; fi
    cp $BASE_DIR/{README.md,LICENSE} $BOTTLE_BASE
    tar -czf $ARTIFACTS/ddev-$NO_V_VERSION.$os.bottle.tar.gz -C /tmp/bottle .
done

# Create the sha256 files
cd $ARTIFACTS
for item in *.*; do
  sha256sum $item > $item.sha256.txt
done
