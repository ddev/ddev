#!/usr/bin/env bash
# wait_for_images_test.sh - unit tests for wait-for-images.sh.
#
# Exercises the fast-path/retry/give-up logic against a stubbed `docker` and a
# throwaway versionconstants.go, so the committed-vs-recomputed decision can be
# driven both ways without touching this checkout. No real registry, no real
# sleeps.
# Run with:
#   containers/wait_for_images_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WAIT_FOR_IMAGES="$SCRIPT_DIR/wait-for-images.sh"
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

# BSD wc pads its output with spaces; GNU wc does not.
count_lines() {
  wc -l < "$1" | tr -d '[:space:]'
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# --- Stub `docker`: a ref can be configured to "exist" outright, or to only
# start existing after N calls (via a counter file), so the eventually-recovers
# scenario is deterministic - no real sleeps or background processes needed.
BINDIR="$WORKDIR/bin"
mkdir -p "$BINDIR"
export DOCKER_EXISTING_REF_FILE="$WORKDIR/docker_existing_refs"
export DOCKER_DELAYED_REF_FILE="$WORKDIR/docker_delayed_ref"
export DOCKER_DELAYED_COUNTER_DIR="$WORKDIR/docker_delayed_counters"
export DOCKER_CALL_LOG="$WORKDIR/docker_calls.log"
mkdir -p "$DOCKER_DELAYED_COUNTER_DIR"
: > "$DOCKER_EXISTING_REF_FILE"
: > "$DOCKER_DELAYED_REF_FILE"
: > "$DOCKER_CALL_LOG"
cat > "$BINDIR/docker" <<'DOCKEREOF'
#!/usr/bin/env bash
set -eu -o pipefail
echo "$*" >> "$DOCKER_CALL_LOG"
if [ "$1" = "buildx" ] && [ "$2" = "imagetools" ] && [ "$3" = "inspect" ]; then
  ref="$4"
  if grep -qxF "$ref" "$DOCKER_EXISTING_REF_FILE"; then
    exit 0
  fi
  delayed_ref="$(cat "$DOCKER_DELAYED_REF_FILE" 2>/dev/null || true)"
  if [ -n "$delayed_ref" ] && [ "$ref" = "$delayed_ref" ]; then
    counter_file="$DOCKER_DELAYED_COUNTER_DIR/count"
    count="$(cat "$counter_file" 2>/dev/null || echo 0)"
    count=$((count + 1))
    echo "$count" > "$counter_file"
    [ "$count" -ge 3 ] && exit 0 || exit 1
  fi
  exit 1
fi
echo "docker stub: unexpected invocation: $*" >&2
exit 1
DOCKEREOF
chmod +x "$BINDIR/docker"

# --- Stub `sleep` so retry-budget tests run instantly and we can count waits.
export SLEEP_CALL_LOG="$WORKDIR/sleep_calls.log"
: > "$SLEEP_CALL_LOG"
cat > "$BINDIR/sleep" <<'SLEEPEOF'
#!/usr/bin/env bash
echo "$*" >> "$SLEEP_CALL_LOG"
SLEEPEOF
chmod +x "$BINDIR/sleep"

export PATH="$BINDIR:$PATH"

export DOCKER_ORG=ddevhq

# shellcheck source=containers/image-configs.sh
source "$SCRIPT_DIR/image-configs.sh"

# --- A throwaway versionconstants.go, so the tags waited for are whatever this
# test says they are rather than whatever the checkout happens to carry.
export VERSIONCONSTANTS_FILE="$WORKDIR/versionconstants.go"

REPOS=()
TAG_VARS=()
declare -A HASH_BY_VAR
declare -A TAG_BY_VAR
for entry in "${DDEV_IMAGE_CONFIGS[@]}"; do
  IFS='|' read -r repo_suffix tag_var hash_paths _ <<< "$entry"
  if [ -z "${HASH_BY_VAR[$tag_var]:-}" ]; then
    # shellcheck disable=SC2086 # hash_paths is a space-separated path list
    HASH_BY_VAR["$tag_var"]="$("$HASH_PATHS" $hash_paths)"
    TAG_BY_VAR["$tag_var"]="${HASH_BY_VAR[$tag_var]}"
  fi
  REPOS+=("ddevhq/${repo_suffix}")
  TAG_VARS+=("$tag_var")
done
IMAGE_COUNT="${#REPOS[@]}"

# All 20 ddev-dbserver variants share BaseDBTag, so the file has one line per
# distinct tag variable, not one per image.
write_versionconstants() {
  : > "$VERSIONCONSTANTS_FILE"
  for var in "${!TAG_BY_VAR[@]}"; do
    echo "var ${var} = \"${TAG_BY_VAR[$var]}\"" >> "$VERSIONCONSTANTS_FILE"
  done
}

mark_all_existing() {
  : > "$DOCKER_EXISTING_REF_FILE"
  for i in "${!REPOS[@]}"; do
    echo "${REPOS[$i]}:${TAG_BY_VAR[${TAG_VARS[$i]}]}" >> "$DOCKER_EXISTING_REF_FILE"
  done
}

write_versionconstants

# 1. Fast path: every committed tag already exists -> one docker call per
#    image, no sleep. This is the shape of every pull request that doesn't
#    change a container image.
mark_all_existing
: > "$DOCKER_CALL_LOG"
: > "$SLEEP_CALL_LOG"
OUTPUT="$("$WAIT_FOR_IMAGES" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -eq 0 ]; then
  pass "fast path succeeds when every committed tag already exists"
else
  fail "fast path should succeed when every committed tag already exists: $OUTPUT"
fi
assert_eq "$IMAGE_COUNT" "$(count_lines "$DOCKER_CALL_LOG")" "fast path makes exactly one docker call per image"
assert_eq "0" "$(count_lines "$SLEEP_CALL_LOG")" "fast path never sleeps"
case "$OUTPUT" in
  *"found ${REPOS[0]}:${TAG_BY_VAR[WebTag]}"*) pass "prints confirmation for each found tag" ;;
  *) fail "should print confirmation for each found tag: $OUTPUT" ;;
