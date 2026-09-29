#!/usr/bin/env bash
# delete-image-tags.sh [--execute] [--keep-set <file>] [<org>/<repo>:<tag> ...]
#
# Example, a dry run listing every tag that would be deleted:
#   containers/delete-image-tags.sh
#
# Deletes the named Docker Hub tags, read from the arguments or else from
# stdin (separated by whitespace or commas). Every one must still be listed by
# a fresh run of image-tag-cleanup-candidates.sh, or nothing is deleted, so a
# stale or hand-edited list can't remove a tag that has since become needed.
# Without --execute it only prints what it would delete. With no tags given
# and stdin a terminal, it names every current candidate; --execute is
# refused in that mode.
#
# Arguments:
#   --execute            actually delete; without it, only print
#   --keep-set <file>    the tags to keep, one per line, as written by
#                        image-tag-keep-set.sh (releases, open PRs, recent
#                        main). None of them can be deleted here. Omitted, it
#                        is built by running image-tag-keep-set.sh.
#   <org>/<repo>:<tag>   what to delete, e.g. ddev/ddev-webserver:20250612_foo
#
# Env:
#   DOCKER_ORG                        - the only organization allowed (default ddev)
#   DOCKERHUB_USERNAME, DOCKERHUB_TOKEN - credentials, needed with --execute
#   CLEANUP_MAX_DELETE                - refuse longer lists (default 1000)

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HUB_API="https://hub.docker.com"
export DOCKER_ORG="${DOCKER_ORG:-ddev}"
CLEANUP_MAX_DELETE="${CLEANUP_MAX_DELETE:-1000}"

die() {
  echo "delete-image-tags.sh: $*" >&2
  exit 1
}

EXECUTE=false
KEEP_SET=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --execute) EXECUTE=true; shift ;;
    --keep-set) [ "$#" -ge 2 ] || die "--keep-set needs a file"; KEEP_SET="$2"; shift 2 ;;
    -*) die "usage: $0 [--execute] [--keep-set <file>] [<org>/<repo>:<tag> ...]" ;;
    *) break ;;
  esac
done

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

if [ -z "$KEEP_SET" ]; then
  KEEP_SET="$WORKDIR/keep-set"
  "$SCRIPT_DIR/image-tag-keep-set.sh" > "$KEEP_SET"
fi

if [ "$#" -gt 0 ]; then
  printf '%s\n' "$@"
elif [ -t 0 ]; then
  [ "$EXECUTE" != true ] || die "--execute needs the tags to delete, as arguments or on stdin"
  "$SCRIPT_DIR/image-tag-cleanup-candidates.sh" "$KEEP_SET"
else
  cat
fi | tr -s ', \t' '\n' | sed '/^$/d' | sort -u > "$WORKDIR/requested"

count="$(wc -l < "$WORKDIR/requested" | tr -d ' ')"
[ "$count" -gt 0 ] || die "no tags to delete"
[ "$count" -le "$CLEANUP_MAX_DELETE" ] || die "${count} tags exceeds CLEANUP_MAX_DELETE=${CLEANUP_MAX_DELETE}"

while IFS= read -r ref; do
  [[ "$ref" =~ ^[^/:]+/[^/:]+:[^/:]+$ ]] || die "'${ref}' is not <org>/<repo>:<tag>"
  "$SCRIPT_DIR/validate-image-repo.sh" "${ref%%:*}" || die "'${ref}' is not in a repository this project publishes"
done < "$WORKDIR/requested"

CLEANUP_REPOS="$(sed -E 's|^[^/]+/([^:]+):.*|\1|' "$WORKDIR/requested" | sort -u | tr '\n' ' ')" \
  "$SCRIPT_DIR/image-tag-cleanup-candidates.sh" "$KEEP_SET" | sort -u > "$WORKDIR/candidates"

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
