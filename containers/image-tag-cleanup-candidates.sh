#!/usr/bin/env bash
# image-tag-cleanup-candidates.sh --keep-set <file> [--explain <file>]
#                                 [--org <org>] [--repo <name>]...
#                                 [--min-age-days <n>] [--pull-grace-days <n>]
#
# Example:
#   containers/image-tag-cleanup-candidates.sh --keep-set keep-set.txt --explain decisions.tsv > candidates.txt
#
# Prints the Docker Hub tags that look safe to delete, one <org>/<repo>:<tag>
# per line. A candidate has a shape CI or a branch build produces, is absent
# from the keep-set (image-tag-keep-set.sh), was pushed long ago and not
# pulled lately, and shares no manifest with a kept tag. Release tags, latest,
# and unrecognized shapes are always kept.
#
# Flags (each also settable by the environment variable in brackets):
#   --keep-set <file>       required: the tags to keep, one per line, as
#                           written by image-tag-keep-set.sh. Never candidates.
#   --explain <file>        also write every tag's keep/delete decision and
#                           reason to <file> as TSV
#   --org <org>             Docker Hub organization [DOCKER_ORG, ddev]
#   --repo <name>           repository to check; repeatable
#                           [CLEANUP_REPOS, else those in image-configs.sh]
#   --min-age-days <n>      keep anything pushed more recently
#                           [CLEANUP_MIN_AGE_DAYS, 90]
#   --pull-grace-days <n>   keep anything pulled more recently
#                           [CLEANUP_PULL_GRACE_DAYS, 30]
# Test hooks, environment only: NOW (epoch seconds), HASH_LEN (default 10).

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

case "${1:-}" in -h | --help) sed -n '2,/^$/s/^# \{0,1\}//p' "$0"; exit 0 ;; esac

# shellcheck source=containers/image-tag-args.sh
source "$SCRIPT_DIR/image-tag-args.sh"
expand_equals_args "$@"
set -- ${EXPANDED_ARGS[@]+"${EXPANDED_ARGS[@]}"}

HUB_API="https://hub.docker.com"
DOCKER_ORG="${DOCKER_ORG:-ddev}"
CLEANUP_MIN_AGE_DAYS="${CLEANUP_MIN_AGE_DAYS:-90}"
CLEANUP_PULL_GRACE_DAYS="${CLEANUP_PULL_GRACE_DAYS:-30}"
NOW="${NOW:-$(date +%s)}"
HASH_LEN="${HASH_LEN:-10}"

die() {
  echo "image-tag-cleanup-candidates.sh: $*" >&2
  exit 1
}

EXPLAIN=""
KEEP_SET=""
FLAG_REPOS=""
while [ "$#" -gt 0 ]; do
  [ "$#" -ge 2 ] || die "unknown argument '$1', or it needs a value; see the header of $0"
  case "$1" in
    --keep-set) KEEP_SET="$2" ;;
    --explain) EXPLAIN="$2" ;;
    --org) DOCKER_ORG="$2" ;;
    --repo) FLAG_REPOS="${FLAG_REPOS} $2" ;;
    --min-age-days) CLEANUP_MIN_AGE_DAYS="$2" ;;
    --pull-grace-days) CLEANUP_PULL_GRACE_DAYS="$2" ;;
    *) die "unknown argument '$1'; see the header of $0" ;;
  esac
  shift 2
done
[ -z "$FLAG_REPOS" ] || CLEANUP_REPOS="$FLAG_REPOS"
[ -n "$KEEP_SET" ] || die "--keep-set <file> is required"
[ -s "$KEEP_SET" ] || die "keep-set file '${KEEP_SET}' is missing or empty"

if [ -z "${CLEANUP_REPOS:-}" ]; then
  # shellcheck source=containers/image-configs.sh
  source "$SCRIPT_DIR/image-configs.sh"
  for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
    IFS='|' read -r repo _ _ _ _ _ _ _ extra <<< "$entry"
    CLEANUP_REPOS="${CLEANUP_REPOS:-} ${repo} ${extra}"
  done
fi

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# Writes one JSON object per tag to stdout, following every page.
fetch_tags() {
  local url="${HUB_API}/v2/namespaces/${DOCKER_ORG}/repositories/$1/tags?page_size=100" page
  while [ -n "$url" ]; do
    # A `next` pointing elsewhere would let the listing answer for itself.
    case "$url" in "${HUB_API}/"*) ;; *) die "refusing to follow pagination to ${url}" ;; esac
    page="$(curl -fsSL --retry 5 --retry-delay 10 "$url")" || die "listing $1 failed at ${url}"
    jq -c '.results[] | {name, digest, last_updated, tag_last_pushed, tag_last_pulled}' <<< "$page"
    url="$(jq -r '.next // empty' <<< "$page")"
  done
}

