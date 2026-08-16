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

# 1. Committed value already the current hash -> committed, and the tag is
#    that hash. This is the case that makes wait-for-images.sh wait.
echo "var XhguiTag = \"${CURRENT_HASH}\" // some_branch-${CURRENT_HASH}" > "$VERSIONCONSTANTS_FILE"
OUTPUT="$("$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")"
assert_eq "committed ${CURRENT_HASH}" "$OUTPUT" "reports committed when versionconstants.go already names this hash"

# 2. Content no longer matches -> recomputed, same tag. The tag never depends
#    on the branch, which is what lets detect, the runner, and `make` agree.
echo "var XhguiTag = \"0000000000\"" > "$VERSIONCONSTANTS_FILE"
OUTPUT="$("$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")"
assert_eq "recomputed ${CURRENT_HASH}" "$OUTPUT" "reports recomputed, still with the bare content hash"

# 3. A value still in the old <branch>-<hash> form is what ddev would pull and
#    it isn't this tag, so it counts as stale and `make` migrates the line.
echo "var XhguiTag = \"an_old_branch-${CURRENT_HASH}\"" > "$VERSIONCONSTANTS_FILE"
OUTPUT="$("$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")"
assert_eq "recomputed ${CURRENT_HASH}" "$OUTPUT" "treats a legacy branch-prefixed value as stale"

# 4. The branch the caller happens to be on cannot change the answer.
echo "var XhguiTag = \"${CURRENT_HASH}\"" > "$VERSIONCONSTANTS_FILE"
assert_eq "$("$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")" \
  "$(REQUIRED_IMAGE_TAG_BRANCH='some/other branch' "$REQUIRED_IMAGE_TAG" XhguiTag "${HASH_PATH_ARGS[@]}")" \
  "the result is independent of the branch"

if "$SCRIPT_DIR/validate-image-tag.sh" "$CURRENT_HASH" >/dev/null 2>&1; then
  pass "the bare hash tag is one validate-image-tag.sh accepts"
else
  fail "validate-image-tag.sh should accept the bare hash '$CURRENT_HASH'"
fi

# 5. A missing tag variable is a hard error, not an empty tag.
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

# 6. Usage error on too few arguments.
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

# --- check-image-tags.sh, the `make staticrequired` gate. Driven with a
# throwaway versionconstants.go so it can be failed on purpose.
CHECK_IMAGE_TAGS="$SCRIPT_DIR/check-image-tags.sh"

# shellcheck source=containers/image-configs.sh
source "$SCRIPT_DIR/image-configs.sh"

CURRENT_FILE="$WORKDIR/current_versionconstants.go"
: > "$CURRENT_FILE"
seen=""
for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
  IFS='|' read -r _ tag_var hash_paths _ <<< "$entry"
  case " $seen " in *" $tag_var "*) continue ;; esac
  seen="$seen $tag_var"
  # shellcheck disable=SC2086 # hash_paths is a space-separated path list
  echo "var ${tag_var} = \"$("$HASH_PATHS" $hash_paths)\"" >> "$CURRENT_FILE"
done

if VERSIONCONSTANTS_FILE="$CURRENT_FILE" "$CHECK_IMAGE_TAGS" >/dev/null 2>&1; then
  pass "check-image-tags.sh passes when every tag matches the content"
else
  fail "check-image-tags.sh should pass when every tag matches the content"
fi

STALE_FILE="$WORKDIR/stale_versionconstants.go"
sed 's/= "[0-9a-f]*"/= "0000000000"/' "$CURRENT_FILE" > "$STALE_FILE"
OUTPUT="$(VERSIONCONSTANTS_FILE="$STALE_FILE" "$CHECK_IMAGE_TAGS" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -ne 0 ]; then
  pass "check-image-tags.sh fails when versionconstants.go is stale"
else
  fail "check-image-tags.sh should fail when versionconstants.go is stale"
fi
case "$OUTPUT" in
  *"run 'make'"*) pass "the failure says how to fix it" ;;
  *) fail "the failure should tell the contributor to run make: $OUTPUT" ;;
esac
case "$OUTPUT" in
  *WebTag*BaseDBTag*|*BaseDBTag*WebTag*) pass "the failure names which tags are stale" ;;
  *) fail "the failure should name the stale tags: $OUTPUT" ;;
esac

if [ "$FAILURES" -eq 0 ]; then
  echo "All required_image_tag_test.sh checks passed."
  exit 0
else
  echo "$FAILURES required_image_tag_test.sh check(s) failed." >&2
  exit 1
fi
