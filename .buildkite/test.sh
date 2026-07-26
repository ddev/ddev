#!/usr/bin/env bash
# This script is used to build ddev/ddev using buildkite

set -eu -o pipefail

# Disable git pager
export GIT_PAGER=""

# We can skip builds with commit message of [skip buildkite] or [skip ci]
DDEV_COMMIT_MESSAGE=$(git log -1 --pretty=%s 2>/dev/null || echo "")
if [[ ${BUILDKITE_MESSAGE:-} == *"[skip buildkite]"* ]] || [[ ${BUILDKITE_MESSAGE:-} == *"[skip ci]"* ]] || [[ ${DDEV_COMMIT_MESSAGE} == *"[skip buildkite]"* ]] || [[ ${DDEV_COMMIT_MESSAGE} == *"[skip ci]"* ]]; then
  echo "+++ SKIP: Build skipped due to commit message"
  echo "BUILDKITE_MESSAGE=${BUILDKITE_MESSAGE:-}"
  echo "DDEV_COMMIT_MESSAGE=${DDEV_COMMIT_MESSAGE}"
  exit 0
fi

git fetch --depth=1 --no-tags https://github.com/ddev/ddev public-variables:refs/public-variables-tmp
while IFS= read -r varname; do
  [[ "$varname" == "README.md" ]] && continue
  # MSYS_NO_PATHCONV prevents Git for Windows bash from mangling the ref:path syntax
  value=$(MSYS_NO_PATHCONV=1 git show "refs/public-variables-tmp:.github/public-variables/$varname")
  echo "$varname=${value}"
  export "$varname=$value"
done < <(MSYS_NO_PATHCONV=1 git ls-tree --name-only refs/public-variables-tmp:.github/public-variables/)
git update-ref -d refs/public-variables-tmp

export PATH=$PATH:/home/linuxbrew/.linuxbrew/bin
os=$(go env GOOS)

# GOTEST_SHORT=16 means drupal11
export GOTEST_SHORT=16
if [ ${OSTYPE:-unknown}  = "msys" ]; then export GOTEST_SHORT=true; fi

export DDEV_SKIP_NODEJS_TEST=true

export DOCKER_SCAN_SUGGEST=false
export DOCKER_SCOUT_SUGGEST=false

# Provider bring-up/cleanup logic (TIMEOUT detection, provider_bringup,
# provider_shutdown) is shared with perf.sh -- see lib-provider.sh.
# shellcheck source=lib-provider.sh
source "$(dirname "$0")/lib-provider.sh"

# On macOS, we can have several different docker providers, allow testing all
# In cleanup, stop everything we know of but leave either Orbstack or Docker Desktop running
if [ "${os:-}" = "darwin" ]; then
  function cleanup {
    # Post-test maintenance, while the provider is still up and before we stop it.
    # Guarded so it doesn't run on the initial cleanup call at startup.
    if [ "${RAN_TESTS:-false}" = "true" ]; then
      echo "--- running testbot_maintenance.sh (post-test)"
      ${TIMEOUT} 10m bash "$(dirname "$0")/testbot_maintenance.sh" || true
    fi
    provider_shutdown
  }
  trap cleanup EXIT

  # Start with a predictable docker provider running
  cleanup

  echo "initial docker context situation:"
  docker context ls
fi

provider_bringup

echo
echo "buildkite building ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER:-unknown} for OS=${os:-} DOCKER_TYPE=${DOCKER_TYPE:-notset} in ${PWD} with GOTEST_SHORT=${GOTEST_SHORT:-notset} golang=$(go version | awk '{print $3}') ddev version=$(ddev --version | awk '{print $3}')"

