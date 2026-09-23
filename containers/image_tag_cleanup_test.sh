#!/usr/bin/env bash
# image_tag_cleanup_test.sh - unit tests for image-tag-keep-set.sh,
# image-tag-cleanup-candidates.sh, and delete-image-tags.sh.
#
# The keep-set runs against a scratch git repository with backdated commits;
# the other two run against a stubbed `curl` that serves Docker Hub listings
# from fixtures and logs every request, so "deletes nothing" is asserted.
# Run with:
#   containers/image_tag_cleanup_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KEEP_SET_SH="$SCRIPT_DIR/image-tag-keep-set.sh"
CANDIDATES_SH="$SCRIPT_DIR/image-tag-cleanup-candidates.sh"
DELETE_SH="$SCRIPT_DIR/delete-image-tags.sh"

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

NOW_TS="$(date +%s)"

# ---------------------------------------------------------------------------
# image-tag-keep-set.sh
# ---------------------------------------------------------------------------

REPO="$WORKDIR/repo"
mkdir -p "$REPO"
git -C "$REPO" init -q -b main
git -C "$REPO" config user.email test@example.com
git -C "$REPO" config user.name test
git -C "$REPO" config commit.gpgsign false
git -C "$REPO" config tag.gpgsign false

# commit_at <days-ago> <message>: commits whatever is staged, backdated.
commit_at() {
  local date="@$((NOW_TS - $1 * 86400)) +0000"
  GIT_AUTHOR_DATE="$date" GIT_COMMITTER_DATE="$date" git -C "$REPO" commit -q --allow-empty -m "$2"
}

# write_version <path> <WebTag> <BaseDBTag>
write_version() {
  mkdir -p "$REPO/$(dirname "$1")"
  printf 'package x\n\nvar WebTag = "%s" // comment\nvar BaseDBTag = "%s"\nvar WebImg = "not-a-tag"\n' "$2" "$3" > "$REPO/$1"
  git -C "$REPO" add -A
}

commit_at 500 "before any version file"
git -C "$REPO" tag v0.1

write_version pkg/version/version.go 20190101_legacy_web v0.9.0
commit_at 400 "old layout release"
git -C "$REPO" tag v0.9.0

git -C "$REPO" rm -q -r pkg/version
write_version pkg/versionconstants/versionconstants.go 0000000001 0000000002
commit_at 300 "superseded before the window opens"

write_version pkg/versionconstants/versionconstants.go 1111111111 v1.0.0
commit_at 200 "release"
git -C "$REPO" tag v1.0.0

write_version pkg/versionconstants/versionconstants.go 2222222222 2222222223
commit_at 150 "in effect when the window opens"

write_version pkg/versionconstants/versionconstants.go 3333333333 3333333334
commit_at 30 "inside the window"

echo readme > "$REPO/README"
git -C "$REPO" add README
commit_at 5 "main tip, version file untouched"

git -C "$REPO" checkout -q -b pr
write_version pkg/versionconstants/versionconstants.go 4444444444 3333333334
commit_at 1 "open pull request"
git -C "$REPO" checkout -q main

keep_set() {
  (cd "$REPO" && KEEP_MAIN_REF=main "$KEEP_SET_SH" "$@")
}

EXPECTED_KEEP="$(printf '%s\n' 1111111111 20190101_legacy_web 2222222222 2222222223 3333333333 3333333334 v0.9.0 v1.0.0 | sort)"
assert_eq "$EXPECTED_KEEP" "$(keep_set 2>/dev/null)" "keep-set holds release tags, both version-file layouts, and main's window"
assert_eq "$(printf '%s\n' "$EXPECTED_KEEP" 4444444444 | sort)" "$(keep_set pr 2>/dev/null)" "keep-set adds the tags of each ref given"
if (cd "$REPO" && KEEP_MAIN_REF=main KEEP_MAIN_DAYS=350 "$KEEP_SET_SH" 2>/dev/null) | grep -qx 0000000001; then
  pass "KEEP_MAIN_DAYS widens main's window"
else
  fail "KEEP_MAIN_DAYS=350 should keep a tag committed 300 days ago"
fi

assert_fails_with "not found" "keep-set fails on a missing main ref" \
  bash -c "cd '$REPO' && KEEP_MAIN_REF=nosuch '$KEEP_SET_SH'"
