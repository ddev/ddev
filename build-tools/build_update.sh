#!/bin/bash

# Update (or install) build-tools to latest release on https://github.com/drud/build-tools/releases/latest

set -e

base_url=$(curl -s -I https://github.com/drud/build-tools/releases/latest | awk '/^Location/ {gsub(/[\n\r]/,"",$2);  printf "%s", $2}')
tag=${base_url##*/}

tarball_url="https://github.com/drud/build-tools/archive/$tag.tar.gz"
internal_name=build-tools-$tag
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
rm -rf build-tools/tests/ build-tools/circle.yml build-tools/.github
touch build-tools/build-tools-VERSION-$tag.txt
git add build-tools
echo "Updated build-tools to $tag

$base_url" | git commit -F -
