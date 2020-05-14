#!/bin/bash

# Update (or install) build-tools to latest release on https://github.com/drud/build-tools/releases/latest

set -e

LATEST_RELEASE=$(curl -L -s -H 'Accept: application/json' https://github.com/drud/build-tools/releases/latest)
# The releases are returned in the format {"id":3622206,"tag_name":"hello-1.0.0.11",...}, we have to extract the tag_name.
LATEST_VERSION=$(echo $LATEST_RELEASE | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
URL="https://github.com/drud/build-tools/releases/download/$LATEST_VERSION"
tag=${LATEST_VERSION}

tarball_url="https://github.com/drud/build-tools/archive/$tag.tar.gz"
internal_name=build-tools-${tag#v}
local_file=/tmp/$internal_name.tgz

# If there is a current build-tools, get permission and remove
if [ "${PWD##*/}" = "build-tools" ]; then
	echo "OK to replace current build-tools at $PWD?"
	read -p "Replace build-tools with latest version? y/N" -n 1 -r
	echo
	if ! [[ $REPLY =~ ^[Yy]$ ]]
	then
	  echo "Exiting"
	  exit 1
	fi
	cd ..
# If no current build-tools, prompt, get permission to add
else
	echo "OK to add current build-tools at $PWD/build-tools?"
	read -p "Add build-tools with latest version? y/N" -n 1 -r
	echo
	if ! [[ $REPLY =~ ^[Yy]$ ]]
	then
	  echo "Exiting"
	  exit 1
	fi
fi


wget -q -O $local_file $tarball_url
tar -xf $local_file
rm -rf build-tools/*
cp -r $internal_name/ build-tools/
rm -rf $internal_name/
rm -rf build-tools/{tests,circle.yml,.circleci,.github,.appveyor.yml,.buildkite,.autotests}
touch build-tools/build-tools-VERSION-$tag.txt
git add build-tools
echo "Updated build-tools to $tag

$base_url" | git commit -F -
