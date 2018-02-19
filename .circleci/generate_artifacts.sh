#!/bin/bash

set -o errexit

ARTIFACTS=$1
BASE_DIR=$PWD

sudo mkdir $ARTIFACTS && sudo chmod 777 $ARTIFACTS
export VERSION=$(git describe --tags --always --dirty)

# Make sure we have all our docker images, and save them in a tarball
$BASE_DIR/bin/linux/ddev version | awk '/drud\// {print $2;}' >/tmp/images.txt
for item in $(cat /tmp/images.txt); do
  docker pull $item
done
docker save -o /tmp/docker_images.tar $(cat /tmp/images.txt)

# Generate and place the extra items like images and autocomplete
bin/linux/ddev_gen_autocomplete
for dir in bin/darwin/darwin_amd64 bin/linux bin/windows/windows_amd64; do
  cp bin/ddev_bash_completion.sh $dir
  cp /tmp/docker_images.tar $dir
done

# Generate macOS tarball/zipball
cd $BASE_DIR/bin/darwin/darwin_amd64
tar -czf $ARTIFACTS/ddev_macos.$VERSION.tar.gz ddev ddev_bash_completion.sh docker_images.tar
zip $ARTIFACTS/ddev_macos.$VERSION.zip ddev ddev_bash_completion.sh docker_images.tar

# Generate linux tarball/zipball
cd $BASE_DIR/bin/linux
tar -czf $ARTIFACTS/ddev_linux.$VERSION.tar.gz ddev ddev_bash_completion.sh docker_images.tar
zip $ARTIFACTS/ddev_linux.$VERSION.zip ddev ddev_bash_completion.sh docker_images.tar

# generate windows tarball/zipball
cd $BASE_DIR/bin/windows/windows_amd64
tar -czf $ARTIFACTS/ddev_windows.$VERSION.tar.gz ddev.exe ddev_bash_completion.sh docker_images.tar
zip $ARTIFACTS/ddev_windows.$VERSION.zip ddev.exe ddev_bash_completion.sh docker_images.tar

# Create the sha256 files
cd $ARTIFACTS
for item in *.tar.gz *.zip; do
  sha256sum $item > $item.sha256.txt
done