# Deciding in two passes lets a kept tag protect every other name on its
# manifest, such as the hash tag a release tag points at.
# shellcheck disable=SC2016 # jq program, not shell
DECIDE='
  ($keep | split("\n") | map(select(length > 0) | {(.): true}) | add // {}) as $keepset
  | def ts: if . == null then null else sub("\\.[0-9]+Z$"; "Z") | fromdateiso8601 end;
  def ago: ($now - .) / 86400 | floor;
  def classify:
    if test("^[0-9a-f]{\($hl)}$") then {class: "hash", key: .}
    elif test("^v[0-9]") or IN("latest", "stable", "edge") then {class: "release", key: .}
    elif test("-(amd64|arm64)$") then {class: "arch", key: sub("-(amd64|arm64)$"; "")}
    elif test("^[A-Za-z0-9_][A-Za-z0-9_.-]*-[0-9a-f]{\($hl)}$") then {class: "alias", key: (match("[0-9a-f]{\($hl)}$").string)}
    elif test("^20[0-9]{6}_[A-Za-z0-9_.-]+$") then {class: "legacy", key: .}
    else {class: "unmanaged", key: .} end;
  map(
    (.name | classify) as $c
    | ((.tag_last_pushed // .last_updated) | ts) as $pushed
    | (.tag_last_pulled | ts) as $pulled
    | {name, digest, class: $c.class}
    + if $c.class == "release" or $c.class == "unmanaged" then {keep: true, reason: "not a CI or branch tag"}
      elif $keepset[$c.key] then {keep: true, reason: "in keep-set as \($c.key)"}
      elif $pushed == null then {keep: true, reason: "no push date"}
      elif ($pushed | ago) < $minage then {keep: true, reason: "pushed \($pushed | ago)d ago"}
      elif $pulled != null and ($pulled | ago) < $grace then {keep: true, reason: "pulled \($pulled | ago)d ago"}
      elif .digest == null then {keep: true, reason: "no digest reported"}
      else {keep: false, reason: "pushed \($pushed | ago)d ago, \(if $pulled == null then "never pulled" else "pulled \($pulled | ago)d ago" end)"}
      end
  )
  | (reduce (.[] | select(.keep and .digest != null)) as $t ({}; .[$t.digest] //= $t.name)) as $kept
  | map(if (.keep | not) and $kept[.digest] then .keep = true | .reason = "shares manifest with kept tag \($kept[.digest])" else . end)
  | .[]
  | [(if .keep then "keep" else "delete" end), .class, "\($org)/\($repo):\(.name)", .reason]
  | @tsv
'

: > "$WORKDIR/decisions.tsv"
for repo in $CLEANUP_REPOS; do
  fetch_tags "$repo" > "$WORKDIR/tags.jsonl"
  [ -s "$WORKDIR/tags.jsonl" ] || die "${DOCKER_ORG}/${repo} lists no tags; refusing to trust the listing"
  jq -rs --rawfile keep "$KEEP_SET" --argjson now "$NOW" --argjson hl "$HASH_LEN" \
    --argjson minage "$CLEANUP_MIN_AGE_DAYS" --argjson grace "$CLEANUP_PULL_GRACE_DAYS" \
    --arg org "$DOCKER_ORG" --arg repo "$repo" "$DECIDE" "$WORKDIR/tags.jsonl" >> "$WORKDIR/decisions.tsv"
done

# If nothing listed is in the keep-set, the keep-set or the listing is wrong,
# and every candidate below is suspect.
grep -q $'\tin keep-set as ' "$WORKDIR/decisions.tsv" || die "no listed tag is in the keep-set; refusing to list candidates"

[ -z "$EXPLAIN" ] || cp "$WORKDIR/decisions.tsv" "$EXPLAIN"
awk -F'\t' '$1 == "delete" { print $3 }' "$WORKDIR/decisions.tsv"
awk -F'\t' '{ n[$1]++ } END { printf "image-tag-cleanup-candidates.sh: %d tags checked, %d candidates\n", n["keep"] + n["delete"], n["delete"] }' "$WORKDIR/decisions.tsv" >&2
