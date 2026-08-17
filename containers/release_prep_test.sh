#!/usr/bin/env bash
# release_prep_test.sh - unit tests for release-prep.sh and retag-images.sh.
#
# Both run against a scratch git repository holding a copy of containers/, so
# the real working tree is never stamped, and against a `docker` that fails on
# sight, so "builds nothing" is asserted rather than assumed.
# Run with:
#   containers/release_prep_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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

BINDIR="$WORKDIR/bin"
mkdir -p "$BINDIR"
cat > "$BINDIR/docker" <<'DOCKEREOF'
#!/usr/bin/env bash
echo "docker stub: must not be called: $*" >&2
exit 1
DOCKEREOF
chmod +x "$BINDIR/docker"
export PATH="$BINDIR:$PATH"

REPO="$WORKDIR/repo"
mkdir -p "$REPO"
cp -R "$SCRIPT_DIR" "$REPO/containers"
(
  cd "$REPO"
  git init -q
  git config user.email test@example.com
  git config user.name test
  git config commit.gpgsign false
  git add -A
  git commit -q -m initial
)

VERSIONCONSTANTS="$WORKDIR/versionconstants.go"
export VERSIONCONSTANTS_FILE="$VERSIONCONSTANTS"

write_fixture() {
  cat > "$VERSIONCONSTANTS" <<'EOF'
package versionconstants

var WebTag = "aaaaaaaaaa" // some-branch-aaaaaaaaaa
var WebTagBranch = "some-branch"
var BaseDBTag = "bbbbbbbbbb" // some-branch-bbbbbbbbbb
var BaseDBTagBranch = "some-branch"
var TraefikRouterTag = "cccccccccc" // some-branch-cccccccccc
var TraefikRouterTagBranch = "some-branch"
var SSHAuthTag = "dddddddddd" // some-branch-dddddddddd
var SSHAuthTagBranch = "some-branch"
var XhguiTag = "eeeeeeeeee" // some-branch-eeeeeeeeee
var XhguiTagBranch = "some-branch"
EOF
}

RELEASE_PREP="$REPO/containers/release-prep.sh"
RETAG="$REPO/containers/retag-images.sh"
MARKER='# ddev-release-marker:'

TAG_VARS=(WebTag BaseDBTag TraefikRouterTag SSHAuthTag XhguiTag)
IMAGE_DIRS=(ddev-webserver ddev-dbserver ddev-traefik-router ddev-ssh-agent ddev-xhgui)

tag_value() {
  grep -E "^var $1 = " "$VERSIONCONSTANTS" | sed -E "s/^var $1 = \"([^\"]*)\".*/\\1/"
}

branch_value() {
  grep -E "^var $1Branch = " "$VERSIONCONSTANTS" | sed -E "s/^var $1Branch = \"([^\"]*)\".*/\\1/"
}

marker_lines() {
  grep -c "^${MARKER} " "$REPO/containers/$1/Dockerfile" || true
}

marker_line() {
  grep "^${MARKER} " "$REPO/containers/$1/Dockerfile" || true
}

