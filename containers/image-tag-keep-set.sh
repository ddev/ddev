#!/usr/bin/env bash
# image-tag-keep-set.sh [--open-prs] [--github-repo <owner/name>] [--ref <ref>]...
#                       [--main-ref <ref>] [--main-days <n>]
#
# Example, from a checkout with tags fetched:
#   containers/image-tag-keep-set.sh --open-prs > keep-set.txt
#
# Prints every image tag a ddev build may still pull, one per line: the *Tag
# values in the version file at every v* release tag, at each --ref, and on
# the main branch at any point in the last --main-days days. The cleanup
# scripts never delete these. Fails rather than printing a short list, since a
# missing tag here is a tag that can be deleted.
#
# Flags (each also settable by the environment variable in brackets):
#   --open-prs         also keep the tags of every open pull request's head,
#                      fetched with gh into refs/keep/. Needs gh and network.
#   --github-repo <o/n>  repository whose pull requests --open-prs reads
#                      [GITHUB_REPOSITORY, else ddev/ddev]
#   --ref <ref>        also keep the tags at this git ref; repeatable
#   --main-ref <ref>   main branch ref [KEEP_MAIN_REF, else upstream/main,
#                      else origin/main]
#   --main-days <n>    how far back main's tags are kept [KEEP_MAIN_DAYS, 90]

set -eu -o pipefail

case "${1:-}" in -h | --help) sed -n '2,/^$/s/^# \{0,1\}//p' "$0"; exit 0 ;; esac

KEEP_MAIN_DAYS="${KEEP_MAIN_DAYS:-90}"
KEEP_MAIN_REF="${KEEP_MAIN_REF:-}"
REPO="${GITHUB_REPOSITORY:-ddev/ddev}"
OPEN_PRS=false
REFS=()

die() {
  echo "image-tag-keep-set.sh: $*" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --open-prs) OPEN_PRS=true; shift ;;
    --github-repo | --ref | --main-ref | --main-days)
      [ "$#" -ge 2 ] || die "$1 needs a value"
      case "$1" in
        --github-repo) REPO="$2" ;;
        --ref) REFS+=("$2") ;;
        --main-ref) KEEP_MAIN_REF="$2" ;;
        --main-days) KEEP_MAIN_DAYS="$2" ;;
      esac
      shift 2 ;;
    *) die "unknown argument '$1'; see the header of $0" ;;
  esac
done

if [ -z "$KEEP_MAIN_REF" ]; then
  KEEP_MAIN_REF=origin/main
  git rev-parse -q --verify upstream/main >/dev/null && KEEP_MAIN_REF=upstream/main
fi

# Releases before v1.21 kept their tags in pkg/version.
VERSION_FILES=(pkg/versionconstants/versionconstants.go pkg/version/version.go)

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

# Stale refs from an earlier run would keep tags of pull requests since closed.
fetch_open_prs() {
  local numbers count
  command -v gh >/dev/null || die "--open-prs needs gh; install it or leave the flag off"
  git for-each-ref --format='delete %(refname)' refs/keep/ | git update-ref --stdin
  numbers="$(gh pr list --repo "$REPO" --state open --limit 1000 --json number --jq '.[].number')" ||
    die "listing open pull requests of ${REPO} failed"
  count="$(grep -c . <<< "$numbers" || true)"
  [ "$count" -lt 1000 ] || die "1000 open pull requests is the listing's limit; some may be missing"
  [ "$count" -eq 0 ] && return 0
  sed -E 's|.*|+refs/pull/&/head:refs/keep/pr-&|' <<< "$numbers" |
    xargs git fetch -q --no-tags "https://github.com/${REPO}.git" ||
    die "fetching open pull request heads of ${REPO} failed"
  while IFS= read -r ref; do REFS+=("$ref"); done < <(git for-each-ref --format='%(refname)' refs/keep/)
}
[ "$OPEN_PRS" != true ] || fetch_open_prs

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

for ref in ${REFS[@]+"${REFS[@]}"}; do
  git rev-parse -q --verify "${ref}^{commit}" >/dev/null || die "ref '${ref}' not found"
  tags_at "$ref" || die "ref '${ref}' has no version file"
done

sort -u "$OUT"
echo "image-tag-keep-set.sh: $(sort -u "$OUT" | wc -l | tr -d ' ') tags from ${releases} releases (${skipped} without a version file), ${main_commits} ${KEEP_MAIN_REF} commits, ${#REFS[@]} other refs" >&2
