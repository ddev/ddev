#!/usr/bin/env bash
# required-image-tag.sh <TagVarName> <hash-path> [<hash-path> ...]
#
# Prints "<state> <tag>" for the image tag this checkout resolves to. The tag
# is the bare content hash either way - it does not depend on the branch, the
# fork, or the registry it was published to - so every caller agrees on it.
# The state says where that leaves the checkout:
#
#   committed <tag>   versionconstants.go already names this tag, so it is
#                     what ddev pulls and it has to exist in the registry.
#   recomputed <tag>  the content changed, so autotag.sh rewrites
#                     versionconstants.go to <tag> and, for an image `make`
#                     builds, produces it locally.
#
# Env:
#   HASH_LEN              - hash length in hex chars (default 10, must match
#                           hash-paths.sh)
#   VERSIONCONSTANTS_FILE - path to versionconstants.go

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git rev-parse --show-toplevel)"
HASH_LEN="${HASH_LEN:-10}"
VERSIONCONSTANTS_FILE="${VERSIONCONSTANTS_FILE:-$REPO_ROOT/pkg/versionconstants/versionconstants.go}"

if [ "$#" -lt 2 ]; then
  echo "Usage: $0 <TagVarName> <hash-path> [<hash-path> ...]" >&2
  exit 2
fi

TAG_VAR="$1"; shift

CURRENT_HASH="$(HASH_LEN="$HASH_LEN" "$SCRIPT_DIR/hash-paths.sh" "$@")"

EXISTING_TAG="$(grep -E "^var ${TAG_VAR} = " "$VERSIONCONSTANTS_FILE" 2>/dev/null | sed -E "s/^var ${TAG_VAR} = \"([^\"]*)\".*/\\1/" || true)"
if [ -z "$EXISTING_TAG" ]; then
  echo "required-image-tag.sh: could not find 'var ${TAG_VAR} = \"...\"' in $VERSIONCONSTANTS_FILE" >&2
  exit 1
fi

# Exact, not a trailing-hash match: a value still in the old <branch>-<hash>
# form is what ddev would pull, and it isn't this tag, so it counts as stale
# and `make` migrates the line.
if [ "$EXISTING_TAG" = "$CURRENT_HASH" ]; then
  echo "committed ${CURRENT_HASH}"
else
  echo "recomputed ${CURRENT_HASH}"
fi
