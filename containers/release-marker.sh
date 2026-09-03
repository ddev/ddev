#!/usr/bin/env bash
# release-marker.sh <image-dir>
#
# Prints the release tag release-prep.sh stamped into
# containers/<image-dir>/Dockerfile, or nothing when there is no marker.
#
# Exits non-zero on a marker that is present but unusable, so a typo fails the
# build rather than publishing a tag nobody meant.

set -eu -o pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
MARKER='# ddev-release-marker:'

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <image-dir>" >&2
  exit 2
fi

DOCKERFILE="$REPO_ROOT/containers/$1/Dockerfile"
if [ ! -f "$DOCKERFILE" ]; then
  echo "release-marker.sh: no Dockerfile at ${DOCKERFILE}" >&2
  exit 2
fi

MARKERS="$(grep "^${MARKER} " "$DOCKERFILE" || true)"
[ -n "$MARKERS" ] || exit 0

if [ "$(printf '%s\n' "$MARKERS" | wc -l)" -ne 1 ]; then
  echo "release-marker.sh: ${DOCKERFILE} carries more than one release marker" >&2
  exit 1
fi

TAG="${MARKERS#"${MARKER}" }"
if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9.]*)?$ ]]; then
  echo "release-marker.sh: '${TAG}' in ${DOCKERFILE} is not a release tag (vX.Y.Z, or a prerelease like v1.25.4-rc1)" >&2
  exit 1
fi

echo "$TAG"