assert_fails_with "not found" "keep-set fails on a missing extra ref" \
  bash -c "cd '$REPO' && KEEP_MAIN_REF=main '$KEEP_SET_SH' nosuch"

NOTAGS="$WORKDIR/notags"
git clone -q --no-tags "$REPO" "$NOTAGS"
assert_fails_with "were tags fetched" "keep-set fails when no release tags are present" \
  bash -c "cd '$NOTAGS' && KEEP_MAIN_REF=origin/main '$KEEP_SET_SH'"

printf 'package x\n' > "$REPO/pkg/versionconstants/versionconstants.go"
git -C "$REPO" add -A
commit_at 0 "version file without tags"
assert_fails_with "names no image tags" "keep-set fails on a version file naming no tags" \
  bash -c "cd '$REPO' && KEEP_MAIN_REF=main '$KEEP_SET_SH'"

# ---------------------------------------------------------------------------
# curl stub, shared by the candidates and delete tests
# ---------------------------------------------------------------------------

BINDIR="$WORKDIR/bin"
mkdir -p "$BINDIR"
export CURL_FIXTURES="$WORKDIR/fixtures"
export CURL_LOG="$WORKDIR/curl.log"
export CURL_FAIL_URLS="$WORKDIR/curl_fail_urls"
cat > "$BINDIR/curl" <<'CURLEOF'
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
    f="$CURL_FIXTURES/$(printf '%s' "$url" | sed 's/[^A-Za-z0-9]/_/g')"
    [ -f "$f" ] || exit 22
    cat "$f"
    ;;
esac
CURLEOF
chmod +x "$BINDIR/curl"
export PATH="$BINDIR:$PATH"

reset_stub() {
  rm -rf "$CURL_FIXTURES"
  mkdir -p "$CURL_FIXTURES"
  : > "$CURL_LOG"
  : > "$CURL_FAIL_URLS"
}

export NOW="$NOW_TS"
export DOCKER_ORG=ddev
HUB="https://hub.docker.com/v2/namespaces/ddev/repositories"

iso_days_ago() {
  jq -rn --argjson t "$((NOW - $1 * 86400))" '$t | todate | sub("Z$"; ".123456789Z")'
}

# tag <name> <pushed-days-ago> <pulled-days-ago or -> <digest or ->
tag() {
  local pulled=null digest=null pushed
  pushed="$(iso_days_ago "$2")"
  [ "$3" = - ] || pulled="\"$(iso_days_ago "$3")\""
  [ "$4" = - ] || digest="\"sha256:$4\""
  printf '{"name":"%s","digest":%s,"last_updated":"%s","tag_last_pushed":"%s","tag_last_pulled":%s,"images":[]}\n' \
    "$1" "$digest" "$pushed" "$pushed" "$pulled"
}

page_url() {
  if [ "$2" = 1 ]; then
    echo "${HUB}/$1/tags?page_size=100"
  else
    echo "${HUB}/$1/tags?page=$2&page_size=100"
  fi
}

# serve <repo> <page> <next-url or -> reads tag lines from stdin.
serve() {
  local next=null
  [ "$3" = - ] || next="\"$3\""
  jq -s --argjson next "$next" '{count: length, next: $next, results: .}' \
    > "$CURL_FIXTURES/$(page_url "$1" "$2" | sed 's/[^A-Za-z0-9]/_/g')"
}

KEEP="$WORKDIR/keep.txt"
printf '%s\n' aaaaaaaaaa 20250612_released_branch > "$KEEP"

serve_standard_fixture() {
  reset_stub
  {
    tag aaaaaaaaaa 400 - d1
    tag main-aaaaaaaaaa 400 - d1
    tag 20240202_same_manifest_as_kept 400 - d1
    tag bbbbbbbbbb 400 - d2
    tag feature-bbbbbbbbbb 400 - d2
    tag cccccccccc 10 - d3
    tag dddddddddd 400 5 d4
    tag feature-dddddddddd 400 5 d4
    tag eeeeeeeeee 400 100 d5
    tag ffffffffff 400 - d6
    tag v1.24.0 400 - d6
    tag latest 400 - d6
    tag 20250612_released_branch 400 - d7
    tag 20240101_old_branch 400 - d8
    tag 20240101_old_branch-amd64 400 - d9
    tag aaaaaaaaaa-arm64 400 - d10
    tag 1111111111 400 - -
    tag weird_tag 400 - d11
    tag v1.2.3-2222222222 400 - d12
    tag 5 400 - d14
  } | serve ddev-webserver 1 "$(page_url ddev-webserver 2)"
  tag 3333333333 400 - d13 | serve ddev-webserver 2 -
}

