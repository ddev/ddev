#!/usr/bin/env bash
# image-tag-cleanup.sh [--execute [--yes]] [--tag <org/repo:tag>... | --tags-from <file|->]
#                      [--keep-set <file>] [--no-open-prs] [--decisions <file>]
#                      [--docker-org <org>] [--image-repo <name>]...
#                      [--older-than-days <n>] [--not-pulled-for-days <n>]
#                      [--keep-main-branch-days <n>] [--main-branch-ref <ref>]
#                      [--github-repo <owner/name>] [--max-delete <n>] [--limit <n>]
#                      [--save-keep-set <file>] [--save-candidates <file>]
#
# Finds unused Docker Hub image tags and, only with --execute, removes them.
# Run it from a checkout with tags fetched, with gh installed and logged in.
# Flags take their value as "--flag value" or "--flag=value". Each step is a
# script of its own, and each takes the flags shown here:
# image-tag-keep-set.sh, image-tag-cleanup-candidates.sh, delete-image-tags.sh.
#
# Examples:
#   containers/image-tag-cleanup.sh                     # what would be deleted
#   containers/image-tag-cleanup.sh --older-than-days 10 --not-pulled-for-days 0
#   containers/image-tag-cleanup.sh --execute                    # asks first; needs DOCKERHUB_*
#
# Flags:
#   --execute                   delete for real; without it, only print.
#                               Deleting every candidate asks first, or needs
#                               --yes without a terminal.
#   --older-than-days <n>       only list tags pushed more than n days ago [90]
#   --not-pulled-for-days <n>   only list tags not pulled in the last n days;
#                               0 ignores pulls [30]
#   --keep-main-branch-days <n> keep every tag the main branch's version file
#                               has named in the last n days [90]
#   --tag, --tags-from <file|->  delete only these tags, not every candidate;
#                               each must still be a candidate
#   --no-open-prs               leave open pull requests' tags out of the
#                               keep-set (unsafe: they may still need them)
#   --yes                       don't ask before deleting
#   --keep-set <file>           use this keep-set instead of building one
#   --save-keep-set <file>, --save-candidates <file>
#                               also write the keep-set or the candidates
#   --decisions <file>          also write every tag's keep/delete reason (TSV)
#   --max-delete <n>            refuse to delete more than n tags, and delete
#                               none rather than the first n [1000]
#   --limit <n>                 act on only the first n candidates, sorted; the
#                               rest stay candidates for a later run
#   --docker-org <org>          Docker Hub organization [ddev]
#   --image-repo <name>         Docker Hub repository to check; repeatable
#                               [those in image-configs.sh]
#   --github-repo <owner/name>  repository whose pull requests are kept
#                               [ddev/ddev]
#   --main-branch-ref <ref>     the main branch [upstream/main, else origin/main]
# The environment variables listed in the other scripts' --help set the same
# things. Credentials are environment only:
#   DOCKERHUB_USERNAME, DOCKERHUB_TOKEN - needed with --execute

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${1:-}" in -h | --help) sed -n '2,/^$/s/^# \{0,1\}//p' "$0"; exit 0 ;; esac

# shellcheck source=containers/image-tag-args.sh
source "$SCRIPT_DIR/image-tag-args.sh"
expand_equals_args "$@"
set -- ${EXPANDED_ARGS[@]+"${EXPANDED_ARGS[@]}"}

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
LIMIT="${CLEANUP_LIMIT:-}"
KEEP_ARGS=()
TAG_ARGS=()
FLAG_REPOS=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --execute) DELETE=true; shift; continue ;;
    --yes) YES=true; shift; continue ;;
    --no-open-prs) OPEN_PRS=false; shift; continue ;;
  esac
  [ "$#" -ge 2 ] || die "unknown argument '$1', or it needs a value; see the header of $0"
  case "$1" in
    --keep-set) KEEP_SET="$2" ;;
    --save-keep-set) SAVE_KEEP="$2" ;;
    --save-candidates) SAVE_CANDIDATES="$2" ;;
    --decisions) EXPLAIN=(--decisions "$2") ;;
    --tag | --tags-from) TAG_ARGS+=("$1" "$2") ;;
    --docker-org) export DOCKER_ORG="$2" ;;
    --image-repo) FLAG_REPOS="${FLAG_REPOS} $2" ;;
    --older-than-days) export CLEANUP_OLDER_THAN_DAYS="$2" ;;
    --not-pulled-for-days) export CLEANUP_NOT_PULLED_FOR_DAYS="$2" ;;
    --max-delete) export CLEANUP_MAX_DELETE="$2" ;;
    --limit) LIMIT="$2" ;;
    --keep-main-branch-days | --main-branch-ref) KEEP_ARGS+=("$1" "$2") ;;
    --github-repo) KEEP_ARGS+=(--github-repo "$2") ;;
    *) die "unknown argument '$1'; see the header of $0" ;;
  esac
  shift 2
done
[ -z "$FLAG_REPOS" ] || export CLEANUP_IMAGE_REPOS="$FLAG_REPOS"
[ "$DELETE" != true ] || require_dockerhub_credentials
[ -z "$LIMIT" ] || [[ "$LIMIT" =~ ^[0-9]+$ ]] || die "--limit must be a number"
[ -z "$LIMIT" ] || [ "${#TAG_ARGS[@]}" -eq 0 ] || die "--limit applies to the candidates found, not to --tag or --tags-from"

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
  if [ -n "$LIMIT" ]; then
    sort "$WORKDIR/candidates" | head -n "$LIMIT" > "$WORKDIR/limited"
    mv "$WORKDIR/limited" "$WORKDIR/candidates"
  fi
  count="$(wc -l < "$WORKDIR/candidates" | tr -d ' ')"
  if [ "$count" -eq 0 ]; then
    echo "image-tag-cleanup.sh: no tags to delete"
    exit 0
  fi
  if [ "$DELETE" != true ]; then
    sort "$WORKDIR/candidates" | sed 's/^/would delete /'
    echo "image-tag-cleanup.sh: dry run; ${count} tags would be deleted with --execute" >&2
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
