#!/usr/bin/env bash
# Prints NAME=value for each file on the public-variables branch, except README.md.
# See .github/public-variables/README.md for the file format.
# Each file holds one value per line. "#" starts a comment that runs to the end
# of the line. Values are joined with "|" so they fit in one environment variable.
# PUBLIC_VARIABLES_REPO overrides the repository, for testing.

set -eu -o pipefail

repo="${PUBLIC_VARIABLES_REPO:-https://github.com/ddev/ddev}"
ref=refs/public-variables-tmp
dir=.github/public-variables
# Git for Windows bash would otherwise mangle the ref:path arguments
export MSYS_NO_PATHCONV=1

git fetch --quiet --depth=1 --no-tags "${repo}" "+public-variables:${ref}"
trap 'git update-ref -d "${ref}"' EXIT

for varname in $(git ls-tree --name-only "${ref}:${dir}/"); do
  [[ "${varname}" == "README.md" ]] && continue
  value=$(git show "${ref}:${dir}/${varname}" | tr -d '\r' \
    | sed -e 's/#.*//' -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//' -e '/^$/d' \
    | paste -s -d '|' -)
  echo "${varname}=${value}"
done
