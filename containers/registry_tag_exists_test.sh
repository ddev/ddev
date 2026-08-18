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

# --- Stub `docker`, controlled by a marker file listing which refs "exist".
BINDIR="$WORKDIR/bin"
mkdir -p "$BINDIR"
export DOCKER_EXISTING_REF_FILE="$WORKDIR/docker_existing_refs"
export DOCKER_CALL_LOG="$WORKDIR/docker_calls.log"
: > "$DOCKER_EXISTING_REF_FILE"
: > "$DOCKER_CALL_LOG"
cat > "$BINDIR/docker" <<'DOCKEREOF'
#!/usr/bin/env bash
set -eu -o pipefail
echo "$*" >> "$DOCKER_CALL_LOG"
if [ "$1" = "buildx" ] && [ "$2" = "imagetools" ] && [ "$3" = "inspect" ]; then
  ref="$4"
  grep -qxF "$ref" "$DOCKER_EXISTING_REF_FILE"
  exit $?
fi
echo "docker stub: unexpected invocation: $*" >&2
exit 1
DOCKEREOF
chmod +x "$BINDIR/docker"
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

# 3. Exactly one docker call per invocation - no retries/loops in this script
#    (retry/backoff, if wanted, is the caller's job, e.g. wait-for-images.sh).
# BSD wc pads its output with spaces; GNU wc does not.
calls="$(wc -l < "$DOCKER_CALL_LOG" | tr -d '[:space:]')"
if [ "$calls" -eq 2 ]; then
  pass "made exactly one docker call per invocation"
else
  fail "expected 2 total docker calls across both invocations, got $calls"
fi

# 4. Usage error on wrong argument count.
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
