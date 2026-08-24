#!/usr/bin/env bash

# This script runs as testuser inside WSL2 to build and test DDEV.
# It mirrors the "DDEV tests" step in test-reusable.yml.
# It expects to be run from within the cloned ddev repo directory.

set -eu -o pipefail
set -x

export GOTEST_SHORT="${GOTEST_SHORT:-16}"
export DDEV_NO_INSTRUMENTATION=true
export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true
export DDEV_SKIP_NODEJS_TEST="${DDEV_SKIP_NODEJS_TEST:-true}"
export DDEV_EMBARGO_TESTS="${DDEV_EMBARGO_TESTS:-}"
export DDEV_EMBARGO_PHP_VERSIONS="${DDEV_EMBARGO_PHP_VERSIONS:-}"
export BUILDKIT_PROGRESS=plain
export DOCKER_CLI_EXPERIMENTAL=enabled
export DOCKER_SCAN_SUGGEST=false
export DOCKER_SCOUT_SUGGEST=false
export CGO_ENABLED="${CGO_ENABLED:-0}"
export BUILDARGS="${BUILDARGS:-}"
export TESTARGS="${TESTARGS:-}"
export MAKEARGS="${MAKEARGS:-}"
export MAKE_TARGET="${MAKE_TARGET:-test}"
export PATH="/usr/local/go/bin:$PATH"
# Picked up by the Makefile's gotest macro (see Makefile) so `make test` runs
# through gotestsum and writes per-test JSON events here, relative to the repo
# root this script runs from. The caller workflow copies this dir out of WSL2
# and uploads it for the CI test-runtime collector (perf/collector/README.md).
export GOTESTSUM_JSONFILE_DIR=test-results

echo "=== Environment ==="
echo "GOTEST_SHORT=${GOTEST_SHORT}"
echo "DDEV_SKIP_NODEJS_TEST=${DDEV_SKIP_NODEJS_TEST}"
echo "DDEV_EMBARGO_TESTS=${DDEV_EMBARGO_TESTS}"
echo "DDEV_EMBARGO_PHP_VERSIONS=${DDEV_EMBARGO_PHP_VERSIONS}"
echo "CGO_ENABLED=${CGO_ENABLED}"
echo "BUILDARGS=${BUILDARGS}"
echo "TESTARGS=${TESTARGS}"
echo "MAKEARGS=${MAKEARGS}"
echo "MAKE_TARGET=${MAKE_TARGET}"

echo "=== Ensuring Docker is running ==="
sudo systemctl start docker
for i in $(seq 1 30); do
  if docker info >/dev/null 2>&1; then
    echo "Docker is ready after ${i}s"
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: Docker not ready after 30s"
    exit 1
  fi
  sleep 1
done

echo "=== Verifying prerequisites ==="
go version
docker version
git --version

# This runner holds no push credentials, so it can race image-push.yml's
# approval/build/push - see containers/wait-for-images.sh.
echo "=== Waiting for pushed images ==="
containers/wait-for-images.sh

echo "=== Building DDEV ==="
make CGO_ENABLED="${CGO_ENABLED}" BUILDARGS="${BUILDARGS}"

echo "=== Running tests ==="
make CGO_ENABLED="${CGO_ENABLED}" BUILDARGS="${BUILDARGS}" TESTARGS="${TESTARGS}" ${MAKE_TARGET} ${MAKEARGS}
