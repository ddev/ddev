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
export BUILDKIT_PROGRESS=plain
export DOCKER_CLI_EXPERIMENTAL=enabled
export DOCKER_SCAN_SUGGEST=false
export DOCKER_SCOUT_SUGGEST=false
export CGO_ENABLED="${CGO_ENABLED:-0}"
export BUILDARGS="${BUILDARGS:-}"
export TESTARGS="${TESTARGS:--failfast}"
export MAKE_TARGET="${MAKE_TARGET:-test}"
export PATH="/usr/local/go/bin:$PATH"

echo "=== Environment ==="
echo "GOTEST_SHORT=${GOTEST_SHORT}"
echo "DDEV_SKIP_NODEJS_TEST=${DDEV_SKIP_NODEJS_TEST}"
echo "DDEV_EMBARGO_TESTS=${DDEV_EMBARGO_TESTS}"
echo "CGO_ENABLED=${CGO_ENABLED}"
echo "BUILDARGS=${BUILDARGS}"
echo "TESTARGS=${TESTARGS}"
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

echo "=== Building DDEV ==="
make CGO_ENABLED="${CGO_ENABLED}" BUILDARGS="${BUILDARGS}"

echo "=== Running tests ==="
make CGO_ENABLED="${CGO_ENABLED}" BUILDARGS="${BUILDARGS}" TESTARGS="${TESTARGS}" ${MAKE_TARGET}
