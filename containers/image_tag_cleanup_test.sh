#!/usr/bin/env bash
# image_tag_cleanup_test.sh - unit tests for image-tag-cleanup.sh.
#
# Fetches from a scratch repository standing in for GitHub, with backdated
# commits and release tags. `curl` is stubbed to serve Docker Hub listings
# from fixtures and log every request, `gh` to list open pull requests.
# Run with:
#   containers/image_tag_cleanup_test.sh

set -eu -o pipefail

CLEANUP_SH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/image-tag-cleanup.sh"

FAILURES=0

fail() {
  echo "FAIL: $1" >&2
  FAILURES=$((FAILURES + 1))
}

pass() {
  echo "PASS: $1"
}

assert_eq() {
  local expected="$1" actual="$2" desc="$3"
  if [ "$expected" = "$actual" ]; then
    pass "$desc"
  else
    fail "$desc"$'\n'"--- expected:"$'\n'"$expected"$'\n'"--- got:"$'\n'"$actual"
  fi
}

# Runs the command, expecting failure and a message containing <needle>.
assert_fails_with() {
  local needle="$1" desc="$2" output
  shift 2
  if output="$("$@" 2>&1)"; then
    fail "$desc (succeeded)"
  elif [[ "$output" == *"$needle"* ]]; then
    pass "$desc"
  else
    fail "$desc (failed for the wrong reason: $output)"
  fi
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT
NOW="$(date +%s)"

# ---------------------------------------------------------------------------
# Scratch repository. The default cutoff is the latest release, v1.1.0,
# committed 100 days ago.
# ---------------------------------------------------------------------------

REPO="$WORKDIR/github/ddev/ddev"
mkdir -p "$REPO"
git -C "$REPO" init -q -b main
git -C "$REPO" config user.email test@example.com
git -C "$REPO" config user.name test
git -C "$REPO" config commit.gpgsign false
git -C "$REPO" config tag.gpgsign false

# commit_at <days-ago> <message>: commits whatever is staged, backdated.
commit_at() {
  local date="@$((NOW - $1 * 86400)) +0000"
  GIT_AUTHOR_DATE="$date" GIT_COMMITTER_DATE="$date" git -C "$REPO" commit -q --allow-empty -m "$2"
}

# write_version <WebTag> <BaseDBTag> [<WebTagBranch>] [<WebImg>]
write_version() {
  local f="$REPO/pkg/versionconstants/versionconstants.go"
  mkdir -p "$(dirname "$f")"
  printf 'package x\n\nvar WebImg = "%s"\nvar WebTag = "%s" // comment\nvar BaseDBTag = "%s"\n' "${4:-ddev/ddev-webserver}" "$1" "$2" > "$f"
  [ -z "${3:-}" ] || printf 'var WebTagBranch = "%s"\nvar BaseDBTagBranch = "some_branch"\n' "$3" >> "$f"
  git -C "$REPO" add -A
}

commit_at 500 "before any version file"
git -C "$REPO" tag v0.0.1

write_version 20190101_drud_web v0.9.0 "" drud/ddev-webserver
commit_at 450 "release that pulled from the drud organization"
git -C "$REPO" tag v0.9.0

write_version 5555555555 5555555556 v0.9.5
commit_at 350 "release that pulls its release tag for web"
git -C "$REPO" tag v0.9.5

write_version 0000000001 0000000002
commit_at 300 "test tag"
git -C "$REPO" tag v0.9.6-aaa1

write_version 1111111111 v1.0.0
commit_at 200 "previous release"
git -C "$REPO" tag v1.0.0

write_version 2222222222 2222222223
commit_at 150 "replaced before the release"

write_version 8888888888 8888888889
commit_at 120 "prerelease"
git -C "$REPO" tag v1.1.0-rc1

write_version 7777777777 7777777778 v1.1.0
commit_at 100 "latest release"
git -C "$REPO" tag v1.1.0

echo readme > "$REPO/README"
git -C "$REPO" add README
commit_at 5 "main tip, version file untouched"

git -C "$REPO" checkout -q -b pr
write_version 4444444444 7777777778
commit_at 1 "open pull request #1"
git -C "$REPO" update-ref refs/pull/1/head pr
git -C "$REPO" checkout -q main

git clone -q --no-tags "$REPO" "$WORKDIR/github/ddev/notags"

git init -q "$WORKDIR/checkout"
cd "$WORKDIR/checkout"
export GITHUB_SERVER_URL="$WORKDIR/github" GITHUB_REPOSITORY=ddev/ddev DOCKER_ORG=ddev

# ---------------------------------------------------------------------------
# Stubs
# ---------------------------------------------------------------------------

BINDIR="$WORKDIR/bin"
mkdir -p "$BINDIR"
export CURL_FIXTURES="$WORKDIR/fixtures" CURL_LOG="$WORKDIR/curl.log" CURL_FAIL_URLS="$WORKDIR/curl_fail_urls"
cat > "$BINDIR/curl" <<'EOF'
#!/usr/bin/env bash
set -eu -o pipefail
method=GET url="" auth=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -X) method="$2"; shift 2 ;;
    -H) [[ "$2" == Authorization:* ]] && auth="$2"; shift 2 ;;
    -o|--data|--retry|--retry-delay) shift 2 ;;
    http*) url="$1"; shift ;;
    *) shift ;;
  esac
