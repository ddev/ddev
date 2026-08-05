#!/usr/bin/env bash
# Nightly performance-benchmark entry point for Buildkite, run in place of
# test.sh by the perf-*.yml pipelines (see perf/README.md for why these are
# separate, schedule-triggered Buildkite pipelines rather than steps added to
# the existing push/PR-triggered test pipelines). Shares Docker-provider
# bring-up/cleanup logic with test.sh via lib-provider.sh so the two can't
# drift out of sync.
set -eu -o pipefail

export GIT_PAGER=""

if [[ ${BUILDKITE_MESSAGE:-} == *"[skip buildkite]"* ]] || [[ ${BUILDKITE_MESSAGE:-} == *"[skip ci]"* ]]; then
  echo "+++ SKIP: Build skipped due to commit message"
  exit 0
fi

os=$(go env GOOS)

# shellcheck source=lib-provider.sh
source "$(dirname "$0")/lib-provider.sh"

if [ "${os:-}" = "darwin" ]; then
  function cleanup {
    provider_shutdown
  }
  trap cleanup EXIT

  # Start with a predictable docker provider running
  cleanup
fi

provider_bringup

# Rootless podman cannot bind privileged ports (<1024), so the DDEV router must
# use non-privileged ports -- same override test.sh applies for this DOCKER_TYPE.
if [ "${DOCKER_TYPE:-}" = "podman-rootless" ]; then
  ddev config global --router-http-port=8080 --router-https-port=8443
fi

echo
echo "buildkite perf run ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER:-unknown} for OS=${os:-} DOCKER_TYPE=${DOCKER_TYPE:-notset}"
echo

echo "--- Building ddev"
make build
export PATH="$PWD/.gotmp/bin/$(go env GOOS)_$(go env GOARCH):$PATH"
ddev version

echo "--- Provisioning benchmark project"
# Reused across nightly runs rather than recreated every time: re-running
# `composer create-project` nightly would add Composer/network variance
# unrelated to what we're actually trying to measure. Each metric script
# resets the db/files/opcache it needs before timing itself (see
# perf/lib/reset-drupal.sh), so reusing the codebase is safe.
PERF_PROJECT_DIR="${PERF_PROJECT_DIR:-$HOME/tmp/ddev-perf-drupal11}"
if [ ! -d "$PERF_PROJECT_DIR/web/core" ]; then
  echo "No existing Drupal11 codebase found at $PERF_PROJECT_DIR; provisioning it once."
  mkdir -p "$PERF_PROJECT_DIR"
  (
    cd "$PERF_PROJECT_DIR"
    ddev config --project-type=drupal11 --docroot=web --project-name=ddev-perf-drupal11
    ddev start -y
    ddev composer create-project drupal/recommended-project
  )
else
  echo "Reusing existing Drupal11 codebase at $PERF_PROJECT_DIR"
  ( cd "$PERF_PROJECT_DIR" && ddev start -y )
fi

PROJECT_URL=$(cd "$PERF_PROJECT_DIR" && ddev describe -j | jq -r '.raw.primary_url')

echo "~~~ Setup complete, starting benchmark battery"
export DDEV_PERF_PROJECT_DIR="$PERF_PROJECT_DIR"
export DDEV_PERF_SITE_URL="$PROJECT_URL"
# Named per DOCKER_TYPE, not a fixed "perf-result.json": perf-macos-shared-providers.yml
# runs several providers as separate jobs within ONE build, and Buildkite artifacts are
# only unique per job, not per build -- a fixed name would make five same-named artifacts
# indistinguishable in the Buildkite UI. collect.js collects every perf-result*.json
# artifact from a build, so this doesn't need to match anything on that end.
RESULT_FILE="perf-result-${DOCKER_TYPE:-unknown}.json"
"$(dirname "$0")/../perf/run-benchmark.sh" | tee "$RESULT_FILE"

echo "--- Uploading result artifact"
buildkite-agent artifact upload "$RESULT_FILE"
