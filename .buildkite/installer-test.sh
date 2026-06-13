#!/usr/bin/env bash

# Runs the Windows installer tests in their own dedicated Buildkite pipeline,
# decoupled from .buildkite/test.sh (the traditional Windows suite). Each
# Buildkite job runs a single matrix case selected by INSTALLER_CASE (the
# instance/subtest name, e.g. "ddev-test-debian-ce").
#
# Requires a Windows-native agent (GOOS=windows) with the named test distro
# already provisioned (see docs/content/developers/buildkite-testmachine-setup.md).

set -eu -o pipefail

# Disable git pager
export GIT_PAGER=""

# Note: [skip ci]/[skip buildkite] gating is handled by the step `if` in
# .buildkite/windows-installer.yml, which still lets manual (UI) builds run.
# Don't re-check the commit message here, or manual triggers of a [skip ci]
# commit would self-skip.

# Only run the installer matrix when installer-relevant files changed. Automatic
# (webhook) builds on a non-main branch skip unless their diff touches winpkg/ or
# the installer pipeline files. Pushes to main and manual (UI/API) builds always
# run. Mirrors the diff-gating in test.sh, scoped to the installer sources.
if [ "${BUILDKITE_SOURCE:-}" = "webhook" ] && [ "${BUILDKITE_BRANCH:-}" != "main" ]; then
  BASE_BRANCH="${BUILDKITE_PULL_REQUEST_BASE_BRANCH:-main}"
  MERGE_BASE=$(git merge-base HEAD "refs/remotes/origin/${BASE_BRANCH}" 2>/dev/null || true)
  if [ -n "${MERGE_BASE}" ] && ! git diff --name-only "${MERGE_BASE}" | grep -E '^(winpkg/|\.buildkite/installer-test\.(sh|cmd)$|\.buildkite/windows-installer\.yml$)' >/dev/null; then
    echo "+++ SKIP: No installer-relevant changes (winpkg/ or installer pipeline files)"
    exit 0
  fi
fi

# Load public variables (e.g. signing-related vars used by the installer build)
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

export DDEV_NONINTERACTIVE=true
export DDEV_DEBUG=true

# Find a suitable timeout command
if command -v gtimeout >/dev/null 2>&1; then
  TIMEOUT="gtimeout"
elif command -v timeout >/dev/null 2>&1; then
  TIMEOUT="timeout"
else
  echo "Error: Neither 'gtimeout' nor 'timeout' found in PATH." >&2
  exit 1
fi

echo
echo "buildkite installer test ${BUILDKITE_JOB_ID:-} at $(date) on $(hostname) as USER=${USER:-unknown} INSTALLER_CASE=${INSTALLER_CASE:-<all>} in ${PWD} golang=$(go version | awk '{print $3}')"

# Run any testbot maintenance that may need to be done
echo "--- running testbot_maintenance.sh"
${TIMEOUT} 10m bash "$(dirname "$0")/testbot_maintenance.sh"

# Our testbot should be sane, run the testbot checker to make sure.
echo "--- running sanetestbot.sh"
${TIMEOUT} 60s bash "$(dirname "$0")/sanetestbot.sh"

echo "~~~ Setup complete, starting installer test"

# Select a single matrix case when INSTALLER_CASE is set (one Buildkite job per
# case). When unset, all cases run (useful for local/manual invocation).
export DDEV_TEST_USE_REAL_INSTALLER=true
RUN_FILTER="TestWindowsInstaller"
if [ -n "${INSTALLER_CASE:-}" ]; then
  RUN_FILTER="TestWindowsInstallerWSL2/${INSTALLER_CASE}"
fi

echo "--- Running Windows installer test: ${RUN_FILTER}"
make testwininstaller TESTARGS="-run ${RUN_FILTER} ${TESTARGS:-}" | sed -u 's/^--- FAIL:/+++ FAIL:/; /\//!s/^=== RUN /--- RUN /'
RV=$?
echo "installer-test.sh completed with status=$RV"
exit $RV
