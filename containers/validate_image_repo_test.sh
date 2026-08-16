#!/usr/bin/env bash
# validate_image_repo_test.sh - unit tests for validate-image-repo.sh.
#
# Pure string checks, no external stubs needed.
# Run with:
#   containers/validate_image_repo_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VALIDATE="$SCRIPT_DIR/validate-image-repo.sh"

FAILURES=0

fail() {
  echo "FAIL: $1" >&2
  FAILURES=$((FAILURES + 1))
}

pass() {
  echo "PASS: $1"
}

export DOCKER_ORG=ddev

assert_valid() {
  local repo="$1"
  if "$VALIDATE" "$repo" >/dev/null 2>&1; then
    pass "accepts '$repo'"
  else
    fail "should have accepted '$repo'"
  fi
}

assert_invalid() {
  local repo="$1" desc="$2"
  if "$VALIDATE" "$repo" >/dev/null 2>&1; then
    fail "should have rejected $desc ('$repo')"
  else
    pass "rejects $desc ('$repo')"
  fi
}

assert_valid "ddev/ddev-webserver"
assert_valid "ddev/ddev-webserver-prod"
assert_valid "ddev/ddev-traefik-router"
assert_valid "ddev/ddev-ssh-agent"
assert_valid "ddev/ddev-xhgui"
assert_valid "ddev/ddev-dbserver-mariadb-11.8"
assert_valid "ddev/ddev-dbserver-mysql-8.0"

assert_invalid "someoneelse/ddev-webserver" "another organization"
assert_invalid "ddev-webserver" "a bare name with no organization"
assert_invalid "ddev/ddev-webserver-evil" "an unknown repository in the right org"
assert_invalid "ddev/ddev-dbserver-postgres-16" "a db engine the flow doesn't build"
assert_invalid "ddev/ddev-dbserver-mariadb-99.9" "a correctly-shaped db variant that isn't in variants.txt"
assert_invalid "ddev/../../etc/passwd" "a path-traversal attempt"
assert_invalid "ddevhq/ddev-webserver" "an org that merely starts the same"

# The db repositories are the variants themselves, so the allowlist tracks
# containers/ddev-dbserver/variants.txt rather than a name pattern.
while IFS= read -r suffix; do
  [ -n "$suffix" ] && assert_valid "ddev/${suffix}"
done < <("$SCRIPT_DIR/ddev-dbserver/variants.sh" repos)

# The org comes from a trusted workflow variable, so a differing DOCKER_ORG
# must move the whole allowlist rather than widen it.
if DOCKER_ORG=ddevhq "$VALIDATE" "ddevhq/ddev-webserver" >/dev/null 2>&1; then
  pass "accepts the configured org when DOCKER_ORG differs"
else
  fail "should accept the configured org when DOCKER_ORG differs"
fi
if DOCKER_ORG=ddevhq "$VALIDATE" "ddev/ddev-webserver" >/dev/null 2>&1; then
  fail "should reject the default org when DOCKER_ORG points elsewhere"
else
  pass "rejects the default org when DOCKER_ORG points elsewhere"
fi

if env -u DOCKER_ORG "$VALIDATE" "ddev/ddev-webserver" >/dev/null 2>&1; then
  fail "should refuse to run with DOCKER_ORG unset"
else
  pass "refuses to run with DOCKER_ORG unset"
fi

if [ "$FAILURES" -eq 0 ]; then
  echo "All validate_image_repo_test.sh checks passed."
  exit 0
else
  echo "$FAILURES validate_image_repo_test.sh check(s) failed." >&2
  exit 1
fi
