#!/usr/bin/env bash

# Prints where this build came from, for versionconstants.BuildSource. On
# GitHub Actions, the run URL (with the PR number appended when triggered by
# a pull_request event). Otherwise, who built it and from which branch, since
# there's no CI run to point to. Invoked by the Makefile via -ldflags.

set -eu -o pipefail

if [ -n "${GITHUB_RUN_ID:-}" ]; then
  url="${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}"
  if [ "${GITHUB_EVENT_NAME:-}" = "pull_request" ] && [ -n "${GITHUB_REF_NAME:-}" ]; then
    pr_number="${GITHUB_REF_NAME%%/*}"
    echo "${url} (PR #${pr_number})"
  else
    echo "${url}"
  fi
  exit 0
fi

branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
user="${USER:-${USERNAME:-unknown}}"
host=$(hostname 2>/dev/null || echo "unknown")
echo "local build by ${user}@${host%%.*}, branch ${branch}"