done
echo "$method $url${auth:+ $auth}" >> "$CURL_LOG"
grep -qxF "$url" "$CURL_FAIL_URLS" 2>/dev/null && exit 22
case "$method" in
  POST) cat > /dev/null; echo '{"token":"stub-token"}' ;;
  DELETE) ;;
  GET)
    # Repositories without a fixture list only `latest`.
    f="$CURL_FIXTURES/$(printf '%s' "$url" | sed 's/[^A-Za-z0-9]/_/g')"
    if [ -f "$f" ]; then cat "$f"; else echo '{"next":null,"results":[{"name":"latest","tag_last_pushed":"2020-01-01T00:00:00Z"}]}'; fi
    ;;
esac
EOF
# GH_PRS lists the open pull request numbers; GH_FAIL makes the listing fail.
cat > "$BINDIR/gh" <<'EOF'
#!/usr/bin/env bash
[ -z "${GH_FAIL:-}" ] || exit 1
[ -z "${GH_PRS:-}" ] || printf '%s\n' $GH_PRS
EOF
chmod +x "$BINDIR/curl" "$BINDIR/gh"
export PATH="$BINDIR:$PATH" GH_PRS=1

HUB="https://hub.docker.com/v2/namespaces/ddev/repositories"

# tag <name> <pushed-days-ago>
tag() {
  local pushed
  pushed="$(jq -rn --argjson t "$((NOW - $2 * 86400))" '$t | todate | sub("Z$"; ".123456789Z")')"
  printf '{"name":"%s","last_updated":"%s","tag_last_pushed":"%s"}\n' "$1" "$pushed" "$pushed"
}

page_url() {
  if [ "$2" = 1 ]; then echo "${HUB}/$1/tags?page_size=100"; else echo "${HUB}/$1/tags?page=$2&page_size=100"; fi
}

# serve <repo> <page> <next-url or -> reads tag lines from stdin.
serve() {
  local next=null
  [ "$3" = - ] || next="\"$3\""
  jq -s --argjson next "$next" '{count: length, next: $next, results: .}' \
    > "$CURL_FIXTURES/$(page_url "$1" "$2" | sed 's/[^A-Za-z0-9]/_/g')"
}

reset_stub() {
  rm -rf "$CURL_FIXTURES"
  mkdir -p "$CURL_FIXTURES"
  : > "$CURL_LOG"
  : > "$CURL_FAIL_URLS"
}