dockerfile_digest() {
  cat "$REPO"/containers/*/Dockerfile | sha256sum | cut -d' ' -f1
}

write_fixture
cd "$REPO"

# 1. Anything that isn't a vX.Y.Z release tag is refused, and refused before
#    any Dockerfile is stamped.
for bad in "" 1.25.4 v1.25 v1.25.4-rc1 latest 20260817_stasadev_test; do
  set +e
  OUTPUT="$("$RELEASE_PREP" "$bad" </dev/null 2>&1)"
  STATUS=$?
  set -e
  if [ "$STATUS" -eq 0 ]; then
    fail "should reject the non-release tag '${bad}'"
  else
    pass "rejects the non-release tag '${bad}'"
  fi
done
assert_eq "0" "$(marker_lines ddev-webserver)" "a rejected tag stamps no Dockerfile"
case "$OUTPUT" in
  *"not a vX.Y.Z release tag"*) pass "rejection message says what shape is wanted" ;;
  *) fail "rejection message should say what shape is wanted: $OUTPUT" ;;
esac

# 2. A release stamps every image and rewrites every tag to a content hash.
"$RELEASE_PREP" v1.25.4 >/dev/null
for d in "${IMAGE_DIRS[@]}"; do
  assert_eq "1" "$(marker_lines "$d")" "$d carries exactly one release marker"
  assert_eq "${MARKER} v1.25.4" "$(marker_line "$d")" "$d marker names the release"
done
for v in "${TAG_VARS[@]}"; do
  t="$(tag_value "$v")"
  if [[ "$t" =~ ^[0-9a-f]{10}$ ]]; then
    pass "$v became a bare content hash"
  else
    fail "$v should be a 10-hex content hash, got '$t'"
  fi
  assert_eq "v1.25.4" "$(branch_value "$v")" "${v}Branch records the release"
done

# 3. The trailing comment keeps naming the branch, not the release: a vX.Y.Z
#    alias is what validate-image-tag.sh rejects, so nothing publishes one.
scratch_branch="$(git rev-parse --abbrev-ref HEAD)"
assert_eq "var WebTag = \"$(tag_value WebTag)\" // ${scratch_branch}-$(tag_value WebTag)" \
  "$(grep '^var WebTag = ' "$VERSIONCONSTANTS")" \
  "the trailing alias comment stays branch-based"

# 4. Re-running for the same release is a no-op, not a second marker.
v1254_tags=()
for v in "${TAG_VARS[@]}"; do v1254_tags+=("$(tag_value "$v")"); done
"$RELEASE_PREP" v1.25.4 >/dev/null
for d in "${IMAGE_DIRS[@]}"; do
  assert_eq "1" "$(marker_lines "$d")" "$d still carries one marker after a repeat run"
done
for i in "${!TAG_VARS[@]}"; do
  assert_eq "${v1254_tags[$i]}" "$(tag_value "${TAG_VARS[$i]}")" \
    "${TAG_VARS[$i]} is unchanged by a repeat run for the same release"
done

# 5. The next release replaces the marker in place and moves every hash.
"$RELEASE_PREP" v1.25.5 >/dev/null
for d in "${IMAGE_DIRS[@]}"; do
  assert_eq "1" "$(marker_lines "$d")" "$d carries one marker after the next release"
  assert_eq "${MARKER} v1.25.5" "$(marker_line "$d")" "$d marker was replaced, not appended"
done
for i in "${!TAG_VARS[@]}"; do
  if [ "${v1254_tags[$i]}" = "$(tag_value "${TAG_VARS[$i]}")" ]; then
    fail "${TAG_VARS[$i]} should move when the release tag changes"
  else
    pass "${TAG_VARS[$i]} moved for the new release"
  fi
  assert_eq "v1.25.5" "$(branch_value "${TAG_VARS[$i]}")" "${TAG_VARS[$i]}Branch tracks the new release"
done

# 6. retag-images.sh rewrites the same tags without touching any Dockerfile.
write_fixture
before_digest="$(dockerfile_digest)"
"$RETAG" >/dev/null
assert_eq "$before_digest" "$(dockerfile_digest)" "retag-images.sh stamps no Dockerfile"
for i in "${!TAG_VARS[@]}"; do
  t="$(tag_value "${TAG_VARS[$i]}")"
  if [[ "$t" =~ ^[0-9a-f]{10}$ ]]; then
    pass "retag-images.sh rewrote ${TAG_VARS[$i]} to a content hash"
  else
    fail "retag-images.sh should rewrite ${TAG_VARS[$i]} to a hash, got '$t'"
  fi
done

if [ "$FAILURES" -eq 0 ]; then
  echo "All release_prep_test.sh checks passed."
  exit 0
else
  echo "$FAILURES release_prep_test.sh check(s) failed." >&2
  exit 1
fi
