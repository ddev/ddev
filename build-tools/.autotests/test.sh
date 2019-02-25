#!/bin/bash

# build drud/build-tools for automated testing (buildkite/appveyor/circleci)

set -o errexit
set -o pipefail
set -o nounset
#set -x

# Make sure that everything remains readable. Go module cache is always getting
# set to read-only, meanning it can't be cleaned up.
function cleanup {
	chmod -R u+w . || true
}
trap cleanup EXIT

BUILD_OS=$(go env GOOS)
echo "--- building at $(date) on $HOSTNAME for OS=$(go env GOOS) in $PWD"

# Our testbot should now be sane, run the testbot checker to make sure.
echo "--- Checking for sane testbot"
./.autotests/sanetestbot.sh

echo "--- make $BUILD_OS"
cd tests
rm -f windows darwin linux && time make
echo "--- make test"
time make test
RV=$?
echo "--- build.sh completed with status=$RV"
exit $RV