serve_fixture() {
  reset_stub
  {
    tag v1.1.0 400                     # kept: release tag
    tag latest 400                     # kept: release tag
    tag v1.2.0 400                     # kept: questionable, no such release
    tag v0.9.6-aaa1 400                # kept: questionable, a test tag isn't a release
    echo '{"name":"nodate"}'           # kept: questionable, no push date
    tag 20190101_drud_web 400          # goes: v0.9.0 pulled from drud
    tag 5555555555 400                 # goes: v0.9.5 pulls its WebTagBranch
    tag 5555555556 400                 # kept: v0.9.5 pulls it
    tag 1111111111 400                 # kept: v1.0.0 pulls it
    tag 2222222222 400                 # goes: main no longer names it
    tag 7777777777 400                 # kept: main names it
    tag 8888888888 400                 # kept: prerelease v1.1.0-rc1 pulls it
    tag 4444444444 400                 # kept: open pull request #1 names it
    tag main-2222222222 400            # goes: an alias of a hash nothing uses
    tag 0000000001 400                 # goes: test tag v0.9.6-aaa1 isn't a release
    tag 0a0a0a0a0a 150                 # goes: pushed before v1.1.0
    tag cccccccccc 10                  # kept: pushed after v1.1.0
    tag 7777777778 10                  # kept: main names it, and so does v1.1.0
    tag 20240101_web-7777777777 400    # kept: an alias of a hash main uses
    tag 20240101_old_branch 400        # goes, with ddev-webserver-prod's
    tag 20240101_old_branch-amd64 400  # goes
  } | serve ddev-webserver 1 "$(page_url ddev-webserver 2)"
  tag junk 450 | serve ddev-webserver 2 -  # goes first: the oldest
  { tag latest 400; tag 20240101_old_branch 400; } | serve ddev-webserver-prod 1 -
}

# Serves 60 tags in each of two images, and one tag main names.
serve_many() {
  reset_stub
  { tag 7777777777 400; seq -f 't%02g' 60 | while read -r t; do tag "$t" 400; done; } | serve ddev-webserver 1 -
  seq -f 't%02g' 60 | while read -r t; do tag "$t" 400; done | serve ddev-webserver-prod 1 -
}

log_count() {
  grep -c "^$1 " "$CURL_LOG" || true
}

# ---------------------------------------------------------------------------
# Decisions
# ---------------------------------------------------------------------------

# Candidates as "<tags> <images>", oldest first, then by tag; a hash and its
# aliases share a line.
CANDIDATES="junk ddev-webserver
0000000001 ddev-webserver
20190101_drud_web ddev-webserver
20240101_old_branch ddev-webserver, ddev-webserver-prod
20240101_old_branch-amd64 ddev-webserver
2222222222, main-2222222222 ddev-webserver
5555555555 ddev-webserver
0a0a0a0a0a ddev-webserver"

# listed <heading> [<arg>...] prints "<tag> <images>" for the rows a run
# lists under the headings that start with <heading>.
listed() {
  "$CLEANUP_SH" "${@:2}" 2>/dev/null | awk -v heading="$1" '/^[^ ]/ { on = index($0, heading) == 1; next } on { $1 = ""; print substr($0, 2) }'
}

serve_fixture
assert_eq "$CANDIDATES" "$(listed "Batch")" "a dry run lists every candidate by tag, oldest first, across pages and images"
assert_eq "Batch 1: pushed $(date -u -d "@$((NOW - 450 * 86400))" +%F) to $(date -u -d "@$((NOW - 150 * 86400))" +%F), 10 tags:"$'\n'"  $(date -u -d "@$((NOW - 450 * 86400))" +%F)  junk  ddev-webserver" \
  "$("$CLEANUP_SH" 2>/dev/null | head -2)" "a dry run shows the batch, and when each tag was pushed"
