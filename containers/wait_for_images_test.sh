#!/usr/bin/env bash
# wait_for_images_test.sh - unit tests for wait-for-images.sh.
#
# Exercises the fast-path/retry/give-up logic against a stubbed `docker` and
# the real hash-paths.sh (run against this checkout's actual content, so the
# expected tags are computed the same way wait-for-images.sh computes them -
# never read from versionconstants.go). No real registry or real sleeps.
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

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# --- Stub `docker`: exists-by-default, except a ref can be configured to
# only start "existing" after N calls (via a per-ref counter file), so the
# eventually-recovers scenario is deterministic - no real sleeps or
# background processes needed.
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
BRANCH="test-branch"
export WAIT_FOR_IMAGES_BRANCH="$BRANCH"

# Same repo_suffix|hash-paths list wait-for-images.sh uses - real hashes of
# this checkout's actual content, computed the same way the script does.
CONFIGS=(
  'ddev-webserver|containers/ddev-webserver containers/containers_shared.mk'
  'ddev-traefik-router|containers/ddev-traefik-router containers/containers_shared.mk'
  'ddev-ssh-agent|containers/ddev-ssh-agent containers/containers_shared.mk'
  'ddev-xhgui|containers/ddev-xhgui containers/containers_shared.mk'
  'ddev-dbserver-mariadb-11.8|containers/ddev-dbserver containers/get_arch.sh'
)
REPOS=()
TAGS=()
for entry in "${CONFIGS[@]}"; do
  IFS='|' read -r repo_suffix hash_paths <<< "$entry"
  hash="$("$HASH_PATHS" $hash_paths)"
  REPOS+=("ddevhq/${repo_suffix}")
  TAGS+=("${BRANCH}-${hash}")
done

# 1. Fast path: every tag already exists -> one docker call per image, no sleep.
: > "$DOCKER_EXISTING_REF_FILE"
for i in "${!REPOS[@]}"; do
  echo "${REPOS[$i]}:${TAGS[$i]}" >> "$DOCKER_EXISTING_REF_FILE"
done
: > "$DOCKER_CALL_LOG"
: > "$SLEEP_CALL_LOG"
if "$WAIT_FOR_IMAGES" >/dev/null 2>&1; then
  pass "fast path succeeds when every tag already exists"
else
  fail "fast path should succeed when every tag already exists"
fi
assert_eq "5" "$(wc -l < "$DOCKER_CALL_LOG")" "fast path makes exactly one docker call per image"
assert_eq "0" "$(wc -l < "$SLEEP_CALL_LOG")" "fast path never sleeps"

# 2. A tag that's initially missing but becomes available on the 3rd check.
: > "$DOCKER_EXISTING_REF_FILE"
for i in "${!REPOS[@]}"; do
  [ "$i" -eq 4 ] && continue
  echo "${REPOS[$i]}:${TAGS[$i]}" >> "$DOCKER_EXISTING_REF_FILE"
done
echo "${REPOS[4]}:${TAGS[4]}" > "$DOCKER_DELAYED_REF_FILE"
rm -f "$DOCKER_DELAYED_COUNTER_DIR/count"
: > "$SLEEP_CALL_LOG"
if WAIT_FOR_IMAGES_ATTEMPTS=5 WAIT_FOR_IMAGES_SLEEP=0 "$WAIT_FOR_IMAGES" >/dev/null 2>&1; then
  pass "recovers once a previously-missing tag appears within the attempt budget"
else
  fail "should recover once a previously-missing tag appears within the attempt budget"
fi
assert_eq "2" "$(wc -l < "$SLEEP_CALL_LOG")" "sleeps twice while waiting for the tag to become available on the 3rd check"
: > "$DOCKER_DELAYED_REF_FILE"

# 3. Gives up cleanly after exhausting the attempt budget, with a clear message.
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
assert_eq "2" "$(wc -l < "$SLEEP_CALL_LOG")" "sleeps exactly (attempts - 1) times before giving up on the first (unavailable) image"

# 4. WAIT_FOR_IMAGES_BRANCH is required - a clear, immediate error when unset.
: > "$DOCKER_EXISTING_REF_FILE"
OUTPUT="$(env -u WAIT_FOR_IMAGES_BRANCH "$WAIT_FOR_IMAGES" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -ne 0 ]; then
  pass "errors out when WAIT_FOR_IMAGES_BRANCH is unset"
else
  fail "should error out when WAIT_FOR_IMAGES_BRANCH is unset"
fi
case "$OUTPUT" in
  *"WAIT_FOR_IMAGES_BRANCH must be set"*) pass "missing-branch message names the required variable" ;;
  *) fail "missing-branch message should name WAIT_FOR_IMAGES_BRANCH: $OUTPUT" ;;
esac

if [ "$FAILURES" -eq 0 ]; then
  echo "All wait_for_images_test.sh checks passed."
  exit 0
else
  echo "$FAILURES wait_for_images_test.sh check(s) failed." >&2
  exit 1
fi
