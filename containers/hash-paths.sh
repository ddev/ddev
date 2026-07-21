#!/usr/bin/env bash
# hash-paths.sh <path> [<path> ...]
#
# Prints a deterministic content hash of the current working-tree content
# under the given path(s), covering both committed and dirty
# (staged/unstaged/untracked-but-not-ignored) state. No Docker, no network.
#
# Env:
#   HASH_LEN - number of hex characters to print (default 10)

set -eu -o pipefail

if [ "$#" -eq 0 ]; then
  echo "Usage: $0 <path> [<path> ...]" >&2
  exit 1
fi

HASH_LEN="${HASH_LEN:-10}"

cd "$(git rev-parse --show-toplevel)"

{
  git ls-files -- "$@"
  git ls-files --others --exclude-standard -- "$@"
} | LC_ALL=C sort -u | xargs -r sha256sum | LC_ALL=C sort | sha256sum | cut -c1-"${HASH_LEN}"
