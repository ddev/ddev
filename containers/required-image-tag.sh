#!/usr/bin/env bash
# required-image-tag.sh <TagVarName> <hash-path> [<hash-path> ...]
#
# Prints "<state> <tag>" for the image tag this checkout actually resolves to,
# so callers agree with what `ddev` will pull and with what autotag.sh would do:
#
#   committed <tag>   versionconstants.go's tag still ends in the current
#                     content hash, so `make` leaves it alone and that exact
#                     tag - branch prefix and all - is what gets pulled.
#   recomputed <tag>  the content changed, so autotag.sh rewrites
#                     versionconstants.go to <tag> and builds it locally.
#
# The branch prefix is only meaningful in the recomputed state: autotag.sh
# rewrites the whole tag when the hash changes, so a committed tag whose hash
# still matches keeps whatever branch last changed that image. Recomputing the
# prefix in the committed state produces a tag nothing ever pushed.
#
# Env:
#   HASH_LEN                  - hash length in hex chars (default 10, must
#                               match hash-paths.sh)
#   VERSIONCONSTANTS_FILE     - path to versionconstants.go
#   REQUIRED_IMAGE_TAG_BRANCH - branch for the recomputed prefix (default: the
#                               current git branch)

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

if [ "${EXISTING_TAG: -${HASH_LEN}}" = "$CURRENT_HASH" ]; then
  echo "committed ${EXISTING_TAG}"
  exit 0
fi

BRANCH="${REQUIRED_IMAGE_TAG_BRANCH:-$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo detached)}"
SANITIZED_BRANCH="$(echo "$BRANCH" | sed -E 's/[^A-Za-z0-9_.-]+/-/g')"
echo "recomputed ${SANITIZED_BRANCH}-${CURRENT_HASH}"
