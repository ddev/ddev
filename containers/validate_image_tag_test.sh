#!/usr/bin/env bash
# validate_image_tag_test.sh - unit tests for validate-image-tag.sh.
#
# Pure string-format checks, no external stubs needed.
# Run with:
#   containers/validate_image_tag_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VALIDATE="$SCRIPT_DIR/validate-image-tag.sh"

FAILURES=0

fail() {
  echo "FAIL: $1" >&2
  FAILURES=$((FAILURES + 1))
}

pass() {
  echo "PASS: $1"
}

assert_valid() {
  local tag="$1"
  if "$VALIDATE" "$tag" >/dev/null 2>&1; then
    pass "accepts valid tag '$tag'"
  else
    fail "should have accepted valid tag '$tag'"
  fi
}

assert_invalid() {
  local tag="$1" desc="$2"
  if "$VALIDATE" "$tag" >/dev/null 2>&1; then
    fail "should have rejected $desc ('$tag')"
  else
    pass "rejects $desc ('$tag')"
  fi
}

assert_rejected_because() {
  local tag="$1" needle="$2" desc="$3"
  local output
  if output="$("$VALIDATE" "$tag" 2>&1)"; then
    fail "should have rejected $desc ('$tag')"
  elif [[ "$output" == *"$needle"* ]]; then
    pass "rejects $desc ('$tag') as expected"
  else
    fail "rejected $desc ('$tag') for the wrong reason: $output"
  fi
}

# The canonical form: a bare content hash, what versionconstants.go holds.
assert_valid "0123456789"
assert_valid "36bceca65e"

# The readable alias published alongside it.
assert_valid "20260721_rfay_content_addressed_image_tags-36bceca65e"
assert_valid "main-0123456789"
assert_valid "v1.2.3-rc1-0123456789"

assert_invalid "0123456789a" "an 11-hex-char bare tag"
assert_invalid "012345678" "a 9-hex-char bare tag"
assert_invalid "0123456789A" "a bare tag with an uppercase hex digit"

assert_invalid "latest" "the reserved literal 'latest'"
assert_invalid "stable" "the reserved literal 'stable'"
assert_invalid "v1.2.3" "a bare release tag"
assert_invalid "latest-0123456789a" "a fake tag with an 11-char hash suffix"
assert_invalid "latest-012345678" "a fake tag with a 9-char hash suffix"
assert_invalid "no-hash-suffix" "a tag without a hex hash suffix"
assert_invalid "bad chars!-0123456789" "a tag with disallowed characters"
assert_invalid "UPPERHASH-0123456789AB" "a tag with an uppercase hash suffix"
assert_invalid "-leading-dash-0123456789" "a tag Docker would reject for its leading dash"
assert_invalid ".leading-dot-0123456789" "a tag Docker would reject for its leading dot"

# A well-formed hash suffix must not be a way to smuggle a tag that reads as
# an official one; these are the checks a format-only validator would miss.
assert_rejected_because "latest-0123456789" "reserved tag" "'latest' dressed up with a hash suffix"
assert_rejected_because "v1.2.3-0123456789" "release tag" "a release tag dressed up with a hash suffix"

if [ "$FAILURES" -eq 0 ]; then
  echo "All validate_image_tag_test.sh checks passed."
  exit 0
else
  echo "$FAILURES validate_image_tag_test.sh check(s) failed." >&2
  exit 1
fi
