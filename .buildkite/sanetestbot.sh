#!/bin/bash

# Check a testbot or test environment to make sure it's likely to be sane.
# We should add to this script whenever a testbot fails and we can figure out why.

set -o errexit
set -o pipefail
set -o nounset
#set -x

# Test to make sure docker is installed and working
docker ps >/dev/null
# Test that docker can allocate 80 and 443, get get busybox
docker pull busybox:latest >/dev/null
docker run -t -p 80:80 -p 443:443 -p 1081:1081 -p 1082:1082 --rm busybox:latest ls >/dev/null

# Check that required commands are available.
for command in mysql git go make; do
    command -v $command >/dev/null || ( echo "Did not find command installed '$command'" && exit 2 )
done

if [ "$(go env GOOS)" = "windows"  -a "$(git config core.autocrlf)" != "false" ] ; then
 echo "git config core.autocrlf is not set to false on windows"
 exit 3
fi

echo "=== testbot $HOSTNAME seems to be set up OK ==="
