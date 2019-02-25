#!/bin/bash

# Check a testbot or test environment to make sure it's likely to be sane.
# We should add to this script whenever a testbot fails and we can figure out why.

set -o errexit
set -o pipefail
set -o nounset


# Check that required commands are available.
for command in git go make; do
    command -v $command >/dev/null || ( echo "Did not find command installed '$command'" && exit 2 )
done
docker run -t busybox ls >/dev/null

if [ "$(go env GOOS)" = "windows"  -a "$(git config core.autocrlf)" != "false" ] ; then
 echo "git config core.autocrlf is not set to false on windows"
 exit 3
fi

echo "--- testbot $HOSTNAME seems to be set up OK"