esac

# 2. The regression that made every non-containers pull request hang was a tag
#    that depended on the current branch. The tag is now the bare content hash,
#    so the same checkout resolves to the same tag on any branch, in any fork.
case "$OUTPUT" in
  *"found ${REPOS[0]}:${HASH_BY_VAR[WebTag]}"*) pass "waits for the bare content hash, with no branch prefix" ;;
  *) fail "should wait for the bare hash '${HASH_BY_VAR[WebTag]}': $OUTPUT" ;;
esac
OUTPUT_OTHER_BRANCH="$(REQUIRED_IMAGE_TAG_BRANCH='some/other branch' "$WAIT_FOR_IMAGES" 2>&1)"
assert_eq "$OUTPUT" "$OUTPUT_OTHER_BRANCH" "the tags waited for don't depend on the branch"

# 2b. Every ddev-dbserver variant is waited for, not just the default one that
#     `make` builds locally - they all share BaseDBTag, so a dbserver change
#     invalidates all of them at once.
for variant_repo in ddevhq/ddev-dbserver-mysql-8.0 ddevhq/ddev-dbserver-mariadb-10.11 ddevhq/ddev-dbserver-mariadb-5.5; do
  case "$OUTPUT" in
    *"found ${variant_repo}:"*) pass "waits for the non-default variant ${variant_repo}" ;;
    *) fail "should wait for the non-default variant ${variant_repo}: $OUTPUT" ;;
  esac
done

# 3. Content that no longer matches versionconstants.go is built locally by
#    make, so there is nothing to wait for and no registry call at all.
TAG_BY_VAR[WebTag]="0000000000"
write_versionconstants
mark_all_existing
: > "$DOCKER_CALL_LOG"
: > "$SLEEP_CALL_LOG"
OUTPUT="$("$WAIT_FOR_IMAGES" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -eq 0 ]; then
  pass "succeeds without waiting when content differs from versionconstants.go"
else
  fail "should succeed when content differs from versionconstants.go: $OUTPUT"
fi
assert_eq "$(( IMAGE_COUNT - 1 ))" "$(count_lines "$DOCKER_CALL_LOG")" "skips the registry check for the locally-built image"
assert_eq "0" "$(count_lines "$SLEEP_CALL_LOG")" "never sleeps for a locally-built image"
case "$OUTPUT" in
  *"not waiting"*) pass "says why it isn't waiting for the changed image" ;;
  *) fail "should explain why it isn't waiting: $OUTPUT" ;;
