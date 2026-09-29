#!/usr/bin/env bash
# delete-image-tags.sh --keep-set <file> (--tag <org/repo:tag>... | --tags-from <file|->)
#                      [--execute] [--docker-org <org>] [--max-delete <n>]
#                      [--older-than-days <n>] [--not-pulled-for-days <n>]
#
# Example, a dry run of a report's candidates:
#   containers/delete-image-tags.sh --keep-set keep-set.txt --tags-from candidates.txt
#
# Deletes the named Docker Hub tags. Every one must still be listed by a fresh
# run of image-tag-cleanup-candidates.sh, or nothing is deleted, so a stale or
# hand-edited list can't remove a tag that has since become needed. Without
# --execute it only prints what it would delete. Most people want
# image-tag-cleanup.sh, which finds the candidates and calls this.
#
# Flags (each also settable by the environment variable in brackets):
#   --keep-set <file>       required: the tags to keep, as written by
#                           image-tag-keep-set.sh; none of them can be deleted
#   --tag <org/repo:tag>    a tag to delete, e.g. ddev/ddev-webserver:20250612_foo;
#                           repeatable
#   --tags-from <file>      tags to delete, separated by whitespace or commas;
#                           "-" reads stdin
#   --execute               delete for real; without it, only print
#   --docker-org <org>      the only organization allowed [DOCKER_ORG, ddev]
#   --max-delete <n>        refuse longer lists [CLEANUP_MAX_DELETE, 1000]
#   --older-than-days <n>, --not-pulled-for-days <n>
#                           the candidate rules, as in image-tag-cleanup-candidates.sh
# Credentials, environment only so they stay out of process listings:
#   DOCKERHUB_USERNAME, DOCKERHUB_TOKEN - needed with --execute

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${1:-}" in -h | --help) sed -n '2,/^$/s/^# \{0,1\}//p' "$0"; exit 0 ;; esac

# shellcheck source=containers/image-tag-args.sh
source "$SCRIPT_DIR/image-tag-args.sh"
expand_equals_args "$@"
set -- ${EXPANDED_ARGS[@]+"${EXPANDED_ARGS[@]}"}

HUB_API="https://hub.docker.com"
export DOCKER_ORG="${DOCKER_ORG:-ddev}"
CLEANUP_MAX_DELETE="${CLEANUP_MAX_DELETE:-1000}"

die() {
  echo "delete-image-tags.sh: $*" >&2
  exit 1
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

EXECUTE=false
KEEP_SET=""
: > "$WORKDIR/given"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--execute" ]; then
    EXECUTE=true
    shift
    continue
  fi
  [ "$#" -ge 2 ] || die "unknown argument '$1', or it needs a value; see the header of $0"
  case "$1" in
    --keep-set) KEEP_SET="$2" ;;
    --tag) echo "$2" >> "$WORKDIR/given" ;;
    --tags-from) if [ "$2" = "-" ]; then cat; else cat "$2"; fi >> "$WORKDIR/given" ;;
    --docker-org) export DOCKER_ORG="$2" ;;
    --max-delete) CLEANUP_MAX_DELETE="$2" ;;
    --older-than-days) export CLEANUP_OLDER_THAN_DAYS="$2" ;;
    --not-pulled-for-days) export CLEANUP_NOT_PULLED_FOR_DAYS="$2" ;;
    *) die "unknown argument '$1'; see the header of $0" ;;
  esac
  shift 2
done
[ -n "$KEEP_SET" ] || die "--keep-set <file> is required"
tr -s ', \t' '\n' < "$WORKDIR/given" | sed '/^$/d' | sort -u > "$WORKDIR/requested"

count="$(wc -l < "$WORKDIR/requested" | tr -d ' ')"
[ "$count" -gt 0 ] || die "no tags given"
[ "$count" -le "$CLEANUP_MAX_DELETE" ] || die "${count} tags exceeds CLEANUP_MAX_DELETE=${CLEANUP_MAX_DELETE}"

while IFS= read -r ref; do
  [[ "$ref" =~ ^[^/:]+/[^/:]+:[^/:]+$ ]] || die "'${ref}' is not <org>/<repo>:<tag>"
  "$SCRIPT_DIR/validate-image-repo.sh" "${ref%%:*}" || die "'${ref}' is not in a repository this project publishes"
done < "$WORKDIR/requested"

CLEANUP_IMAGE_REPOS="$(sed -E 's|^[^/]+/([^:]+):.*|\1|' "$WORKDIR/requested" | sort -u | tr '\n' ' ')" \
  "$SCRIPT_DIR/image-tag-cleanup-candidates.sh" --keep-set "$KEEP_SET" | sort -u > "$WORKDIR/candidates"

comm -23 "$WORKDIR/requested" "$WORKDIR/candidates" > "$WORKDIR/refused"
if [ -s "$WORKDIR/refused" ]; then
  echo "delete-image-tags.sh: these are not cleanup candidates now; deleting nothing:" >&2
  sed 's/^/  /' "$WORKDIR/refused" >&2
  exit 1
fi

if [ "$EXECUTE" != true ]; then
  sed 's/^/would delete /' "$WORKDIR/requested"
  echo "delete-image-tags.sh: dry run; ${count} tags would be deleted with --execute" >&2
  exit 0
fi

: "${DOCKERHUB_USERNAME:?delete-image-tags.sh: DOCKERHUB_USERNAME must be set with --execute}"
: "${DOCKERHUB_TOKEN:?delete-image-tags.sh: DOCKERHUB_TOKEN must be set with --execute}"
TOKEN="$(jq -n --arg u "$DOCKERHUB_USERNAME" --arg p "$DOCKERHUB_TOKEN" '{username: $u, password: $p}' |
  curl -fsS -H "Content-Type: application/json" -X POST --data @- "${HUB_API}/v2/users/login/" | jq -r '.token // empty')" ||
  die "Docker Hub login failed"
[ -n "$TOKEN" ] || die "Docker Hub login returned no token"

failed=0
while IFS= read -r ref; do
  repo="${ref%%:*}"
  tag="${ref#*:}"
  if curl -fsS -o /dev/null -X DELETE -H "Authorization: JWT ${TOKEN}" "${HUB_API}/v2/repositories/${repo}/tags/${tag}/"; then
    echo "deleted ${ref}"
  else
    echo "delete-image-tags.sh: failed to delete ${ref}" >&2
    failed=$((failed + 1))
  fi
done < "$WORKDIR/requested"

[ "$failed" -eq 0 ] || die "${failed} of ${count} deletions failed"