echo
case ${DOCKER_TYPE:-none} in
  "docker-ce")
    echo "Running docker-ce (Docker CE)"
    ;;
  "docker-desktop")
    echo "docker-desktop for mac version=$(scripts/docker-desktop-version.sh)"
    ;;
  "colima")
    echo "colima version=$(colima version)"
    ;;
  "colima_vz")
    echo "colima version=$(colima version)"
    ;;
  "lima")
    echo "limactl --version=$(limactl --version)"
    ;;
  "orbstack")
    echo "orbstack version=$(orbctl version)"
    ;;
  "rancher-desktop")
    echo "rancher-desktop=$(~/.rd/bin/rdctl version)"
    ;;
  "podman-rootless")
    echo "podman version=$(podman --version)"
    ;;
  "wsl2dockerinside")
    echo "Running wsl2dockerinside"
    ;;
  "dockerforwindows")
    echo "Running Windows docker desktop for windows"
    ;;
  "wsl2-docker-desktop")
    echo "Running wsl2-docker-desktop"
    ;;
  *)
    echo "$DOCKER_TYPE not found"
    ;;
esac

echo "Docker version:"
docker version
if command -v ddev >/dev/null ; then
  echo "ddev version:"
  ddev version
else
  echo "ddev not installed"
fi
echo

export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

# If this is a PR and the diff doesn't have code, skip it
set -x
if [ "${BUILDKITE_PULL_REQUEST:-false}" != "false" ]; then
  # Find the merge base between the PR branch and the base branch
  MERGE_BASE=$(git merge-base HEAD refs/remotes/origin/${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-})
  # Check if there are any changes in the specified directories or files since the merge base
  if ! git diff --name-only "$MERGE_BASE" | grep -E '^(\.buildkite/|Makefile$|pkg/|cmd/|vendor/|winpkg/|go\.)' >/dev/null; then
    echo "+++ SKIP: No relevant code changes found"
    echo "No changes in: .buildkite/, Makefile, pkg/, cmd/, vendor/, winpkg/, go.*"
    exit 0
  fi

fi

# Run any testbot maintenance that may need to be done
echo "--- running testbot_maintenance.sh"

${TIMEOUT} 10m bash "$(dirname "$0")/testbot_maintenance.sh"

# Our testbot should be sane, run the testbot checker to make sure.
echo "--- running sanetestbot.sh"
${TIMEOUT} 60s bash "$(dirname "$0")/sanetestbot.sh"

# Rootless podman cannot bind privileged ports (<1024), so the DDEV router must
# use non-privileged ports. This must run AFTER testbot_maintenance.sh, which
# deletes ~/.ddev/global_config.yaml; otherwise the override is wiped and the
# config regenerates with the default 80/443 (which rootless podman can't bind).
if [ "${DOCKER_TYPE:-}" = "podman-rootless" ]; then
  ddev config global --router-http-port=8080 --router-https-port=8443
fi

# Close the setup sections before starting tests
echo "~~~ Setup complete, starting tests"

# Make sure we start with mutagen daemon off.
unset MUTAGEN_DATA_DIRECTORY
if [ -f ~/.ddev/bin/mutagen -o -f ~/.ddev/bin/mutagen.exe ]; then
  MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory/ ~/.ddev/bin/mutagen sync terminate -a || true
  MUTAGEN_DATA_DIRECTORY=~/.mutagen ~/.ddev/bin/mutagen daemon stop || true
  MUTAGEN_DATA_DIRECTORY=~/.ddev_mutagen_data_directory/ ~/.ddev/bin/mutagen daemon stop || true
fi
if command -v killall >/dev/null ; then
  killall mutagen || true
fi

echo "--- Running tests..."

# Note: Windows installer tests are no longer run here. They run in their own
# dedicated Buildkite pipeline (.buildkite/installer-test.sh +
# .buildkite/windows-installer.yml), decoupled from this suite so they can fan
# out across runners.

# From here on, cleanup (the EXIT trap) runs testbot_maintenance.sh post-test.
RAN_TESTS=true
make ${MAKE_TARGET:-test} TESTARGS="${TESTARGS:-}" TESTPKG="${TESTPKG:-}" TESTFILE="${TESTFILE:-}" | sed -u 's/^--- FAIL:/+++ FAIL:/; /\//!s/^=== RUN /--- RUN /'
RV=$?
echo "test.sh completed with status=$RV"
ddev poweroff || true

exit $RV