esac
TAG_BY_VAR[WebTag]="${HASH_BY_VAR[WebTag]}"
write_versionconstants

# 3b. A changed dbserver still has to wait for the 19 variants `make` doesn't
#     build locally. The tag is the bare content hash, so it's the same string
#     image-build-push.yml pushes and waiting for it is well-defined - only the
#     default variant is skipped.
TAG_BY_VAR[BaseDBTag]="0000000000"
write_versionconstants
: > "$DOCKER_EXISTING_REF_FILE"
for i in "${!REPOS[@]}"; do
  tag="${TAG_BY_VAR[${TAG_VARS[$i]}]}"
  [ "${TAG_VARS[$i]}" = "BaseDBTag" ] && tag="${HASH_BY_VAR[BaseDBTag]}"
  echo "${REPOS[$i]}:${tag}" >> "$DOCKER_EXISTING_REF_FILE"
done
: > "$DOCKER_CALL_LOG"
OUTPUT="$("$WAIT_FOR_IMAGES" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -eq 0 ]; then
  pass "waits for the db variants make doesn't build, rather than failing"
else
  fail "should wait for the non-locally-built db variants: $OUTPUT"
fi
case "$OUTPUT" in
  *"ddevhq/ddev-dbserver-mariadb-11.8 content differs"*) pass "skips only the default variant make builds locally" ;;
  *) fail "should skip the locally-built default variant: $OUTPUT" ;;
esac
case "$OUTPUT" in
  *"found ddevhq/ddev-dbserver-mysql-8.0:${HASH_BY_VAR[BaseDBTag]}"*) pass "waits for a non-default variant at the bare hash tag" ;;
  *) fail "should wait for the non-default variants at the recomputed hash: $OUTPUT" ;;
esac
TAG_BY_VAR[BaseDBTag]="${HASH_BY_VAR[BaseDBTag]}"
write_versionconstants

# 4. A tag that's initially missing but becomes available on the 3rd check.
LAST=$(( IMAGE_COUNT - 1 ))
: > "$DOCKER_EXISTING_REF_FILE"
for i in "${!REPOS[@]}"; do
  [ "$i" -eq "$LAST" ] && continue
  echo "${REPOS[$i]}:${TAG_BY_VAR[${TAG_VARS[$i]}]}" >> "$DOCKER_EXISTING_REF_FILE"
done
echo "${REPOS[$LAST]}:${TAG_BY_VAR[${TAG_VARS[$LAST]}]}" > "$DOCKER_DELAYED_REF_FILE"
rm -f "$DOCKER_DELAYED_COUNTER_DIR/count"
: > "$SLEEP_CALL_LOG"
if WAIT_FOR_IMAGES_ATTEMPTS=5 WAIT_FOR_IMAGES_SLEEP=0 "$WAIT_FOR_IMAGES" >/dev/null 2>&1; then
  pass "recovers once a previously-missing tag appears within the attempt budget"
else
  fail "should recover once a previously-missing tag appears within the attempt budget"
fi
assert_eq "2" "$(count_lines "$SLEEP_CALL_LOG")" "sleeps twice while waiting for the tag to become available on the 3rd check"
: > "$DOCKER_DELAYED_REF_FILE"

# 5. Gives up cleanly after exhausting the attempt budget, with a clear message.
: > "$DOCKER_EXISTING_REF_FILE"
: > "$SLEEP_CALL_LOG"
OUTPUT="$(WAIT_FOR_IMAGES_ATTEMPTS=3 WAIT_FOR_IMAGES_SLEEP=0 "$WAIT_FOR_IMAGES" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -ne 0 ]; then
  pass "gives up (non-zero exit) once the attempt budget is exhausted"
else
  fail "should give up (non-zero exit) once the attempt budget is exhausted"
fi
case "$OUTPUT" in
  *"gave up waiting"*"has the maintainer approved"*) pass "give-up message is actionable" ;;
  *) fail "give-up message should mention giving up and approval: $OUTPUT" ;;
esac
assert_eq "2" "$(count_lines "$SLEEP_CALL_LOG")" "sleeps exactly (attempts - 1) times before giving up on the first (unavailable) image"

if [ "$FAILURES" -eq 0 ]; then
  echo "All wait_for_images_test.sh checks passed."
  exit 0
else
  echo "$FAILURES wait_for_images_test.sh check(s) failed." >&2
  exit 1
fi
