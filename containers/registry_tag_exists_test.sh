#!/usr/bin/env bash
# registry_tag_exists_test.sh - unit tests for registry-tag-exists.sh.
#
# Exercises the exists/doesn't-exist/unreachable outcomes against a stubbed
# `docker`, without talking to a real registry.
# Run with:
#   containers/registry_tag_exists_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REGISTRY_TAG_EXISTS="$SCRIPT_DIR/registry-tag-exists.sh"

FAILURES=0

fail() {
  echo "FAIL: $1" >&2
  FAILURES=$((FAILURES + 1))
}

pass() {
  echo "PASS: $1"
}

WORKDIR="$(mktemp -d)"
trap 'rm -rf "$WORKDIR"' EXIT

# --- Stub `docker`, controlled by marker files listing which refs "exist"
# and how many times the registry fails before answering. `sleep` is a no-op.
BINDIR="$WORKDIR/bin"
mkdir -p "$BINDIR"
export DOCKER_EXISTING_REF_FILE="$WORKDIR/docker_existing_refs"
export DOCKER_CALL_LOG="$WORKDIR/docker_calls.log"
export DOCKER_FAILURES_LEFT_FILE="$WORKDIR/docker_failures_left"
: > "$DOCKER_EXISTING_REF_FILE"
: > "$DOCKER_CALL_LOG"
echo 0 > "$DOCKER_FAILURES_LEFT_FILE"
cat > "$BINDIR/docker" <<'DOCKEREOF'
#!/usr/bin/env bash
set -eu -o pipefail
echo "$*" >> "$DOCKER_CALL_LOG"
if [ "$1" = "buildx" ] && [ "$2" = "imagetools" ] && [ "$3" = "inspect" ]; then
  ref="$4"
  failures_left="$(cat "$DOCKER_FAILURES_LEFT_FILE")"
  if [ "$failures_left" -gt 0 ]; then
    echo $((failures_left - 1)) > "$DOCKER_FAILURES_LEFT_FILE"
    echo "ERROR: toomanyrequests: You have reached your unauthenticated pull rate limit" >&2
    exit 1
  fi
  if grep -qxF "$ref" "$DOCKER_EXISTING_REF_FILE"; then
    exit 0
  fi
  case "$ref" in
    ddev/no-such-repo:*) echo "ERROR: pull access denied, repository does not exist or may require authorization: server message: insufficient_scope: authorization failed" >&2 ;;
    *) echo "ERROR: docker.io/${ref}: not found" >&2 ;;
  esac
  exit 1
fi
echo "docker stub: unexpected invocation: $*" >&2
exit 1
DOCKEREOF
chmod +x "$BINDIR/docker"
printf '#!/bin/sh\n' > "$BINDIR/sleep"
chmod +x "$BINDIR/sleep"
export PATH="$BINDIR:$PATH"

# 1. Missing tag -> non-zero exit, no crash.
if "$REGISTRY_TAG_EXISTS" ddev/dummy-image missing-0123456789 >/dev/null 2>&1; then
  fail "should report missing tag as not existing"
else
  pass "reports missing tag as not existing"
fi

# 2. Existing tag -> zero exit.
echo "ddev/dummy-image:present-0123456789" > "$DOCKER_EXISTING_REF_FILE"
if "$REGISTRY_TAG_EXISTS" ddev/dummy-image present-0123456789 >/dev/null 2>&1; then
  pass "reports existing tag as existing"
else
  fail "should report existing tag as existing"
fi

# 3. A definite answer takes one docker call; "not found" is never retried
#    (waiting for a tag to appear is the caller's job, e.g. wait-for-images.sh).
# BSD wc pads its output with spaces; GNU wc does not.
calls="$(wc -l < "$DOCKER_CALL_LOG" | tr -d '[:space:]')"
if [ "$calls" -eq 2 ]; then
  pass "made exactly one docker call per invocation"
else
  fail "expected 2 total docker calls across both invocations, got $calls"
fi

# 4. A repo that doesn't exist yet (a brand-new image) is "not found" too.
rc=0
"$REGISTRY_TAG_EXISTS" ddev/no-such-repo some-tag >/dev/null 2>&1 || rc=$?
if [ "$rc" -eq 1 ]; then
  pass "reports a nonexistent repo as not existing"
else
  fail "expected exit 1 for a nonexistent repo, got $rc"
fi

# 5. A transient registry error is retried until the registry answers.
echo 2 > "$DOCKER_FAILURES_LEFT_FILE"
if "$REGISTRY_TAG_EXISTS" ddev/dummy-image present-0123456789 >/dev/null 2>&1; then
  pass "retries a transient registry error"
else
  fail "should retry a transient registry error and report the existing tag"
fi

# 6. A registry that never answers is exit 3, not "doesn't exist".
echo 100 > "$DOCKER_FAILURES_LEFT_FILE"
rc=0
"$REGISTRY_TAG_EXISTS" ddev/dummy-image present-0123456789 >/dev/null 2>&1 || rc=$?
if [ "$rc" -eq 3 ]; then
  pass "reports an unreachable registry as exit 3"
else
  fail "expected exit 3 for an unreachable registry, got $rc"
fi
echo 0 > "$DOCKER_FAILURES_LEFT_FILE"

# 7. Usage error on wrong argument count.
if "$REGISTRY_TAG_EXISTS" only-one-arg >/dev/null 2>&1; then
  fail "should reject wrong argument count"
else
  pass "rejects wrong argument count"
fi

if [ "$FAILURES" -eq 0 ]; then
  echo "All registry_tag_exists_test.sh checks passed."
  exit 0
else
  echo "$FAILURES registry_tag_exists_test.sh check(s) failed." >&2
  exit 1
fi
