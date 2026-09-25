#!/usr/bin/env bash
# image-tag-keep-set.sh [<ref> ...]
#
# Prints every image tag a ddev build may still pull, one per line: the *Tag
# values in the version file at every v* release tag, at each <ref> given
# (open pull request heads), and on the main branch at any point in the last
# KEEP_MAIN_DAYS days. image-tag-cleanup-candidates.sh never lists these.
# Fails rather than printing a short list, since a missing tag here is a tag
# that can be deleted.
#
# Env:
#   KEEP_MAIN_REF   - main branch ref (default upstream/main, else origin/main)
#   KEEP_MAIN_DAYS  - how far back main's tags are kept (default 90)

set -eu -o pipefail

KEEP_MAIN_DAYS="${KEEP_MAIN_DAYS:-90}"
if [ -z "${KEEP_MAIN_REF:-}" ]; then
  KEEP_MAIN_REF=origin/main
  git rev-parse -q --verify upstream/main >/dev/null && KEEP_MAIN_REF=upstream/main
fi

# Releases before v1.21 kept their tags in pkg/version.
VERSION_FILES=(pkg/versionconstants/versionconstants.go pkg/version/version.go)

die() {
  echo "image-tag-keep-set.sh: $*" >&2
  exit 1
}

OUT="$(mktemp)"
trap 'rm -f "$OUT"' EXIT

# Returns 1 when <commit> has no version file at all.
tags_at() {
  local commit="$1" f content tags
  for f in "${VERSION_FILES[@]}"; do
    content="$(git show "${commit}:${f}" 2>/dev/null)" || continue
    tags="$(sed -nE 's/^var [A-Za-z0-9_]*Tag = "([^"]+)".*/\1/p' <<< "$content")"
    [ -n "$tags" ] || die "${commit}:${f} names no image tags"
    echo "$tags" >> "$OUT"
    return 0
  done
  return 1
}

releases=0
skipped=0
while IFS= read -r tag; do
  if tags_at "refs/tags/${tag}"; then
    releases=$((releases + 1))
  else
    skipped=$((skipped + 1))
  fi
done < <(git tag -l 'v*')
[ "$releases" -gt 0 ] || die "no v* release tag has a version file; were tags fetched?"

git rev-parse -q --verify "${KEEP_MAIN_REF}^{commit}" >/dev/null || die "main ref '${KEEP_MAIN_REF}' not found"
# The commit in effect when the window opened, plus every later change.
MAIN_COMMITS="$(
  git rev-list -1 --before="${KEEP_MAIN_DAYS} days ago" "$KEEP_MAIN_REF"
  git rev-list --since="${KEEP_MAIN_DAYS} days ago" "$KEEP_MAIN_REF" -- "${VERSION_FILES[@]}"
  git rev-parse "${KEEP_MAIN_REF}^{commit}"
)"
main_commits=0
for commit in $MAIN_COMMITS; do
  tags_at "$commit" || die "${KEEP_MAIN_REF} commit ${commit} has no version file"
  main_commits=$((main_commits + 1))
done

for ref in "$@"; do
  git rev-parse -q --verify "${ref}^{commit}" >/dev/null || die "ref '${ref}' not found"
  tags_at "$ref" || die "ref '${ref}' has no version file"
done

sort -u "$OUT"
echo "image-tag-keep-set.sh: $(sort -u "$OUT" | wc -l | tr -d ' ') tags from ${releases} releases (${skipped} without a version file), ${main_commits} ${KEEP_MAIN_REF} commits, $# other refs" >&2