assert_eq "0 0" "$(log_count POST) $(log_count DELETE)" "a dry run neither logs in nor deletes"
assert_eq "" "$(git for-each-ref refs/image-tag-cleanup/)" "the fetched refs are removed after the run"
summary="$("$CLEANUP_SH" 2>&1 >/dev/null)"
for line in "cutoff v1.1.0" $'keep\tquestionable' $'keep\tused by a release' $'keep\tused by main' $'keep\tused by a pull request' $'delete\tpushed before v1.1.0'; do
  if [[ "$summary" == *"$line"* ]]; then pass "the summary says '$line'"; else fail "the summary should say '$line': $summary"; fi
done
assert_eq "$(sed '/^2222222222, /a 4444444444 ddev-webserver' <<< "$CANDIDATES")" \
  "$(GH_PRS="" listed "Batch")" "a closed pull request keeps nothing"

serve_fixture
assert_eq "$(grep -v -e '^0a0a0a0a0a ' <<< "$CANDIDATES")" \
  "$(listed "Batch" --before-release v1.0.0)" "--before-release moves the cutoff"
assert_fails_with "is not a vX.Y.Z release" "--before-release refuses a test tag" "$CLEANUP_SH" --before-release v0.9.6-aaa1
assert_fails_with "has no release v9.9.9" "--before-release refuses a missing release" "$CLEANUP_SH" --before-release v9.9.9
assert_fails_with "ddev/notags has no vX.Y.Z release" "the default cutoff needs a release" env GITHUB_REPOSITORY=ddev/notags GH_PRS= "$CLEANUP_SH"

serve_many
assert_eq "Batch 1, 100 tags:"$'\n'"Batch 2, 20 tags:" \
  "$("$CLEANUP_SH" 2>/dev/null | sed -n 's/: pushed [0-9-]* to [0-9-]*//p')" "batches hold 100 tags at most, and never split a tag"

serve_fixture
export GITHUB_STEP_SUMMARY="$WORKDIR/step-summary.md"
"$CLEANUP_SH" >/dev/null 2>&1
unset GITHUB_STEP_SUMMARY
for line in "| delete | pushed before v1.1.0 | 10 |" "<details><summary>10 tags</summary>" \
  "| $(date -u -d "@$((NOW - 450 * 86400))" +%F) | \`junk\` | ddev-webserver |" \
  "<details><summary>Kept, used by main: 3 tags</summary>" \
  "<details><summary>Kept, pushed after v1.1.0: 1 tag</summary>" \
  "<details><summary>Kept, questionable: 3 tags</summary>" \
  "| $(date -u -d "@$((NOW - 400 * 86400))" +%F) | \`v1.2.0\` (not a release) | ddev-webserver |" \
  "| $(date -u -d "@$((NOW - 400 * 86400))" +%F) | \`7777777777\`, \`20240101_web-7777777777\` | ddev-webserver |" \
  "| $(date -u -d "@$((NOW - 400 * 86400))" +%F) | \`5555555556\` (v0.9.5) | ddev-webserver |" \
  "| $(date -u -d "@$((NOW - 400 * 86400))" +%F) | \`4444444444\` (#1) | ddev-webserver |"; do
  if grep -qxF "$line" "$WORKDIR/step-summary.md"; then pass "the step summary has '$line'"; else fail "the step summary should have '$line': $(cat "$WORKDIR/step-summary.md")"; fi
done
batch="$(grep '^ddev/' "$WORKDIR/step-summary.md")"
assert_eq "$CANDIDATES" "$(listed "Would delete" "$batch")" "a batch's line, pasted back, would delete that batch"

# ---------------------------------------------------------------------------
# Given tags
# ---------------------------------------------------------------------------

REQUESTED=("ddev/ddev-webserver:5555555555, ddev/ddev-webserver:junk" ddev/ddev-webserver:4444444444 ddev/ddev-webserver:gone)
serve_fixture
assert_eq "$(printf '%s ddev-webserver\n' junk 5555555555)" "$(listed "Would delete" "${REQUESTED[@]}")" \
  "given tags that are candidates are shown, oldest first"
