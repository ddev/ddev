#!/usr/bin/env bash
# image-tag-cleanup.sh [--before-release <vX.Y.Z>] [--execute] [<org>/<repo>:<tag>...]
#
# Candidates are the build tags of $DOCKER_ORG on Docker Hub pushed before a
# release, except v* tags, latest, stable, edge, the tags that the version
# file of $GITHUB_REPOSITORY names at a release or prerelease, on main, or at
# an open pull request, and aliases of those hashes. Without tags, it lists
# every candidate, oldest first. With tags, separated by spaces or commas, it
# shows which are candidates, and fails if none is. --execute deletes those,
# 100 at most, and skips the rest.
#
# Flags:
#   --before-release <vX.Y.Z>  the cutoff [the latest release]
#   --execute                  delete; needs DOCKERHUB_USERNAME and DOCKERHUB_TOKEN
# Environment: DOCKER_ORG and GITHUB_REPOSITORY, both required. Needs gh, jq,
# and curl. In GitHub Actions, the report also goes to the step summary.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly HUB_API="https://hub.docker.com"
readonly REFS="refs/image-tag-cleanup"
readonly VERSION_FILE="pkg/versionconstants/versionconstants.go"
readonly STABLE_RE='^v[0-9]+\.[0-9]+\.[0-9]+$'
readonly MAX_DELETE=100
# The order reasons are listed in: candidates, then what main, pull requests,
# and releases use, then release tags, questionable tags, and recent pushes.
readonly RANK_AWK='function rank(why) { return why ~ /^pushed before/ ? 0 : why == "used by main" ? 1 : why == "used by a pull request" ? 2 : why == "used by a release" ? 3 : why == "release tag" ? 4 : why == "questionable" ? 5 : 6 }'
# Releases and prereleases, as releaseTagPattern in pkg/docker/images.go.
readonly RELEASE_RE='^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9.]*)?$'

BEFORE_RELEASE=""
EXECUTE=false
REQUESTED=""
LISTING_PIDS=()

main() {
  parse_args "${@}"
  WORKDIR="$(mktemp -d)"
  trap cleanup EXIT

  start_listing
  fetch_refs
  find_cutoff
  log "Reading the version files"
  {
    keep_main_tags
    keep_pull_request_tags
    keep_release_tags
  } > "${WORKDIR}/keep.tsv"
  log "Waiting for the Docker Hub listing"
  finish_listing > "${WORKDIR}/tags.tsv"
  # If the version files name nothing listed, the listing or the refs are wrong.
  awk -F'\t' 'NR == FNR { used[$1]; next } $2 in used { found = 1; exit } END { exit !found }' "${WORKDIR}/keep.tsv" "${WORKDIR}/tags.tsv" ||
    die "no version file names a listed tag; refusing to delete anything"
  decide "${WORKDIR}/keep.tsv" "${WORKDIR}/tags.tsv" > "${WORKDIR}/decisions.tsv"

  select_tags
  if [[ "${EXECUTE}" == true && "$(count selected)" -gt "${MAX_DELETE}" ]]; then
    die "the tags given are $(count selected) candidates; delete at most ${MAX_DELETE} per run"
  fi
  summarize > "${WORKDIR}/summary"
  cat "${WORKDIR}/summary" >&2
  if [[ "${EXECUTE}" == true ]]; then
    delete_tags
  fi
  print_report
  if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    write_step_summary >> "${GITHUB_STEP_SUMMARY}"
  fi
  [[ ! -s "${WORKDIR}/failed" ]] || die "$(count failed) of $(count selected) deletions failed"
  [[ -z "${REQUESTED}" || -s "${WORKDIR}/selected" ]] || die "none of the tags given is a candidate; nothing to delete"
}

log() {
  echo "${*}" >&2
}

die() {
  log "image-tag-cleanup.sh: ${*}"
  exit 1
}

# A subshell that dies of a shell error can run this trap too.
cleanup() {
  [[ "${BASHPID}" == "${$}" ]] || return 0
  kill "${LISTING_PIDS[@]}" 2>/dev/null || true
  rm -rf "${WORKDIR}"
  git for-each-ref --format='delete %(refname)' "${REFS}/" | git update-ref --stdin
}

