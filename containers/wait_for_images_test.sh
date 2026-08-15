#!/usr/bin/env bash
# wait_for_images_test.sh - unit tests for wait-for-images.sh.
#
# Exercises the fast-path/retry/give-up logic against a stubbed `docker` and
# a fabricated versionconstants.go, without a real registry or real sleeps.
# Run with:
#   containers/wait_for_images_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WAIT_FOR_IMAGES="$SCRIPT_DIR/wait-for-images.sh"

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

VERSIONCONSTANTS="$WORKDIR/versionconstants.go"
write_versionconstants() {
  cat > "$VERSIONCONSTANTS" <<'EOF'
package versionconstants

var WebTag = "main-1111111111"
var TraefikRouterTag = "main-2222222222"
var SSHAuthTag = "main-3333333333"
var XhguiTag = "main-4444444444"
var BaseDBTag = "main-5555555555"
EOF
}
write_versionconstants

export VERSIONCONSTANTS_FILE="$VERSIONCONSTANTS"
export DOCKER_ORG=ddevhq

# 1. Fast path: every tag already exists -> one docker call per image, no sleep.
cat > "$DOCKER_EXISTING_REF_FILE" <<'EOF'
ddevhq/ddev-webserver:main-1111111111
ddevhq/ddev-traefik-router:main-2222222222
ddevhq/ddev-ssh-agent:main-3333333333
ddevhq/ddev-xhgui:main-4444444444
ddevhq/ddev-dbserver-mariadb-11.8:main-5555555555
EOF
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
cat >> "$DOCKER_EXISTING_REF_FILE" <<'EOF'
ddevhq/ddev-webserver:main-1111111111
ddevhq/ddev-traefik-router:main-2222222222
ddevhq/ddev-ssh-agent:main-3333333333
ddevhq/ddev-xhgui:main-4444444444
EOF
echo "ddevhq/ddev-dbserver-mariadb-11.8:main-5555555555" > "$DOCKER_DELAYED_REF_FILE"
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

# 4. A tag variable missing from versionconstants.go is a clear, immediate error.
cat > "$VERSIONCONSTANTS" <<'EOF'
package versionconstants

var WebTag = "main-1111111111"
EOF
: > "$DOCKER_EXISTING_REF_FILE"
echo "ddevhq/ddev-webserver:main-1111111111" >> "$DOCKER_EXISTING_REF_FILE"
OUTPUT="$(WAIT_FOR_IMAGES_ATTEMPTS=1 "$WAIT_FOR_IMAGES" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -ne 0 ]; then
  pass "errors out when a tag var is missing from versionconstants.go"
else
  fail "should error out when a tag var is missing from versionconstants.go"
fi
case "$OUTPUT" in
  *"could not find"*"TraefikRouterTag"*) pass "missing-tag-var message names the missing var" ;;
  *) fail "missing-tag-var message should name the missing var: $OUTPUT" ;;
esac

if [ "$FAILURES" -eq 0 ]; then
  echo "All wait_for_images_test.sh checks passed."
  exit 0
else
  echo "$FAILURES wait_for_images_test.sh check(s) failed." >&2
  exit 1
fi
