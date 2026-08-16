#!/usr/bin/env bash
# autotag_test.sh - unit tests for hash-paths.sh and autotag.sh.
#
# Exercises the tag-format/change-detection/rewrite logic against a scratch
# git repository and a stubbed `docker`, without building any real images.
# Run with:
#   containers/autotag_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HASH_PATHS="$SCRIPT_DIR/hash-paths.sh"
AUTOTAG="$SCRIPT_DIR/autotag.sh"

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

# --- Stub `docker`, controlled by a marker file, so "image already exists
# locally" can be flipped on/off without a real Docker daemon, and every
# invocation is logged so we can assert docker is never called at all on
# the unchanged/no-op path.
BINDIR="$WORKDIR/bin"
mkdir -p "$BINDIR"
export DOCKER_EXISTING_REF_FILE="$WORKDIR/docker_existing_ref"
export DOCKER_CALL_LOG="$WORKDIR/docker_calls.log"
: > "$DOCKER_EXISTING_REF_FILE"
: > "$DOCKER_CALL_LOG"
cat > "$BINDIR/docker" <<'DOCKEREOF'
#!/usr/bin/env bash
set -eu -o pipefail
echo "$*" >> "$DOCKER_CALL_LOG"
if [ "$1" = "image" ] && [ "$2" = "inspect" ]; then
  ref="$3"
  grep -qxF "$ref" "$DOCKER_EXISTING_REF_FILE"
  exit $?
fi
echo "docker stub: unexpected invocation: $*" >&2
exit 1
DOCKEREOF
chmod +x "$BINDIR/docker"
export PATH="$BINDIR:$PATH"

# --- Scratch git repo with one throwaway "image" directory.
REPO="$WORKDIR/repo"
mkdir -p "$REPO/imgdir"
(
  cd "$REPO"
  git init -q
  git config user.email test@example.com
  git config user.name test
  git config commit.gpgsign false
  echo "original" > imgdir/Dockerfile
  git add imgdir
  git commit -q -m "initial"
)

VERSIONCONSTANTS="$WORKDIR/versionconstants.go"
cat > "$VERSIONCONSTANTS" <<'EOF'
package versionconstants

// WebTag defines the default web image tag
var WebTag = "v1.0.0" // some-old-branch-v1.0.0

// WebTagBranch is the branch WebTag's content was built from.
var WebTagBranch = "some-old-branch"
EOF

export VERSIONCONSTANTS_FILE="$VERSIONCONSTANTS"
export HASH_LEN=10

cd "$REPO"

# 1. hash-paths.sh determinism.
h1="$("$HASH_PATHS" imgdir)"
h2="$("$HASH_PATHS" imgdir)"
assert_eq "$h1" "$h2" "hash-paths.sh is deterministic"
assert_eq "10" "${#h1}" "hash-paths.sh returns HASH_LEN characters"

# 2. hash-paths.sh reacts to a dirty-tree content change, and reverts cleanly.
echo "changed" > imgdir/Dockerfile
h3="$("$HASH_PATHS" imgdir)"
if [ "$h1" = "$h3" ]; then
  fail "hash-paths.sh should change when a tracked file's content changes"
else
  pass "hash-paths.sh detects a dirty-tree content change"
fi
git checkout -q -- imgdir/Dockerfile
h4="$("$HASH_PATHS" imgdir)"
assert_eq "$h1" "$h4" "hash-paths.sh returns to original hash after revert"

# 3. hash-paths.sh reacts to a new untracked-but-not-ignored file.
echo "new" > imgdir/new-file.txt
h5="$("$HASH_PATHS" imgdir)"
if [ "$h1" = "$h5" ]; then
  fail "hash-paths.sh should change when an untracked file is added"
else
  pass "hash-paths.sh detects a new untracked file"
fi

# 4. hash-paths.sh respects .gitignore.
echo "ignored/" > .gitignore
mkdir -p imgdir/ignored
echo "should not affect hash" > imgdir/ignored/whatever.txt
h6="$("$HASH_PATHS" imgdir)"
assert_eq "$h5" "$h6" "hash-paths.sh ignores gitignored files"
rm -rf imgdir/ignored .gitignore
rm -f imgdir/new-file.txt

# --- autotag.sh scenarios. Back to the original (h1) content at this point. ---