parse_args() {
  while [[ "${#}" -gt 0 ]]; do
    case "${1}" in
      -h | --help) sed -n '2,/^$/s/^# \{0,1\}//p' "${BASH_SOURCE[0]}"; exit 0 ;;
      --execute) EXECUTE=true ;;
      --before-release) [[ -n "${2:-}" ]] || die "${1} needs a value"; BEFORE_RELEASE="${2}"; shift ;;
      -*) die "unknown argument '${1}'; see --help" ;;
      *) REQUESTED+="${1} " ;;
    esac
    shift
  done
  REQUESTED="$(tr -s ', \t' '\n' <<< "${REQUESTED}" | sed '/^$/d')"
  [[ -n "${DOCKER_ORG:-}" && -n "${GITHUB_REPOSITORY:-}" ]] || die "DOCKER_ORG and GITHUB_REPOSITORY must be set"
  if [[ "${EXECUTE}" == true ]]; then
    [[ -n "${REQUESTED}" ]] || die "--execute deletes only the tags given; run without it to list them"
  fi
  if [[ "${EXECUTE}" == true && ( -z "${DOCKERHUB_USERNAME:-}" || -z "${DOCKERHUB_TOKEN:-}" ) ]]; then
    die "--execute needs DOCKERHUB_USERNAME and DOCKERHUB_TOKEN set in the environment"
  fi
}

# Fetches main, the release tags, and open pull request heads under ${REFS},
# rather than trusting the checkout's, which can be stale or local-only.
fetch_refs() {
  local remote="${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY}" prs pr
  local refspecs=("+refs/heads/main:${REFS}/main" "+refs/tags/v*:${REFS}/tags/v*")
  prs="$(gh pr list --repo "${GITHUB_REPOSITORY}" --state open --limit 1000 --json number --jq '.[].number')" ||
    die "listing open pull requests of ${GITHUB_REPOSITORY} failed"
  for pr in ${prs}; do
    refspecs+=("+refs/pull/${pr}/head:${REFS}/pr/${pr}")
  done
  log "Fetching main, releases, and $(wc -w <<< "${prs}" | tr -d ' ') open pull requests from ${GITHUB_REPOSITORY}"
  git fetch -q --no-tags "${remote}" "${refspecs[@]}" || die "fetching from ${remote} failed"
}

releases() {
  git for-each-ref --format='%(refname:lstrip=3)' "${REFS}/tags/" | grep -E "${RELEASE_RE}" || true
}

# Sets BEFORE_RELEASE, the latest release unless given, and CUTOFF, its date.
find_cutoff() {
  if [[ -z "${BEFORE_RELEASE}" ]]; then
    BEFORE_RELEASE="$(releases | grep -E "${STABLE_RE}" | sort -V | tail -n 1 || true)"
    [[ -n "${BEFORE_RELEASE}" ]] || die "${GITHUB_REPOSITORY} has no vX.Y.Z release"
  fi
  [[ "${BEFORE_RELEASE}" =~ ${STABLE_RE} ]] || die "--before-release '${BEFORE_RELEASE}' is not a vX.Y.Z release"
  CUTOFF="$(git for-each-ref --format='%(creatordate:unix)' "${REFS}/tags/${BEFORE_RELEASE}")"
  [[ -n "${CUTOFF}" ]] || die "${GITHUB_REPOSITORY} has no release ${BEFORE_RELEASE}"
}

