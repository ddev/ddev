#!/bin/bash
# This script builds ddev artifacts and their sha256 hashes.
# First arg is the artifact directory

set -o errexit
set -o pipefail
set -o nounset

MKCERT_VERSION=v1.4.6
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

BUILTPATH=.gotmp/bin/$(go env GOOS)_$(go env GOARCH)

if [ "${BUILD_IMAGE_TARBALLS}" = "true" ]; then
    # Make sure we have all our docker images, and save them in a tarball
    $BUILTPATH/ddev version | awk '/(drud|phpmyadmin)\// {print $2;}' >/tmp/images.txt
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
for dir in .gotmp/bin/linux_amd64 .gotmp/bin/linux_arm64 .gotmp/bin/darwin_amd64 .gotmp/bin/darwin_arm64 .gotmp/bin/windows_amd64; do
  cp .gotmp/bin/ddev_*completion* $dir
done

# Create tarball of completion scripts
pushd .gotmp/bin >/dev/null && tar -czf $ARTIFACTS/ddev_shell_completion_scripts.$VERSION.tar.gz *completion*.sh && popd >/dev/null
cp $BASE_DIR/.gotmp/bin/windows_amd64/ddev_windows_installer*.exe $ARTIFACTS

# Generate macOS-amd64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/darwin_amd64 >/dev/null
curl -sSL -o mkcert https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-darwin-amd64 && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_macos-amd64.$VERSION.tar.gz ddev *completion*.sh mkcert
popd >/dev/null

# Generate macOS-arm64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/darwin_arm64 >/dev/null
curl -sSL -o mkcert https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-darwin-amd64 && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_macos-arm64.$VERSION.tar.gz ddev *completion*.sh mkcert
popd >/dev/null

# Generate linux-amd64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/linux_amd64 >/dev/null
curl -sSL -o mkcert https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-amd64 && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_linux-amd64.$VERSION.tar.gz ddev *completion*.sh mkcert
popd >/dev/null

# Generate linux-arm64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/linux_arm64 >/dev/null
curl -sSL -o mkcert https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-arm64 && chmod +x mkcert
tar -czf $ARTIFACTS/ddev_linux-arm64.$VERSION.tar.gz ddev *completion*.sh mkcert
popd >/dev/null

# Generate linux-arm tarball/zipball
#pushd $BASE_DIR/.gotmp/bin/linux_arm >/dev/null
#curl -sSL -o mkcert https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-linux-arm && chmod +x mkcert
#tar -czf $ARTIFACTS/ddev_linux-arm.$VERSION.tar.gz ddev *completion*.sh
#popd >/dev/null

# generate windows-amd64 tarball/zipball
pushd $BASE_DIR/.gotmp/bin/windows_amd64 >/dev/null
curl -sSL -o mkcert.exe https://github.com/drud/mkcert/releases/download/${MKCERT_VERSION}/mkcert-${MKCERT_VERSION}-windows-amd64.exe
tar -czf $ARTIFACTS/ddev_windows-amd64.$VERSION.tar.gz ddev.exe *completion*.sh mkcert.exe
if [ -d chocolatey ]; then
    tar -czf $ARTIFACTS/ddev_chocolatey_amd64-.$VERSION.tar.gz chocolatey
fi
popd >/dev/null

# Create macOS and Linux homebrew bottles
for os in high_sierra arm64_big_sur x86_64_linux ; do
    NO_V_VERSION=${VERSION#v}
    rm -rf /tmp/bottle
    BOTTLE_BASE=/tmp/bottle/ddev/$NO_V_VERSION
    mkdir -p $BOTTLE_BASE/{bin,etc/bash_completion.d,share/zsh/site-functions,share/fish/vendor_completions.d}
    cp $BASE_DIR/.gotmp/bin/ddev_bash_completion.sh $BOTTLE_BASE/etc/bash_completion.d/ddev
    cp $BASE_DIR/.gotmp/bin/ddev_zsh_completion.sh $BOTTLE_BASE/share/zsh/site-functions/_ddev
    cp $BASE_DIR/.gotmp/bin/ddev_fish_completion.sh $BOTTLE_BASE/share/fish/vendor_completions.d/ddev.fish

    if [ "${os}" = "high_sierra" ]; then cp $BASE_DIR/.gotmp/bin/darwin_amd64/ddev $BOTTLE_BASE/bin ; fi
    if [ "${os}" = "arm64_big_sur" ]; then cp $BASE_DIR/.gotmp/bin/darwin_arm64/ddev $BOTTLE_BASE/bin ; fi
    if [ "${os}" = "x86_64_linux" ]; then cp $BASE_DIR/.gotmp/bin/linux_amd64/ddev $BOTTLE_BASE/bin ; fi
    cp $BASE_DIR/{README.md,LICENSE} $BOTTLE_BASE
    tar -czf $ARTIFACTS/ddev-$NO_V_VERSION.$os.bottle.tar.gz -C /tmp/bottle .
done

# Create the sha256 files
cd $ARTIFACTS
for item in *.*; do
  sha256sum $item > $item.sha256.txt
done
