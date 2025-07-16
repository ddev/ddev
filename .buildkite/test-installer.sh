#!/usr/bin/env bash
# This script is used to test the ddev Windows isntaller using buildkite

set -eu -o pipefail

export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin

# Make sure docker is working
echo "Waiting for docker provider to come up: $(date)"
date && ${TIMEOUT_CMD} 3m bash -c 'while ! docker ps >/dev/null 2>&1 ; do
  sleep 10
  echo "Waiting: $(date)"
done'
echo "Testing again to make sure docker came up: $(date)"
if ! docker ps >/dev/null 2>&1 ; then
  echo "Docker is not running, exiting"
  exit 1
fi

echo
echo "buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER:-unknown} for OS=${OSTYPE:-unknown} DOCKER_TYPE=${DOCKER_TYPE:-notset} in ${PWD} with GOTEST_SHORT=${GOTEST_SHORT:-notset} golang=$(go version | awk '{print $3}') ddev version=$(ddev --version | awk '{print $3}')"

echo

echo "Docker version:"
docker version
echo

export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

# We can skip builds with commit message of [skip buildkite]
if [[ ${BUILDKITE_MESSAGE:-} == *"[skip buildkite]"* ]] || [[ ${BUILDKITE_MESSAGE:-} == *"[skip ci]"* ]]; then
  echo "Skipping build because message has '[skip buildkite]' or '[skip ci]'"
  exit 0
fi

# If this is a PR and the diff doesn't have code, skip it
set -x
if [ "${BUILDKITE_PULL_REQUEST:-false}" != "false" ]; then
  # Find the merge base between the PR branch and the base branch
  MERGE_BASE=$(git merge-base HEAD refs/remotes/origin/${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-})
  # Check if there are any changes in the specified directories or files since the merge base
  if ! git diff --name-only $MERGE_BASE | egrep "^(\.buildkite|Makefile|pkg|cmd|winpkg|vendor|go\.)" >/dev/null; then
    echo "Skipping buildkite build since no code changes found"
    exit 0
  fi
fi


echo "--- Running tests..."
make testwininstaller TESTARGS="-failfast"
RV=$?
echo "test-installer.sh completed with status=$RV"

exit $RV