# tags_at <ref> <reason> [release] prints "<tag> <reason>" for each *Tag
# variable in the version file at <ref>, and fails if there are none. With
# "release", the tag a release pulls: <Var>Branch if that is a release tag.
tags_at() {
  git show "${1}:${VERSION_FILE}" 2>/dev/null | RE="${RELEASE_RE}" awk -v OFS='\t' -v reason="${2}" -v release="${3:-}" '
    $1 == "var" && $3 == "=" && $4 ~ /^"/ { gsub(/"/, "", $4); value[$2] = $4; if ($2 ~ /Tag$/) vars[++n] = $2 }
    # Releases before v1.22.0 pulled from the drud organization: nothing to keep.
    END {
      if (value["WebImg"] ~ /^drud\//) exit 0
      for (i = 1; i <= n; i++) {
        branch = value[vars[i] "Branch"]
        print (release && branch ~ ENVIRON["RE"] ? branch : value[vars[i]]), reason
      }
      exit !n
    }'
}

# The keep_* functions print "<tag> <reason>", tab-separated. CI waits for
# the tags main and pull requests name, not the ones they would pull.

# Releases without this version file are older than any tag in ${DOCKER_ORG}.
keep_release_tags() {
  local release ref
  for release in $(releases); do
    ref="${REFS}/tags/${release}"
    git cat-file -e "${ref}:${VERSION_FILE}" 2>/dev/null || continue
    tags_at "${ref}" "used by a release (${release})" release || die "${release} names no image tags in ${VERSION_FILE}"
  done
}

keep_main_tags() {
  tags_at "${REFS}/main" "used by main" || die "main names no image tags in ${VERSION_FILE}"
}

keep_pull_request_tags() {
  local ref
  for ref in $(git for-each-ref --format='%(refname)' "${REFS}/pr/"); do
    tags_at "${ref}" "used by a pull request (#${ref##*/})" || die "pull request #${ref##*/} names no image tags in ${VERSION_FILE}"
  done
}

# Lists every repository in image-configs.sh at once, in the background.
start_listing() {
  local entry repo extra name
  # shellcheck source=containers/image-configs.sh
  source "${SCRIPT_DIR}/image-configs.sh"
  mkdir "${WORKDIR}/tags"
  for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
    IFS='|' read -r repo _ _ _ _ _ _ _ extra <<< "${entry}"
    for name in ${repo} ${extra}; do
      list_tags "${name}" > "${WORKDIR}/tags/${name}" &
      LISTING_PIDS+=("${!}")
    done
  done
  log "Listing ${#LISTING_PIDS[@]} repositories of ${DOCKER_ORG} on Docker Hub"
}

# Prints "<org/repo:tag> <tag> <pushed, epoch> <pushed, date>" for every tag.
finish_listing() {
  local pid file
  for pid in "${LISTING_PIDS[@]}"; do
    wait "${pid}" || exit 1
  done
  LISTING_PIDS=()
  for file in "${WORKDIR}/tags/"*; do
    [[ -s "${file}" ]] || die "${DOCKER_ORG}/${file##*/} lists no tags; refusing to trust the listing"
  done
  cat "${WORKDIR}/tags/"*
}

