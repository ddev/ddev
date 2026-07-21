#!/usr/bin/env bash
# autotag.sh [--print-only] <TagVarName> <ImageRepo> <hash-path> [<hash-path> ...] [-- <build-cmd...>]
#
# Detects whether an image's content hash (see hash-paths.sh) differs from
# the hash embedded in the tag currently committed in
# pkg/versionconstants/versionconstants.go. If unchanged, does nothing (no
# Docker, no network). If changed, builds the image locally at the new
# <branch>-<hash> tag (unless already built) using the given build command,
# then rewrites just that tag's line in versionconstants.go in place.
#
# --print-only prints the candidate <branch>-<hash> tag and exits, without
# touching Docker or versionconstants.go.
#
# Env:
#   HASH_LEN               - hash length in hex chars (default 10, must match hash-paths.sh)
#   VERSIONCONSTANTS_FILE   - path to versionconstants.go (default pkg/versionconstants/versionconstants.go)

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git rev-parse --show-toplevel)"
HASH_LEN="${HASH_LEN:-10}"
VERSIONCONSTANTS_FILE="${VERSIONCONSTANTS_FILE:-$REPO_ROOT/pkg/versionconstants/versionconstants.go}"

PRINT_ONLY=false
if [ "${1:-}" = "--print-only" ]; then
  PRINT_ONLY=true
  shift
fi

if [ "$#" -lt 3 ]; then
  echo "Usage: $0 [--print-only] <TagVarName> <ImageRepo> <hash-path> [<hash-path> ...] [-- <build-cmd...>]" >&2
  exit 1
fi

TAG_VAR="$1"; shift
IMAGE_REPO="$1"; shift

HASH_PATHS=()
BUILD_CMD=()
SEEN_DASHDASH=false
for arg in "$@"; do
  if [ "$SEEN_DASHDASH" = false ] && [ "$arg" = "--" ]; then
    SEEN_DASHDASH=true
    continue
  fi
  if [ "$SEEN_DASHDASH" = true ]; then
    BUILD_CMD+=("$arg")
  else
    HASH_PATHS+=("$arg")
  fi
done

if [ "${#HASH_PATHS[@]}" -eq 0 ]; then
  echo "autotag.sh: no hash paths given" >&2
  exit 1
fi

CURRENT_HASH="$(HASH_LEN="$HASH_LEN" "$SCRIPT_DIR/hash-paths.sh" "${HASH_PATHS[@]}")"

BRANCH="$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo detached)"
SANITIZED_BRANCH="$(echo "$BRANCH" | sed -E 's/[^A-Za-z0-9_.-]+/-/g')"
CANDIDATE_TAG="${SANITIZED_BRANCH}-${CURRENT_HASH}"

if [ "$PRINT_ONLY" = true ]; then
  echo "$CANDIDATE_TAG"
  exit 0
fi

EXISTING_TAG="$(grep -E "^var ${TAG_VAR} = " "$VERSIONCONSTANTS_FILE" | sed -E "s/^var ${TAG_VAR} = \"([^\"]*)\".*/\\1/")"
if [ -z "$EXISTING_TAG" ]; then
  echo "autotag.sh: could not find 'var ${TAG_VAR} = \"...\"' in $VERSIONCONSTANTS_FILE" >&2
  exit 1
fi

EXISTING_HASH="${EXISTING_TAG: -${HASH_LEN}}"

if [ "$EXISTING_HASH" = "$CURRENT_HASH" ]; then
  # Unchanged - nothing to do.
  exit 0
fi

echo "autotag.sh: ${TAG_VAR} content changed (${EXISTING_TAG} -> ${CANDIDATE_TAG})"

if docker image inspect "${IMAGE_REPO}:${CANDIDATE_TAG}" >/dev/null 2>&1; then
  echo "autotag.sh: ${IMAGE_REPO}:${CANDIDATE_TAG} already built locally, skipping build"
else
  if [ "${#BUILD_CMD[@]}" -eq 0 ]; then
    echo "autotag.sh: no build command given, cannot build ${IMAGE_REPO}:${CANDIDATE_TAG}" >&2
    exit 1
  fi
  "${BUILD_CMD[@]}" "VERSION=${CANDIDATE_TAG}"
fi

sed -i.bak -E "s/^var ${TAG_VAR} = \"[^\"]*\"/var ${TAG_VAR} = \"${CANDIDATE_TAG}\"/" "$VERSIONCONSTANTS_FILE"
rm -f "${VERSIONCONSTANTS_FILE}.bak"

echo "autotag.sh: updated ${TAG_VAR} to ${CANDIDATE_TAG} in $VERSIONCONSTANTS_FILE"
