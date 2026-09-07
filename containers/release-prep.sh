#!/usr/bin/env bash
# release-prep.sh [<release-tag>]
#
# Prepares the source-file half of a release. Stamps the release tag into every
# image's Dockerfile so its content hash moves and CI rebuilds it, points
# versionconstants.go at the resulting hashes, and records the release in each
# <TagVar>Branch for `ddev version`.
#
# Builds nothing and commits nothing: the images come from the pull request's
# own CI run, which is also what tests them before the release is cut.
#
# Env:
#   VERSIONCONSTANTS_FILE - path to versionconstants.go

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git rev-parse --show-toplevel)"
VERSIONCONSTANTS_FILE="${VERSIONCONSTANTS_FILE:-$REPO_ROOT/pkg/versionconstants/versionconstants.go}"

# A Dockerfile comment is stripped by the parser, so stamping one changes the
# content hash without changing a single layer of the image it produces.
MARKER='# ddev-release-marker:'

TAG="${1:-}"
if [ -z "$TAG" ]; then
  if [ ! -t 0 ]; then
    echo "Usage: $0 <release-tag>   (for example: $0 v1.25.4)" >&2
    exit 2
  fi
  read -r -p "Release tag (vX.Y.Z): " TAG
fi

if ! [[ "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9.]*)?$ ]]; then
  echo "release-prep.sh: '${TAG}' is not a release tag (vX.Y.Z, or a prerelease like v1.25.4-rc1)" >&2
  exit 1
fi

# shellcheck source=containers/image-configs.sh
source "$SCRIPT_DIR/image-configs.sh"

tag_value() {
  grep -E "^var $1 = " "$VERSIONCONSTANTS_FILE" | sed -E "s/^var $1 = \"([^\"]*)\".*/\\1/"
}

TAG_VARS=()
DOCKERFILES=()
OLD_TAGS=()
SEEN=""
for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
  IFS='|' read -r _ tag_var _ make_dir _ <<< "$entry"
  # Every ddev-dbserver variant shares BaseDBTag and one Dockerfile.
  case " $SEEN " in *" $tag_var "*) continue ;; esac
  SEEN="$SEEN $tag_var"

  dockerfile="$REPO_ROOT/containers/${make_dir}/Dockerfile"
  if [ ! -f "$dockerfile" ]; then
    echo "release-prep.sh: no Dockerfile at ${dockerfile}" >&2
    exit 1
  fi
  TAG_VARS+=("$tag_var")
  DOCKERFILES+=("$dockerfile")
  OLD_TAGS+=("$(tag_value "$tag_var")")
done

for dockerfile in "${DOCKERFILES[@]}"; do
  if grep -q "^${MARKER} " "$dockerfile"; then
    sed -i.bak -E "s|^${MARKER} .*|${MARKER} ${TAG}|" "$dockerfile"
    rm -f "${dockerfile}.bak"
  else
    printf '%s %s\n' "$MARKER" "$TAG" >> "$dockerfile"
  fi
done

"$SCRIPT_DIR/retag-images.sh"

# Overwrites what retag-images.sh just wrote, which is the current branch. The
# trailing // <branch>-<hash> comment is left as the branch, since that is the
# alias the push actually publishes - validate-image-tag.sh rejects a vX.Y.Z one.
for tag_var in "${TAG_VARS[@]}"; do
  sed -i.bak -E "s|^var ${tag_var}Branch = \"[^\"]*\"|var ${tag_var}Branch = \"${TAG}\"|" "$VERSIONCONSTANTS_FILE"
  rm -f "${VERSIONCONSTANTS_FILE}.bak"
done

echo
echo "release-prep.sh: prepared ${TAG}"
for i in "${!TAG_VARS[@]}"; do
  printf '  %-18s %s -> %s\n' "${TAG_VARS[$i]}" "${OLD_TAGS[$i]}" "$(tag_value "${TAG_VARS[$i]}")"
done
echo
echo "Nothing was built or committed. Commit the result and open a pull request;"
echo "CI builds and pushes these images, and the tests run against them."
