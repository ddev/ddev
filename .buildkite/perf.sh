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
# PERF_PROJECT_DIR_SUFFIX (not a full path override) lets a leg get its own
# isolated project dir without baking a `~`/$HOME-relative path into YAML,
# where it wouldn't actually expand (env: values are literal strings, not
# passed through the shell) -- appending here, where $HOME is real, does.
PERF_PROJECT_DIR="${PERF_PROJECT_DIR:-$HOME/tmp/ddev-perf-drupal11${PERF_PROJECT_DIR_SUFFIX:-}}"
if [ ! -d "$PERF_PROJECT_DIR/web/core" ]; then
  echo "No existing Drupal11 codebase found at $PERF_PROJECT_DIR; provisioning it once."
  mkdir -p "$PERF_PROJECT_DIR"
  (
    cd "$PERF_PROJECT_DIR"
    # Project name is derived from the directory, not a fixed literal: a leg that
    # sets PERF_PROJECT_DIR to get its own isolated project (e.g. a Mutagen-off
    # variant, see perf-macos-shared-providers.yml) needs its own ddev project
    # name too, or its registration collides with the default project's.
    ddev config --project-type=drupal11 --docroot=web --project-name="$(basename "$PERF_PROJECT_DIR")" \
      ${PERF_PERFORMANCE_MODE:+--performance-mode="$PERF_PERFORMANCE_MODE"}
    ddev start -y
    ddev composer create-project drupal/recommended-project
  )
else
  echo "Reusing existing Drupal11 codebase at $PERF_PROJECT_DIR"
  ( cd "$PERF_PROJECT_DIR" && ddev start -y )
fi

# drupal/recommended-project doesn't include drush/drush, which 05-drush-install.sh
# needs -- add it once if missing, whether this is a fresh provision or a project
# from before this check existed.
if ! ( cd "$PERF_PROJECT_DIR" && ddev drush --version >/dev/null 2>&1 ); then
  echo "drush not available in $PERF_PROJECT_DIR; adding it once."
  ( cd "$PERF_PROJECT_DIR" && ddev composer require drush/drush )
fi

PROJECT_URL=$(cd "$PERF_PROJECT_DIR" && ddev describe -j | jq -r '.raw.primary_url')

echo "~~~ Setup complete, starting benchmark battery"
export DDEV_PERF_PROJECT_DIR="$PERF_PROJECT_DIR"
export DDEV_PERF_SITE_URL="$PROJECT_URL"
# Named per DOCKER_PROVIDER_LABEL/DOCKER_TYPE, not a fixed "perf-result.json":
# perf-macos-shared-providers.yml runs several providers as separate jobs within
# ONE build, and Buildkite artifacts are only unique per job, not per build -- a
# fixed name would make several same-named artifacts indistinguishable in the
# Buildkite UI. DOCKER_PROVIDER_LABEL (when set) takes priority over DOCKER_TYPE
# so two legs sharing a DOCKER_TYPE for provider bring-up (e.g. two orbstack
# variants) still upload distinctly-named artifacts. collect.js collects every
# perf-result*.json artifact from a build, so this doesn't need to match
# anything on that end.
RESULT_FILE="perf-result-${DOCKER_PROVIDER_LABEL:-${DOCKER_TYPE:-unknown}}.json"
"$(dirname "$0")/../perf/run-benchmark.sh" | tee "$RESULT_FILE"

echo "--- Uploading result artifact"
buildkite-agent artifact upload "$RESULT_FILE"