assert_eq "$(printf '  ddev/ddev-webserver:%s  %s\n' 4444444444 "used by a pull request (#1)" gone "not on Docker Hub")" \
  "$("$CLEANUP_SH" "${REQUESTED[@]}" 2>/dev/null | sed -n '/^Skipped/,$p' | tail -n +2)" "given tags that aren't candidates are skipped, with why"
assert_eq "0" "$(log_count DELETE)" "given tags without --execute is a dry run"
GITHUB_STEP_SUMMARY="$WORKDIR/given.md" "$CLEANUP_SH" "${REQUESTED[@]}" >/dev/null 2>&1
assert_eq "$(printf '<details><summary>Tags given: 4</summary>\n\n```text\n'; printf 'ddev/ddev-webserver:%s\n' 5555555555 junk 4444444444 gone; echo '```')" \
  "$(sed -n '/^<details><summary>Tags given/,/^```$/p' "$WORKDIR/given.md")" "the step summary lists the tags given"

serve_fixture
assert_fails_with "--execute deletes only the tags given" "--execute needs the tags" \
  env DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$CLEANUP_SH" --execute
assert_fails_with "needs DOCKERHUB_USERNAME and DOCKERHUB_TOKEN" "--execute needs both credentials" \
  env -u DOCKERHUB_USERNAME -u DOCKERHUB_TOKEN "$CLEANUP_SH" --execute "${REQUESTED[@]}"
assert_eq "0" "$(wc -l < "$CURL_LOG" | tr -d ' ')" "a refused --execute makes no request"

serve_many
assert_fails_with "the tags given are 120 candidates; delete at most 100 per run" "--execute deletes 100 at most" \
  env DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$CLEANUP_SH" --execute "$(seq -f 'ddev/ddev-webserver:t%02g' 60) $(seq -f 'ddev/ddev-webserver-prod:t%02g' 60)"
assert_eq "0" "$(log_count DELETE)" "too many tags delete nothing"

NOT_CANDIDATES="ddev/ddev-webserver:v1.1.0 ddev/ddev-webserver:4444444444 ddev/ddev-webserver:gone"
serve_fixture
assert_eq "  ddevhq/ddev-webserver:junk  not in ddev" \
  "$("$CLEANUP_SH" ddevhq/ddev-webserver:junk ddev/ddev-webserver:junk 2>/dev/null | sed -n '/^Skipped/,$p' | tail -n +2)" \
  "a tag of another organization is skipped as not in DOCKER_ORG"
assert_fails_with "none of the tags given is a candidate" "a dry run of tags that aren't candidates fails" "$CLEANUP_SH" "$NOT_CANDIDATES"
assert_fails_with "none of the tags given is a candidate" "--execute of tags that aren't candidates fails" \
  env DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$CLEANUP_SH" --execute "$NOT_CANDIDATES"
assert_eq "00" "$(log_count POST)$(log_count DELETE)" "--execute with no candidates doesn't log in or delete"

serve_fixture
assert_eq "$(printf '  ddev/ddev-webserver:%s  %s\n' v1.1.0 "release tag" 4444444444 "used by a pull request (#1)" gone "not on Docker Hub")" \
  "$(DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$CLEANUP_SH" --execute "${REQUESTED[@]}" ddev/ddev-webserver:v1.1.0 2>/dev/null | sed -n '/^Skipped/,$p' | tail -n +2)" \
  "--execute skips the tags that aren't candidates, with why"
assert_eq "2" "$(log_count DELETE)" "--execute deletes the candidates among the tags given"

CANDIDATE_TAGS="ddev/ddev-webserver:5555555555, ddev/ddev-webserver:junk"
serve_fixture
assert_eq "$(printf '%s ddev-webserver\n' junk 5555555555)" \
  "$(DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t listed "Deleted" --execute "$CANDIDATE_TAGS")" "--execute reports what it deleted"
assert_eq "$(printf '%s\n' \
  "DELETE https://hub.docker.com/v2/repositories/ddev/ddev-webserver/tags/junk/ Authorization: JWT stub-token" \
  "DELETE https://hub.docker.com/v2/repositories/ddev/ddev-webserver/tags/5555555555/ Authorization: JWT stub-token")" \
  "$(grep '^DELETE ' "$CURL_LOG")" "--execute deletes only the given candidates, with the login token"
