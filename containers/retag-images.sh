#!/usr/bin/env bash
# retag-images.sh
#
# Rewrites every changed image's tag in versionconstants.go without building
# anything, for a change whose images CI will build. The image list comes from
# image-configs.sh rather than being repeated here.

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKER_ORG="${DOCKER_ORG:-ddev}"

# shellcheck source=containers/image-configs.sh
source "$SCRIPT_DIR/image-configs.sh"

SEEN=""
for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
  IFS='|' read -r repo_suffix tag_var hash_paths _ <<< "$entry"
  # Every ddev-dbserver variant shares BaseDBTag; rewrite each variable once.
  case " $SEEN " in *" $tag_var "*) continue ;; esac
  SEEN="$SEEN $tag_var"

  # shellcheck disable=SC2086 # hash_paths is a space-separated path list
  "$SCRIPT_DIR/autotag.sh" --no-build "$tag_var" "${DOCKER_ORG}/${repo_suffix}" $hash_paths
done