# 5. --print-only prints the candidate tag and touches neither docker nor the file.
before="$(cat "$VERSIONCONSTANTS")"
candidate="$("$AUTOTAG" --print-only WebTag ddev/dummy-image imgdir)"
after="$(cat "$VERSIONCONSTANTS")"
assert_eq "$before" "$after" "--print-only does not modify the versionconstants file"
current_hash="$("$HASH_PATHS" imgdir)"
assert_eq "$current_hash" "$candidate" "--print-only candidate tag is the bare content hash"
calls="$(count_lines "$DOCKER_CALL_LOG")"
assert_eq "0" "$calls" "--print-only makes no docker calls"

# 6. Change detected (today's fixture tag is "v1.0.0", never a real hash),
#    no local image exists -> build runs, file is rewritten.
BUILD_MARKER="$WORKDIR/build_ran"
rm -f "$BUILD_MARKER"
"$AUTOTAG" WebTag ddev/dummy-image imgdir -- bash -c "touch '$BUILD_MARKER'"
if [ -f "$BUILD_MARKER" ]; then
  pass "build command ran when the tag changed and no local image existed"
else
  fail "build command should have run when the tag changed and no local image existed"
fi

current_branch="$(git rev-parse --abbrev-ref HEAD | sed -E 's/[^A-Za-z0-9_.-]+/-/g')"
new_tag_line="$(grep '^var WebTag = ' "$VERSIONCONSTANTS")"
assert_eq "var WebTag = \"${current_hash}\" // ${current_branch}-${current_hash}" "$new_tag_line" \
  "the tag becomes the bare hash, with the branch alias as a trailing comment"
assert_eq "var WebTagBranch = \"${current_branch}\"" "$(grep '^var WebTagBranch = ' "$VERSIONCONSTANTS")" \
  "the companion Branch variable is updated for ddev version"

# 7. Idempotency: re-run with no further changes -> no-op. No build, no file
#    rewrite, and (the key design property) no docker call at all.
rm -f "$BUILD_MARKER"
before2="$(cat "$VERSIONCONSTANTS")"
calls_before="$(count_lines "$DOCKER_CALL_LOG")"
"$AUTOTAG" WebTag ddev/dummy-image imgdir -- bash -c "touch '$BUILD_MARKER'"
after2="$(cat "$VERSIONCONSTANTS")"
calls_after="$(count_lines "$DOCKER_CALL_LOG")"
if [ -f "$BUILD_MARKER" ]; then
  fail "unexpected rebuild on unchanged content"
else
  pass "no rebuild on unchanged content"
fi
assert_eq "$before2" "$after2" "versionconstants file untouched on unchanged content"
assert_eq "$calls_before" "$calls_after" "no docker calls at all on the unchanged/no-op path"

# 8. Change detected again, but a local image already exists at the computed
#    tag -> build is skipped, but the file is still rewritten.
echo "changed again" > imgdir/Dockerfile
new_hash="$("$HASH_PATHS" imgdir)"
echo "ddev/dummy-image:${new_hash}" > "$DOCKER_EXISTING_REF_FILE"
rm -f "$BUILD_MARKER"
"$AUTOTAG" WebTag ddev/dummy-image imgdir -- bash -c "touch '$BUILD_MARKER'"
if [ -f "$BUILD_MARKER" ]; then
  fail "build should have been skipped when a local image already exists at the computed tag"
else
  pass "build skipped when a local image already exists at the computed tag"
fi
if grep -q "^var WebTag = \"${new_hash}\"" "$VERSIONCONSTANTS"; then
  pass "versionconstants file rewritten even when the build was skipped"
else
  fail "versionconstants file should still be rewritten when the build is skipped"
fi

# 9. A line still carrying the old <branch>-<hash> form is migrated to the bare
#    hash on the next run, even though its trailing hash already matches.
cat > "$VERSIONCONSTANTS" <<EOF
package versionconstants

// WebTag defines the default web image tag
var WebTag = "legacy-branch-${new_hash}" // legacy-branch-${new_hash}

// WebTagBranch is the branch WebTag's content was built from.
var WebTagBranch = "legacy-branch"
EOF
"$AUTOTAG" WebTag ddev/dummy-image imgdir -- bash -c "touch '$BUILD_MARKER'"
assert_eq "var WebTag = \"${new_hash}\" // ${current_branch}-${new_hash}" \
  "$(grep '^var WebTag = ' "$VERSIONCONSTANTS")" \
  "an old <branch>-<hash> value is migrated to the bare hash"

if [ "$FAILURES" -eq 0 ]; then
  echo "All autotag_test.sh checks passed."
  exit 0
else
  echo "$FAILURES autotag_test.sh check(s) failed." >&2
  exit 1
fi
