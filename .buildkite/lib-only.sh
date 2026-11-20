# shellcheck shell=bash
# Sourced by the Buildkite entry scripts (test.sh, installer-test.sh).
#
# A commit whose subject line contains `[only <pipeline> ...]` runs only the
# named Buildkite pipelines; every other pipeline's job exits successfully
# before doing any work. Pipelines are named by their Buildkite slug, the name
# in the build URL, such as macos-socktainer.
#
# This reads the commit, not BUILDKITE_MESSAGE: for a pull request build,
# Buildkite's message is the PR title. Only the subject line counts, so a
# squash merge, whose subject is the PR title, doesn't carry a tag from a
# commit body into main. Builds started from the Buildkite UI always run.

function skip_unless_named_by_only {
  local subject names re='\[only ([^]]+)\]'
  [[ "${BUILDKITE_SOURCE:-}" == "ui" ]] && return 0
  [[ -z "${BUILDKITE_PIPELINE_SLUG:-}" ]] && return 0
  subject=$(git log -1 --pretty=%s 2>/dev/null || echo "")
  [[ ${subject} =~ ${re} ]] || return 0
  names=" ${BASH_REMATCH[1]} "
  if [[ ${names} != *" ${BUILDKITE_PIPELINE_SLUG} "* ]]; then
    echo "+++ SKIP: the commit subject runs only: ${BASH_REMATCH[1]}"
    echo "This pipeline is ${BUILDKITE_PIPELINE_SLUG}."
    echo "DDEV_COMMIT_MESSAGE=${subject}"
    exit 0
  fi
}
