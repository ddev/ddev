#!/usr/bin/env bash
# image-tag-cleanup.sh [--delete [--yes]] [--tag <org/repo:tag>... | --tags-from <file|->]
#                      [--keep-set <file>] [--no-open-prs] [--explain <file>]
#                      [--org <org>] [--repo <name>]... [--min-age-days <n>]
#                      [--pull-grace-days <n>] [--main-days <n>] [--main-ref <ref>]
#                      [--github-repo <owner/name>] [--max-delete <n>]
#                      [--save-keep-set <file>] [--save-candidates <file>]
#
# Finds unused Docker Hub image tags and, only with --delete, removes them.
# Run it from a checkout with tags fetched, with gh installed and logged in.
# Each step is a script of its own, and each takes the flags shown here:
# image-tag-keep-set.sh, image-tag-cleanup-candidates.sh, delete-image-tags.sh.
#
# Examples:
#   containers/image-tag-cleanup.sh                     # what would be deleted
#   containers/image-tag-cleanup.sh --min-age-days 10 --pull-grace-days 0
#   containers/image-tag-cleanup.sh --delete            # asks first; needs DOCKERHUB_*
#
# Flags not described in the scripts' own headers:
#   --delete            delete for real; without it, only print. Deleting
#                       every candidate asks first, or needs --yes without a terminal.
#   --yes               don't ask before deleting
#   --save-keep-set, --save-candidates <file>
#                       also write the keep-set, or the candidate list, to <file>
#   --tag, --tags-from  delete only these tags, not every candidate; each must
#                       still be a candidate
#   --keep-set <file>   use this keep-set instead of building one
#   --no-open-prs       leave open pull requests' tags out of the keep-set
#                       (unsafe: an open pull request may still need them)
#   --github-repo <o/n> repository whose pull requests and version files matter
#                       [GITHUB_REPOSITORY, ddev/ddev]
# The rest are described in image-tag-cleanup-candidates.sh, image-tag-keep-set.sh
# and delete-image-tags.sh. Credentials stay in the environment:
#   DOCKERHUB_USERNAME, DOCKERHUB_TOKEN - needed with --delete

set -eu -o pipefail

case "${1:-}" in -h | --help) sed -n '2,/^$/s/^# \{0,1\}//p' "$0"; exit 0 ;; esac

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

die() {
  echo "image-tag-cleanup.sh: $*" >&2
  exit 1
}

DELETE=false
YES=false
OPEN_PRS=true
KEEP_SET=""
EXPLAIN=()
SAVE_KEEP=""
SAVE_CANDIDATES=""
KEEP_ARGS=()
TAG_ARGS=()
FLAG_REPOS=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --delete) DELETE=true; shift; continue ;;
    --yes) YES=true; shift; continue ;;
    --no-open-prs) OPEN_PRS=false; shift; continue ;;
  esac
  [ "$#" -ge 2 ] || die "unknown argument '$1', or it needs a value; see the header of $0"
  case "$1" in
    --keep-set) KEEP_SET="$2" ;;
    --save-keep-set) SAVE_KEEP="$2" ;;
    --save-candidates) SAVE_CANDIDATES="$2" ;;
    --explain) EXPLAIN=(--explain "$2") ;;
    --tag | --tags-from) TAG_ARGS+=("$1" "$2") ;;
    --org) export DOCKER_ORG="$2" ;;
    --repo) FLAG_REPOS="${FLAG_REPOS} $2" ;;
    --min-age-days) export CLEANUP_MIN_AGE_DAYS="$2" ;;
    --pull-grace-days) export CLEANUP_PULL_GRACE_DAYS="$2" ;;
    --max-delete) export CLEANUP_MAX_DELETE="$2" ;;
    --main-days | --main-ref) KEEP_ARGS+=("$1" "$2") ;;
    --github-repo) KEEP_ARGS+=(--github-repo "$2") ;;
    *) die "unknown argument '$1'; see the header of $0" ;;
  esac
  shift 2
done
[ -z "$FLAG_REPOS" ] || export CLEANUP_REPOS="$FLAG_REPOS"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

if [ -z "$KEEP_SET" ]; then
  KEEP_SET="$WORKDIR/keep-set"
  [ "$OPEN_PRS" != true ] || KEEP_ARGS+=(--open-prs)
  "$SCRIPT_DIR/image-tag-keep-set.sh" ${KEEP_ARGS[@]+"${KEEP_ARGS[@]}"} > "$KEEP_SET"
fi

[ -z "$SAVE_KEEP" ] || cp "$KEEP_SET" "$SAVE_KEEP"

if [ "${#TAG_ARGS[@]}" -eq 0 ]; then
  "$SCRIPT_DIR/image-tag-cleanup-candidates.sh" --keep-set "$KEEP_SET" ${EXPLAIN[@]+"${EXPLAIN[@]}"} > "$WORKDIR/candidates"
  [ -z "$SAVE_CANDIDATES" ] || cp "$WORKDIR/candidates" "$SAVE_CANDIDATES"
  count="$(wc -l < "$WORKDIR/candidates" | tr -d ' ')"
  if [ "$count" -eq 0 ]; then
    echo "image-tag-cleanup.sh: no tags to delete"
    exit 0
  fi
  if [ "$DELETE" != true ]; then
    sort "$WORKDIR/candidates" | sed 's/^/would delete /'
    echo "image-tag-cleanup.sh: dry run; ${count} tags would be deleted with --delete" >&2
    exit 0
  fi
  TAG_ARGS=(--tags-from "$WORKDIR/candidates")
  if [ "$YES" != true ]; then
    [ -t 0 ] || die "deleting every candidate without a terminal to ask needs --yes"
    read -r -p "Delete ${count} tags? [y/N] " answer
    [ "$answer" = y ] || die "not deleting"
  fi
fi

EXEC=()
[ "$DELETE" != true ] || EXEC=(--execute)
"$SCRIPT_DIR/delete-image-tags.sh" --keep-set "$KEEP_SET" ${EXEC[@]+"${EXEC[@]}"} "${TAG_ARGS[@]}"
