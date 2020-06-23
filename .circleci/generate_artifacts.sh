#!/bin/bash
# This script builds ddev artifacts and their sha256 hashes.
# First arg is the artifact directory

set -o errexit
set -o pipefail
set -o nounset

MKCERT_VERSION=v1.4.0
BUILD_IMAGE_TARBALLS=false

ARTIFACTS=${1:-/artifacts}
BASE_DIR=$PWD

mkdir -p $ARTIFACTS || (sudo mkdir -p $ARTIFACTS && sudo chmod 777 $ARTIFACTS)
export VERSION=$(git describe --tags --always --dirty)

# If the version does not have a dash in it, it's not prerelease,
# so build image tarballs
if [ "${VERSION}" = "${VERSION%%-*}" ]; then
    BUILD_IMAGE_TARBALLS=true
fi

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
    $BUILTPATH/ddev version | awk '/(drud|phpmyadmin)\// {print $2;}' >/tmp/images.txt
    # Quicksprint is the only known consumer of this tarball, and Drupal 9 needs non-default mariadb 10.3
    $BUILTPATH/ddev version | awk ' $1 == "db" { sub(/mariadb-10.2/, "mariadb-10.3"); print $2 }' >>/tmp/images.txt
    for item in $(cat /tmp/images.txt); do
      docker pull $item
    done
    echo "Generating ddev_docker_images.$VERSION.tar"
    docker save -o $ARTIFACTS/ddev_docker_images.$VERSION.tar $(cat /tmp/images.txt)
    echo "Generating ddev_docker_images.$VERSION.tar.xz"
    xz $ARTIFACTS/ddev_docker_images.$VERSION.tar
fi

# Generate and place extra items like autocomplete
$BUILTPATH/ddev_gen_autocomplete

# The completion scripts get placed into the linux build dir (.gotmp/bin)
# So now copy them into the real build directory
for dir in .gotmp/bin/darwin_amd64 .gotmp/bin/windows_amd64; do
  cp .gotmp/bin/ddev_*completion* $dir
done

# Generate macOS tarball/zipball
cd $BASE_DIR/.gotmp/bin/darwin_amd64
curl -sSL -o mkcert https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-darwin-amd64 && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_macos.$VERSION.tar.gz ddev *completion*.sh mkcert

# Generate linux tarball/zipball
cd $BASE_DIR/.gotmp/bin
curl -sSL -o mkcert https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-amd64 && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_linux.$VERSION.tar.gz ddev *completion*.sh mkcert

# generate windows tarball/zipball
cd $BASE_DIR/.gotmp/bin/windows_amd64
curl -sSL -o mkcert.exe https://github.com/FiloSottile/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-windows-amd64.exe
tar -czf $ARTIFACTS/ddev_windows.$VERSION.tar.gz ddev.exe *completion*.sh mkcert.exe
zip $ARTIFACTS/ddev_windows.$VERSION.zip ddev.exe *completion*.sh
if [ -d chocolatey ]; then
    tar -czf $ARTIFACTS/ddev_chocolatey.$VERSION.tar.gz chocolatey
fi

cp ddev_windows_installer*.exe $ARTIFACTS

# Create tarball of completion scripts
tar -czf $ARTIFACTS/ddev_shell_completion_scripts.$VERSION.tar.gz *completion*.sh

# Create macOS and Linux homebrew bottles
for os in sierra x86_64_linux ; do
    NO_V_VERSION=${VERSION#v}
    rm -rf /tmp/bottle
    BOTTLE_BASE=/tmp/bottle/ddev/$NO_V_VERSION
    mkdir -p $BOTTLE_BASE/{bin,etc/bash_completion.d,share/zsh/site-functions,share/fish/vendor_completions.d}
    cp $BASE_DIR/.gotmp/bin/ddev_bash_completion.sh $BOTTLE_BASE/etc/bash_completion.d/ddev
    cp $BASE_DIR/.gotmp/bin/ddev_zsh_completion.sh $BOTTLE_BASE/share/zsh/site-functions/_ddev
    cp $BASE_DIR/.gotmp/bin/ddev_fish_completion.sh $BOTTLE_BASE/share/fish/vendor_completions.d/ddev.fish

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
