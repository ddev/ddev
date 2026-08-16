#!/usr/bin/env bash
# required_image_tag_test.sh - unit tests for required-image-tag.sh.
#
# Runs against a throwaway versionconstants.go and this checkout's real
# content hashes. No Docker, no network.
# Run with:
#   containers/required_image_tag_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REQUIRED_IMAGE_TAG="$SCRIPT_DIR/required-image-tag.sh"
HASH_PATHS="$SCRIPT_DIR/hash-paths.sh"

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
    fail "$desc (expected '$expected', got '$actual')"
  fi
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

export VERSIONCONSTANTS_FILE="$WORKDIR/versionconstants.go"
HASH_PATH_ARGS=(containers/ddev-xhgui containers/containers_shared.mk)
CURRENT_HASH="$("$HASH_PATHS" "${HASH_PATH_ARGS[@]}")"

# 1. Committed tag still matches the content: returned as-is, keeping the
#    branch prefix of whatever branch last changed the image. This is the case
#    that makes wait-for-images.sh wait for a tag that actually exists.
echo "var XhguiTag = \"an_old_branch-${CURRENT_HASH}\" // trailing comment" > "$VERSIONCONSTANTS_FILE"
OUTPUT="$(REQUIRED_IMAGE_TAG_BRANCH=current_branch "$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")"
assert_eq "committed an_old_branch-${CURRENT_HASH}" "$OUTPUT" "keeps the committed tag when the hash still matches"

# 2. Content no longer matches: the tag autotag.sh would rewrite it to,
#    prefixed with the current branch.
echo "var XhguiTag = \"an_old_branch-0000000000\"" > "$VERSIONCONSTANTS_FILE"
OUTPUT="$(REQUIRED_IMAGE_TAG_BRANCH=current_branch "$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")"
assert_eq "recomputed current_branch-${CURRENT_HASH}" "$OUTPUT" "recomputes a branch-prefixed tag when the hash changed"

# 3. Branch names are sanitized to the tag charset, the same way autotag.sh
#    does it - a fork may name its branch anything git accepts.
# shellcheck disable=SC2016 # the un-expanded $(id) is the point
OUTPUT="$(REQUIRED_IMAGE_TAG_BRANCH='feature/oh no$(id)' "$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")"
assert_eq "recomputed feature-oh-no-id--${CURRENT_HASH}" "$OUTPUT" "sanitizes the branch name into the tag charset"
if "$SCRIPT_DIR/validate-image-tag.sh" "${OUTPUT#recomputed }" >/dev/null 2>&1; then
  pass "a sanitized hostile branch name still yields a pushable tag"
else
  fail "sanitized branch name should still yield a tag validate-image-tag.sh accepts: $OUTPUT"
fi

# 4. A missing tag variable is a hard error, not an empty tag.
echo "var SomethingElse = \"whatever\"" > "$VERSIONCONSTANTS_FILE"
OUTPUT="$("$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -ne 0 ]; then
  pass "errors out when the tag variable isn't in versionconstants.go"
else
  fail "should error out when the tag variable isn't in versionconstants.go"
fi
case "$OUTPUT" in
  *"could not find"*XhguiTag*) pass "missing-variable message names the variable" ;;
  *) fail "missing-variable message should name the variable: $OUTPUT" ;;
esac

# 5. Usage error on too few arguments.
if "$REQUIRED_IMAGE_TAG" XhguiTag >/dev/null 2>&1; then
  fail "should reject a missing hash path"
else
  pass "rejects a missing hash path"
fi

# --- image-tag-for.sh, the lookup the manual push workflows use to default
# their tag input. Against the real image-configs.sh and this checkout.
unset VERSIONCONSTANTS_FILE
IMAGE_TAG_FOR="$SCRIPT_DIR/image-tag-for.sh"

web_tag="$("$IMAGE_TAG_FOR" ddev-webserver)"
web_hash="$("$HASH_PATHS" containers/ddev-webserver containers/containers_shared.mk)"
case "$web_tag" in
  *"$web_hash") pass "image-tag-for.sh resolves ddev-webserver to a tag with the current hash" ;;
  *) fail "ddev-webserver tag '$web_tag' should end in '$web_hash'" ;;
esac

# Every db variant shares BaseDBTag, so the lookup has to agree across them.
assert_eq "$("$IMAGE_TAG_FOR" ddev-dbserver-mariadb-11.8)" "$("$IMAGE_TAG_FOR" ddev-dbserver-mysql-8.0)" \
  "image-tag-for.sh gives every db variant the same tag"

# An image the automatic flow doesn't cover has no hash to derive, so the
# caller has to be told rather than handed a wrong tag.
if "$IMAGE_TAG_FOR" test-ssh-server >/dev/null 2>&1; then
  fail "should refuse an image that isn't content-addressed"
else
  pass "refuses an image that isn't content-addressed"
fi
if "$IMAGE_TAG_FOR" no-such-image >/dev/null 2>&1; then
  fail "should refuse an unknown image"
else
  pass "refuses an unknown image"
fi

if [ "$FAILURES" -eq 0 ]; then
  echo "All required_image_tag_test.sh checks passed."
  exit 0
else
  echo "$FAILURES required_image_tag_test.sh check(s) failed." >&2
  exit 1
fi