list_tags() {
  local url="${HUB_API}/v2/namespaces/${DOCKER_ORG}/repositories/${1}/tags?page_size=100" page
  while [[ -n "${url}" ]]; do
    # A `next` pointing elsewhere would let the listing answer for itself.
    [[ "${url}" == "${HUB_API}/"* ]] || die "refusing to follow pagination to ${url}"
    page="$(curl -fsSL --retry 5 --retry-delay 10 "${url}")" || die "listing ${1} failed at ${url}"
    jq -r --arg ref "${DOCKER_ORG}/${1}" '.results[]
      | (.tag_last_pushed // .last_updated // "" | sub("\\.[0-9]+Z$"; "Z")) as $pushed
      | [$ref + ":" + .name, .name, (if $pushed == "" then "" else ($pushed | fromdateiso8601) end), $pushed[0:10]]
      | @tsv' <<< "${page}"
    url="$(jq -r '.next // empty' <<< "${page}")"
  done
}

# decide <keep.tsv> <tags.tsv> prints "<keep|delete> <org/repo:tag> <pushed,
# epoch> <pushed, date> <reason>", the first rule that matches, so "pushed
# after" lists only recent tags that nothing uses. An alias of a hash in use,
# like <branch>-<hash>, is kept for the same reason. A v* tag that names no
# release of ${GITHUB_REPOSITORY} is questionable, and so is one without a
# push date: both are kept for a maintainer to check.
# Pull dates aren't used: a scanner pulls nearly every tag each month.
decide() {
  awk -F'\t' -v OFS='\t' -v cutoff="${CUTOFF}" -v release="${BEFORE_RELEASE}" -v releases="$(releases)" '
    BEGIN { n = split(releases, list, "\n"); for (i = 1; i <= n; i++) is_release[list[i]] }
    NR == FNR { if (!($1 in kept)) kept[$1] = $2; if ($1 ~ /^[0-9a-f]+$/ && length($1) == 10) hashes[$1]; next }
    $2 in is_release || $2 == "latest" || $2 == "stable" || $2 == "edge" { print "keep", $1, $3, $4, "release tag"; next }
    $2 in kept { print "keep", $1, $3, $4, kept[$2]; next }
    { n = split($2, part, "-"); for (i = 1; i <= n; i++) if (part[i] in hashes) { print "keep", $1, $3, $4, kept[part[i]]; next } }
    $2 ~ /^v[0-9]/ { print "keep", $1, $3, $4, "questionable (not a release)"; next }
    $3 != "" && $3 >= cutoff { print "keep", $1, $3, $4, "pushed after " release; next }
    $3 == "" { print "keep", $1, $3, $4, "questionable (no push date)"; next }
    { print "delete", $1, $3, $4, "pushed before " release }
  ' "${1}" "${2}"
}

count() {
  wc -l < "${WORKDIR}/${1}" | tr -d ' '
}

# Writes the candidates, the requested ones if any, to selected as "<pushed,
# epoch> <org/repo:tag> <pushed, date>" lines, oldest first, then by tag and
# image. Requested tags that aren't candidates go to skipped as " <org/repo:tag> <why>".
select_tags() {
  awk -F'\t' -v OFS='\t' -v requested="${REQUESTED}" -v skipped="${WORKDIR}/skipped" '
    BEGIN { n = split(requested, list, "\n"); for (i = 1; i <= n; i++) want[list[i]] }
    n && !($2 in want) { next }
    { seen[$2] }
    $1 == "delete" { image = tag = $2; sub(/:.*/, "", image); sub(/^[^:]*:/, "", tag); print $3, tag, image, $2, $4; next }
    n { print "", $2, $5 > skipped }
    END { for (ref in want) if (!(ref in seen)) print "", ref, "not on Docker Hub" > skipped }
  ' "${WORKDIR}/decisions.tsv" | LC_ALL=C sort -t$'\t' -k1,1n -k2,2 -k3,3 | cut -f1,4,5 > "${WORKDIR}/selected"
  touch "${WORKDIR}/skipped" "${WORKDIR}/deleted" "${WORKDIR}/failed"
}

# Prints "<tags> <decision> <why>" for each reason, in RANK_AWK order.
reasons() {
  awk -F'\t' -v OFS='\t' "${RANK_AWK}"' { why = $5; sub(/ \(.*\)$/, "", why); n[$1 OFS why]++ }
    END { for (k in n) { split(k, f, OFS); print rank(f[2]), n[k], f[1], f[2] } }' "${WORKDIR}/decisions.tsv" |
    LC_ALL=C sort -t$'\t' -k1,1n -k4,4 | cut -f2-
}

summarize() {
  echo "${DOCKER_ORG}: $(count decisions.tsv) tags checked, cutoff ${BEFORE_RELEASE}"
  reasons | awk -F'\t' '{ printf "%7d %s\t%s\n", $1, $2, $3 }'
}

# Prints the candidates in batches to delete, or what the tags given come to.
print_report() {
  local markdown="${1:-}"
  if [[ -z "${REQUESTED}" ]]; then
    group selected | awk -F'\t' -v OFS='\t' -v size="${MAX_DELETE}" '
      { if (used && used + $3 > size) { batch++; used = 0 } used += $3; print "Batch " batch + 1, $0 }' > "${WORKDIR}/rows"
    render "${markdown}" copy
  elif [[ "${EXECUTE}" == true ]]; then
    group deleted | under "Deleted" > "${WORKDIR}/rows"
    render "${markdown}"
    listing "Failed to delete" failed "${markdown}"
  else
    group selected | under "Would delete" > "${WORKDIR}/rows"
    render "${markdown}"
  fi
  listing "Skipped, not candidates" skipped "${markdown}"
}

# Writes the kept tags to rows, under a "Kept, <why>" heading for each reason.
kept_rows() {
  local why
  awk -F'\t' -v OFS='\t' "${RANK_AWK}"' $1 == "keep" { why = $5; sub(/ \(.*\)$/, "", why); print rank(why), why, $3, $2, $4, $5 }' "${WORKDIR}/decisions.tsv" | LC_ALL=C sort -t$'\t' -k1,1n -k2,2 -k3,3n -k4,4 | cut -f2- > "${WORKDIR}/kept"
  cut -f1 "${WORKDIR}/kept" | uniq | while IFS= read -r why; do
    awk -F'\t' -v OFS='\t' -v why="${why}" '$1 == why { print $2, $3, $4, $5 }' "${WORKDIR}/kept" > "${WORKDIR}/kept_one"
    group kept_one | under "Kept, ${why}"
  done > "${WORKDIR}/rows"
}

under() {
  awk -v heading="${1}" '{ print heading "\t" $0 }'
}

# group <file> prints "<pushed> <tag names> <tags> <images> <org/repo:tag ...>
# <used by>" for each build in a "<pushed, epoch> <org/repo:tag> <pushed, date>
# [<why>]" file, oldest first. A hash and its <branch>-<hash> aliases are one
# build. The many ddev-dbserver images are only counted.
group() {
  awk -F'\t' -v OFS='\t' '
    {
      tag = $2; sub(/^[^:]*:/, "", tag)
      repo = $2; sub(/:.*/, "", repo); sub(/^[^\/]*\//, "", repo)
      key = tag
      if (match(tag, /-[0-9a-f]+$/) && RLENGTH == 11) key = substr(tag, RSTART + 1)
      if (!(key in n)) { order[++builds] = key; pushed[key] = $3; if (match($4, /\(.*\)$/)) by[key] = substr($4, RSTART + 1, RLENGTH - 2) }
      n[key]++; refs[key] = refs[key] " " $2
      if (!((key, tag) in seen)) { seen[key, tag]; tags[key] = (tag == key ? tag (tags[key] == "" ? "" : ", " tags[key]) : tags[key] (tags[key] == "" ? "" : ", ") tag) }
      if ((key, repo) in seen) next
      seen[key, repo]
      if (repo ~ /^ddev-dbserver-/) db[key]++; else images[key] = images[key] (images[key] == "" ? "" : ", ") repo
    }
    END {
      for (i = 1; i <= builds; i++) {
        key = order[i]
        if (db[key]) images[key] = images[key] (images[key] == "" ? "" : ", ") db[key] " ddev-dbserver"
        print pushed[key], tags[key], n[key], images[key], substr(refs[key], 2), by[key]
      }
    }' "${WORKDIR}/${1}"
}

# render [markdown] [copy|kept] prints the "<heading>" + group rows in rows as
# a table under each heading. In Markdown, copy adds the heading's tags as one
# line to paste into images_to_delete; copy and kept collapse the table.
# shellcheck disable=SC2016 # the backticks are Markdown
render() {
  awk -F'\t' -v markdown="${1:-}" -v mode="${2:-}" '
    NR == FNR { if (!($1 in from)) from[$1] = $2; to[$1] = $2; tags[$1] += $4; refs[$1] = refs[$1] " " $6; next }
    $1 != heading {
      if (markdown && heading != "") print "\n</details>"
      heading = $1
      title = sprintf("%s: pushed %s to %s", heading, from[heading], to[heading])
      size = sprintf("%d tag%s", tags[heading], (tags[heading] == 1 ? "" : "s"))
      if (!markdown) printf "%s, %s:\n", title, size
      else if (mode == "kept") printf "\n<details><summary>%s: %s</summary>\n\n| Pushed | Tag | Images |\n| --- | --- | --- |\n", heading, size
      else {
        printf "\n### %s\n\n", title
        if (mode == "copy") printf "```text\n%s\n```\n\n", substr(refs[heading], 2)
        printf "<details%s><summary>%s</summary>\n\n| Pushed | Tag | Images |\n| --- | --- | --- |\n", (mode ? "" : " open"), size
      }
    }
    { by = ($7 == "" ? "" : " (" $7 ")") }
    markdown { gsub(/, /, "`, `", $3); printf "| %s | `%s`%s | %s |\n", $2, $3, by, $5; next }
    { printf "  %s  %s%s  %s\n", $2, $3, by, $5 }
    END { if (markdown && heading != "") print "\n</details>" }
  ' "${WORKDIR}/rows" "${WORKDIR}/rows"
}

# listing <title> <file> [markdown] prints "<org/repo:tag>  <why or pushed>"
# for each line of a non-empty file.
# shellcheck disable=SC2016 # the backticks are Markdown
listing() {
  [[ -s "${WORKDIR}/${2}" ]] || return 0
  if [[ -n "${3:-}" ]]; then
    printf '\n### %s (%d)\n\n```text\n%s\n```\n' "${1}" "$(count "${2}")" "$(awk -F'\t' '{ print $2 "  " $3 }' "${WORKDIR}/${2}")"
  else
    echo "${1} ($(count "${2}")):"
    awk -F'\t' '{ print "  " $2 "  " $3 }' "${WORKDIR}/${2}"
  fi
}

# shellcheck disable=SC2016 # the backticks are Markdown
write_step_summary() {
  printf '## Image tag cleanup for %s\n\nCutoff **%s**: %d tags checked, **%d candidates**.\n\n' \
    "${DOCKER_ORG}" "${BEFORE_RELEASE}" "$(count decisions.tsv)" "$(grep -c '^delete' "${WORKDIR}/decisions.tsv" || true)"
  echo "| Decision | Why | Tags |"
  echo "| --- | --- | --- |"
  reasons | awk -F'\t' '{ print "| " $2 " | " $3 " | " $1 " |" }'
  if [[ -n "${REQUESTED}" ]]; then
    printf '\n<details><summary>Tags given: %d</summary>\n\n```text\n%s\n```\n\n</details>\n' "$(wc -l <<< "${REQUESTED}")" "${REQUESTED}"
  fi
  if [[ -z "${REQUESTED}" ]]; then
    kept_rows
    render markdown kept
    printf '\nTo delete a batch, copy its line into `images_to_delete`. Open the batch to see what is in it.\n'
  fi
  print_report markdown
}

delete_tags() {
  local token pushed ref date done=0 total
  [[ -s "${WORKDIR}/selected" ]] || return 0
  total="$(count selected)"
  log "Deleting $(count selected) tags"
  token="$(jq -n --arg u "${DOCKERHUB_USERNAME}" --arg p "${DOCKERHUB_TOKEN}" '{username: $u, password: $p}' |
    curl -fsS -H "Content-Type: application/json" -X POST --data @- "${HUB_API}/v2/users/login/" | jq -r '.token // empty')" ||
    die "Docker Hub login failed"
  [[ -n "${token}" ]] || die "Docker Hub login returned no token"

  while IFS=$'\t' read -r pushed ref date; do
    done=$((done + 1))
    if curl -fsS -o /dev/null -X DELETE -H "Authorization: JWT ${token}" "${HUB_API}/v2/repositories/${ref%%:*}/tags/${ref#*:}/"; then
      printf '%s\t%s\t%s\n' "${pushed}" "${ref}" "${date}" >> "${WORKDIR}/deleted"
      log "  [${done}/${total}] deleted ${ref}"
    else
      printf '%s\t%s\t%s\n' "${pushed}" "${ref}" "${date}" >> "${WORKDIR}/failed"
      log "  [${done}/${total}] failed ${ref}"
    fi
  done < "${WORKDIR}/selected"
}

main "${@}"