serve_fixture
assert_eq "$(printf '  [1/2] deleted ddev/ddev-webserver:junk\n  [2/2] deleted ddev/ddev-webserver:5555555555')" \
  "$(DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$CLEANUP_SH" --execute "$CANDIDATE_TAGS" 2>&1 >/dev/null | grep '^  \[')" "--execute logs each deletion as it goes"

serve_fixture
echo "https://hub.docker.com/v2/repositories/ddev/ddev-webserver/tags/junk/" > "$CURL_FAIL_URLS"
assert_fails_with "1 of 2 deletions failed" "a failed deletion fails the run" \
  env DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$CLEANUP_SH" --execute "$CANDIDATE_TAGS"
assert_eq "2" "$(log_count DELETE)" "a failed deletion doesn't stop the rest"
serve_fixture
echo "https://hub.docker.com/v2/repositories/ddev/ddev-webserver/tags/junk/" > "$CURL_FAIL_URLS"
assert_eq "  [1/2] failed ddev/ddev-webserver:junk" \
  "$(DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$CLEANUP_SH" --execute "$CANDIDATE_TAGS" 2>&1 >/dev/null | grep -m1 '^  \[')" "a failed deletion is logged as it happens"

# ---------------------------------------------------------------------------
# Refusals
# ---------------------------------------------------------------------------

serve_fixture
page_url ddev-webserver 2 > "$CURL_FAIL_URLS"
assert_fails_with "listing ddev-webserver failed" "a failed page aborts" "$CLEANUP_SH"
if [ -n "$("$CLEANUP_SH" 2>/dev/null)" ]; then fail "a failed page printed candidates"; else pass "a failed page prints no candidates"; fi

reset_stub
: | serve ddev-webserver 1 -
assert_fails_with "lists no tags" "an empty listing aborts" "$CLEANUP_SH"

reset_stub
tag junk 400 | serve ddev-webserver 1 "https://evil.example.com/tags?page=2"
assert_fails_with "refusing to follow pagination" "pagination off Docker Hub aborts" "$CLEANUP_SH"

reset_stub
tag junk 400 | serve ddev-webserver 1 -
assert_fails_with "no version file names a listed tag" "a listing nothing names aborts" "$CLEANUP_SH"

serve_fixture
assert_fails_with "listing open pull requests" "a failed pull request listing aborts" env GH_FAIL=1 "$CLEANUP_SH"
assert_fails_with "fetching from $WORKDIR/github/ddev/nosuch failed" "a failed fetch aborts" env GITHUB_REPOSITORY=ddev/nosuch "$CLEANUP_SH"
assert_fails_with "DOCKER_ORG and GITHUB_REPOSITORY must be set" "DOCKER_ORG is required" env -u DOCKER_ORG "$CLEANUP_SH"
assert_fails_with "DOCKER_ORG and GITHUB_REPOSITORY must be set" "GITHUB_REPOSITORY is required" env -u GITHUB_REPOSITORY "$CLEANUP_SH"
assert_fails_with "unknown argument" "an unknown flag is refused" "$CLEANUP_SH" --bogus x
assert_fails_with "--before-release needs a value" "a flag without its value is refused" "$CLEANUP_SH" --before-release

printf 'package x\n' > "$REPO/pkg/versionconstants/versionconstants.go"
git -C "$REPO" add -A
commit_at 0 "version file without tags"
assert_fails_with "names no image tags" "a version file naming no tags aborts" "$CLEANUP_SH"

if [ "$FAILURES" -gt 0 ]; then
  echo "${FAILURES} test(s) failed" >&2
  exit 1
fi
echo "All tests passed"
