#!/usr/bin/env bash
# check-image-tags.sh
#
# Fails when versionconstants.go names a tag that no longer matches the content
# under containers/, which happens whenever an image changes and `make` hasn't
# been run since. Left uncaught, that ships a binary pulling tags nothing
# builds, so this runs as part of `make staticrequired` - the one target both
# the pre-commit and pre-push hooks already insist on.
#
# Deliberately a check and not a fix: correcting it means building the changed
# image, which is minutes of Docker work and has no business happening inside a
# commit hook. This only compares hashes - no Docker, no network.

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=containers/image-configs.sh
source "$SCRIPT_DIR/image-configs.sh"

STALE=()
SEEN=""
for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
  IFS='|' read -r _ tag_var hash_paths _ <<< "$entry"
  # Every ddev-dbserver variant shares BaseDBTag; check each variable once.
  case " $SEEN " in *" $tag_var "*) continue ;; esac
  SEEN="$SEEN $tag_var"

  # shellcheck disable=SC2086 # hash_paths is a space-separated path list
  read -r state tag <<< "$("$SCRIPT_DIR/required-image-tag.sh" "$tag_var" $hash_paths)"
  [ "$state" = "committed" ] || STALE+=("${tag_var} -> ${tag}")
done

if [ "${#STALE[@]}" -eq 0 ]; then
  echo "check-image-tags.sh: versionconstants.go is up to date with containers/"
  exit 0
fi

echo "check-image-tags.sh: containers/ changed but versionconstants.go wasn't updated:" >&2
printf '  %s\n' "${STALE[@]}" >&2
echo "check-image-tags.sh: run 'make' to rebuild the changed image(s) and rewrite those tags, then commit the result." >&2
exit 1
