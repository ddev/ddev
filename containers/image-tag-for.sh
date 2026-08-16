#!/usr/bin/env bash
# image-tag-for.sh <repo-suffix>
#
# Prints the tag this checkout needs for one image, by repository suffix
# (ddev-webserver, ddev-dbserver-mysql-8.0, ...). A thin lookup over
# image-configs.sh so the manual push workflows can default their tag input
# instead of asking a human to retype a content hash - the mismatch that
# causes is the whole reason wait-for-images.sh has to fail loudly.
#
# Exits non-zero for a repository the automatic flow doesn't cover
# (test-ssh-server, say), where there is no content hash to derive and the
# caller has to supply a tag.

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <repo-suffix>" >&2
  exit 2
fi

WANTED="$1"

# shellcheck source=containers/image-configs.sh
source "$SCRIPT_DIR/image-configs.sh"

for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
  IFS='|' read -r repo_suffix tag_var hash_paths _ <<< "$entry"
  [ "$repo_suffix" = "$WANTED" ] || continue
  # shellcheck disable=SC2086 # hash_paths is a space-separated path list
  read -r _ tag <<< "$("$SCRIPT_DIR/required-image-tag.sh" "$tag_var" $hash_paths)"
  echo "$tag"
  exit 0
done

echo "image-tag-for.sh: '${WANTED}' is not one of the content-addressed images; pass a tag explicitly" >&2
exit 1
