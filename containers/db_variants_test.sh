#!/usr/bin/env bash
# db_variants_test.sh - unit tests for ddev-dbserver/variants.sh.
#
# Pure text transforms, no Docker or network. The interesting property is that
# every consumer's view stays mutually consistent, since the whole point of
# variants.txt is that the four of them can't drift apart.
# Run with:
#   containers/db_variants_test.sh

set -eu -o pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VARIANTS="$SCRIPT_DIR/ddev-dbserver/variants.sh"

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

# --- A fixture matrix, so these assertions don't move every time a database
# version is added or dropped.
export VARIANTS_FILE="$WORKDIR/variants.txt"
cat > "$VARIANTS_FILE" <<'EOF'
# comment line, and a blank line, both ignored

mysql_9.7 amd64 arm64
mysql_5.6 amd64
mariadb_11.8 amd64 arm64
mariadb_5.5 amd64
EOF

assert_eq "mysql_9.7_both mysql_5.6_amd64 mariadb_11.8_both mariadb_5.5_amd64" \
  "$("$VARIANTS" build-targets amd64)" "build-targets on amd64 covers every variant"
assert_eq "mysql_9.7_both mariadb_11.8_both" \
  "$("$VARIANTS" build-targets arm64)" "build-targets on arm64 omits the amd64-only variants"
assert_eq "mysql_9.7_amd64 mysql_9.7_arm64 mariadb_11.8_amd64 mariadb_11.8_arm64" \
  "$("$VARIANTS" single-arch-targets)" "single-arch-targets covers only multi-arch variants, both ways"
assert_eq "mysql_9.7 mariadb_11.8" \
  "$("$VARIANTS" multi-arch-variants)" "multi-arch-variants is what gets a combined manifest"
assert_eq "mysql_9.7_test mysql_5.6_test mariadb_11.8_test mariadb_5.5_test" \
  "$("$VARIANTS" test-targets amd64)" "test-targets on amd64 covers every variant"
assert_eq "mysql_9.7_test mariadb_11.8_test" \
  "$("$VARIANTS" test-targets arm64)" "test-targets on arm64 omits the amd64-only variants"
assert_eq "ddev-dbserver-mysql-9.7 ddev-dbserver-mysql-5.6 ddev-dbserver-mariadb-11.8 ddev-dbserver-mariadb-5.5" \
  "$("$VARIANTS" repos | tr '\n' ' ' | sed 's/ $//')" "repos names the Docker Hub repositories"
assert_eq "6" "$("$VARIANTS" json | jq 'length')" "json emits one entry per (variant, arch)"
assert_eq "amd64" "$("$VARIANTS" json | jq -r '[.[] | select(.dbtype=="mysql_5.6")] | .[].arch')" \
  "json gives an amd64-only variant exactly one entry"

if "$VARIANTS" bogus-view >/dev/null 2>&1; then
  fail "should reject an unknown view"
else
  pass "rejects an unknown view"
fi
if "$VARIANTS" build-targets >/dev/null 2>&1; then
  fail "should require a host arch for build-targets"
else
  pass "requires a host arch for build-targets"
fi

# --- Against the real variants.txt: the views have to agree with each other,
# which is what stops the Makefile, image-configs.sh, and the push workflow
# from disagreeing about what exists.
unset VARIANTS_FILE

real_build_amd64="$("$VARIANTS" build-targets amd64 | wc -w | tr -d '[:space:]')"
real_repos="$("$VARIANTS" repos | wc -l | tr -d '[:space:]')"
assert_eq "$real_build_amd64" "$real_repos" "every variant has a repo and an amd64 build target"

real_json="$("$VARIANTS" json | jq 'length')"
multi="$("$VARIANTS" multi-arch-variants | wc -w | tr -d '[:space:]')"
single_arch="$(( real_repos - multi ))"
assert_eq "$(( multi * 2 + single_arch ))" "$real_json" "the json matrix is 2 jobs per multi-arch variant plus 1 each for the rest"

assert_eq "$(( multi * 2 ))" "$("$VARIANTS" single-arch-targets | wc -w | tr -d '[:space:]')" \
  "single-arch-targets is two per multi-arch variant"

# The default variant `make` builds locally must be in the matrix, or
# image-configs.sh would mark a nonexistent variant as locally built.
if "$VARIANTS" repos | grep -qx "ddev-dbserver-mariadb-11.8"; then
  pass "the default variant image-configs.sh builds locally is in the matrix"
else
  fail "mariadb_11.8 (image-configs.sh's DDEV_DBSERVER_DEFAULT_TARGET) is missing from variants.txt"
fi

if [ "$FAILURES" -eq 0 ]; then
  echo "All db_variants_test.sh checks passed."
  exit 0
else
  echo "$FAILURES db_variants_test.sh check(s) failed." >&2
  exit 1
fi