# ---------------------------------------------------------------------------
# image-tag-cleanup-candidates.sh
# ---------------------------------------------------------------------------

export CLEANUP_REPOS=ddev-webserver

serve_standard_fixture
EXPECTED_CANDIDATES="$(printf 'ddev/ddev-webserver:%s\n' 20240101_old_branch 20240101_old_branch-amd64 3333333333 bbbbbbbbbb eeeeeeeeee feature-bbbbbbbbbb | sort)"
assert_eq "$EXPECTED_CANDIDATES" "$("$CANDIDATES_SH" --explain "$WORKDIR/explain.tsv" "$KEEP" 2>/dev/null | sort)" \
  "candidates are exactly the old, unpulled, unreferenced CI and branch tags, across pages"

reason() {
  awk -F'\t' -v ref="ddev/ddev-webserver:$1" '$3 == ref { print $1 ": " $4 }' "$WORKDIR/explain.tsv"
}
assert_eq "keep: in keep-set as aaaaaaaaaa" "$(reason main-aaaaaaaaaa)" "an alias is kept when its hash is in the keep-set"
assert_eq "keep: in keep-set as aaaaaaaaaa" "$(reason aaaaaaaaaa-arm64)" "an arch tag is kept when its base is in the keep-set"
assert_eq "keep: shares manifest with kept tag v1.24.0" "$(reason ffffffffff)" "a hash tag is kept when a release tag names the same manifest"
assert_eq "keep: shares manifest with kept tag aaaaaaaaaa" "$(reason 20240202_same_manifest_as_kept)" "a legacy tag is kept when a kept tag names the same manifest"
assert_eq "keep: pushed 10d ago" "$(reason cccccccccc)" "a recently pushed tag is kept"
assert_eq "keep: pulled 5d ago" "$(reason feature-dddddddddd)" "a recently pulled tag is kept"
assert_eq "keep: no digest reported" "$(reason 1111111111)" "a tag without a digest is kept"
assert_eq "keep: not a CI or branch tag" "$(reason v1.2.3-2222222222)" "a release-shaped alias prefix is kept"
assert_eq "keep: not a CI or branch tag" "$(reason 5)" "an unrecognized tag shape is kept"
assert_eq "delete: pushed 400d ago, pulled 100d ago" "$(reason eeeeeeeeee)" "a candidate's reason gives its push and pull age"

assert_eq "$(printf 'ddev/ddev-webserver:%s\n' 20240101_old_branch 20240101_old_branch-amd64 3333333333 bbbbbbbbbb cccccccccc feature-bbbbbbbbbb | sort)" \
  "$(CLEANUP_PULL_GRACE_DAYS=200 CLEANUP_MIN_AGE_DAYS=5 "$CANDIDATES_SH" "$KEEP" 2>/dev/null | sort)" \
  "the push and pull thresholds are configurable"

serve_standard_fixture
page_url ddev-webserver 2 > "$CURL_FAIL_URLS"
assert_fails_with "listing ddev-webserver failed" "a failed page aborts instead of listing a partial result" \
  "$CANDIDATES_SH" "$KEEP"
if [ -n "$("$CANDIDATES_SH" "$KEEP" 2>/dev/null)" ]; then
  fail "a failed page printed candidates"
else
  pass "a failed page prints no candidates"
fi

reset_stub
: | serve ddev-webserver 1 -
assert_fails_with "lists no tags" "an empty listing aborts" "$CANDIDATES_SH" "$KEEP"

reset_stub
tag bbbbbbbbbb 400 - d2 | serve ddev-webserver 1 "https://evil.example.com/tags?page=2"
assert_fails_with "refusing to follow pagination" "pagination off Docker Hub aborts" "$CANDIDATES_SH" "$KEEP"

serve_standard_fixture
echo 9999999999 > "$WORKDIR/unrelated-keep.txt"
assert_fails_with "no listed tag is in the keep-set" "a keep-set matching nothing listed aborts" \
  "$CANDIDATES_SH" "$WORKDIR/unrelated-keep.txt"

: > "$WORKDIR/empty-keep.txt"
assert_fails_with "missing or empty" "an empty keep-set aborts" "$CANDIDATES_SH" "$WORKDIR/empty-keep.txt"

