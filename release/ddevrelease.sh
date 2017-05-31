#!/bin/bash -ex
# This script downloads artifacts from CircleCI and prepares them for upload to
# accopany a GitHub release.  See the ddevrelease component of this projects
# Makefile for additional information.
function package {
    dir=${PWD##*/}
    for filename in "`pwd`/*"
    do
        echo $filename
    if [ "$dir" == "windows" ]
    then
        zip ddev_"$dir".zip `basename $filename`
        shasum -a 256 ddev_"$dir".zip > ddev_"$dir".sha256
    else
        tar -cvzf ddev_"$dir".tar.gz `basename $filename`
        shasum -a 256 ddev_"$dir".tar.gz > ddev_"$dir".sha256
    fi
    done;
}
function setup {
    mkdir -p $1/release/$3/darwin
    cd $1/release/$3/darwin
    curl -O "$2/darwin/ddev"
    chmod +x ddev

    mkdir -p $1/release/$3/linux
    cd $1/release/$3/linux
    curl -O "$2/linux/ddev"
    chmod +x ddev

    mkdir -p $1/release/$3/windows
    cd $1/release/$3/windows
    curl -O "$2/windows/ddev.exe"
}
function cleanup {
    mv $1/release/$2/*/*.sha256 $1/release/$2
    mv $1/release/$2/*/*.tar.gz $1/release/$2
    mv $1/release/$2/*/*.zip $1/release/$2
    rm -rf $1/release/$2/darwin
    rm -rf $1/release/$2/linux
    rm -rf $1/release/$2/windows
}

DIR=`pwd`
TAG=$(git describe --tags)
setup $DIR $1 $TAG
for d in $DIR/release/$TAG/*/ ; do (cd "$d" && package); done
cleanup $DIR $TAG
