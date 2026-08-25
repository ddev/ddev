#!/usr/bin/env bash

# Prints the GitHub Actions run URL for the current build, with the PR
# number appended when triggered by a pull_request event. Prints nothing
# for a local build, where GITHUB_RUN_ID isn't set. Invoked by the Makefile
# to populate versionconstants.BuildSource via -ldflags.

set -eu -o pipefail

if [ -z "${GITHUB_RUN_ID:-}" ]; then
  exit 0
fi

url="${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"

if [ "${GITHUB_EVENT_NAME:-}" = "pull_request" ] && [ -n "${GITHUB_REF_NAME:-}" ]; then
  pr_number="${GITHUB_REF_NAME%%/*}"
  echo "${url} (PR #${pr_number})"
else
  echo "${url}"
fi