unset CLEANUP_REPOS

# ---------------------------------------------------------------------------
# delete-image-tags.sh
# ---------------------------------------------------------------------------

log_count() {
  grep -c "^$1 " "$CURL_LOG" || true
}

serve_standard_fixture
output="$("$DELETE_SH" "$KEEP" ddev/ddev-webserver:bbbbbbbbbb,ddev/ddev-webserver:eeeeeeeeee 2>/dev/null)"
assert_eq "$(printf 'would delete ddev/ddev-webserver:%s\n' bbbbbbbbbb eeeeeeeeee)" "$output" "a dry run lists what it would delete"
assert_eq "0 0" "$(log_count POST) $(log_count DELETE)" "a dry run neither logs in nor deletes"

serve_standard_fixture
assert_fails_with "ddev/ddev-webserver:aaaaaaaaaa" "a kept tag in the list is refused by name" \
  env DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$DELETE_SH" --execute "$KEEP" ddev/ddev-webserver:bbbbbbbbbb ddev/ddev-webserver:aaaaaaaaaa
assert_eq "0 0" "$(log_count POST) $(log_count DELETE)" "one refused tag means nothing is deleted"

serve_standard_fixture
assert_fails_with "ddev/ddev-webserver:0123456789" "a tag that isn't listed at all is refused" \
  "$DELETE_SH" "$KEEP" ddev/ddev-webserver:0123456789

reset_stub
assert_fails_with "not in a repository this project publishes" "another organization is refused" \
  "$DELETE_SH" "$KEEP" someone/ddev-webserver:bbbbbbbbbb
assert_fails_with "not in a repository this project publishes" "an unknown repository is refused" \
  "$DELETE_SH" "$KEEP" ddev/not-ours:bbbbbbbbbb
assert_fails_with "is not <org>/<repo>:<tag>" "a malformed entry is refused" \
  "$DELETE_SH" "$KEEP" ddev/ddev-webserver
assert_eq "0" "$(wc -l < "$CURL_LOG" | tr -d ' ')" "validation failures happen before any request"

assert_fails_with "exceeds CLEANUP_MAX_DELETE" "a list over the limit is refused" \
  env CLEANUP_MAX_DELETE=1 "$DELETE_SH" "$KEEP" ddev/ddev-webserver:bbbbbbbbbb ddev/ddev-webserver:eeeeeeeeee

serve_standard_fixture
assert_fails_with "DOCKERHUB_USERNAME must be set" "--execute needs credentials" \
  env -u DOCKERHUB_USERNAME -u DOCKERHUB_TOKEN "$DELETE_SH" --execute "$KEEP" ddev/ddev-webserver:bbbbbbbbbb
assert_eq "0" "$(log_count DELETE)" "no deletion without credentials"

serve_standard_fixture
printf 'ddev/ddev-webserver:bbbbbbbbbb\nddev/ddev-webserver:eeeeeeeeee, ddev/ddev-webserver:bbbbbbbbbb\n' |
  DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$DELETE_SH" --execute "$KEEP" >/dev/null 2>&1
assert_eq "$(printf '%s\n' \
  "DELETE https://hub.docker.com/v2/repositories/ddev/ddev-webserver/tags/bbbbbbbbbb/ Authorization: JWT stub-token" \
  "DELETE https://hub.docker.com/v2/repositories/ddev/ddev-webserver/tags/eeeeeeeeee/ Authorization: JWT stub-token")" \
  "$(grep '^DELETE ' "$CURL_LOG")" "--execute deletes each requested tag once, from stdin, with the login token"

serve_standard_fixture
echo "https://hub.docker.com/v2/repositories/ddev/ddev-webserver/tags/bbbbbbbbbb/" > "$CURL_FAIL_URLS"
assert_fails_with "1 of 2 deletions failed" "a failed deletion fails the run" \
  env DOCKERHUB_USERNAME=u DOCKERHUB_TOKEN=t "$DELETE_SH" --execute "$KEEP" ddev/ddev-webserver:bbbbbbbbbb ddev/ddev-webserver:eeeeeeeeee
assert_eq "2" "$(log_count DELETE)" "a failed deletion doesn't stop the rest"

if [ "$FAILURES" -gt 0 ]; then
  echo "${FAILURES} test(s) failed" >&2
  exit 1
fi
echo "All tests passed"
