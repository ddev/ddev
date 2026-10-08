#!/usr/bin/env bash
# Prints the `go test -run` pattern for the tests that call the testcommon
# GOTEST_SHORT helpers, which the all-project-types job runs with GOTEST_SHORT
# unset. Fails on a test that reads GOTEST_SHORT itself, which would be missed.
# With `pkg` or `cmd`, prints one of the two halves the job is split into: the
# tests under that directory, with the ones in MOVE_TO_CMD moved over to even
# them out. The pattern ends in \b rather than $, which make would expand.
set -euo pipefail
cd "$(dirname "$0")/../.."

# They take about 40 and 13 of the 120 minutes the tests under pkg/ take.
MOVE_TO_CMD="TestDdevAllDatabases TestHttpsRedirection"

tests=$(find pkg cmd -name '*_test.go' -print0 | xargs -0 awk '
FNR == 1 { fn = ""; dir = substr(FILENAME, 1, index(FILENAME, "/") - 1) }
/^func / { fn = $0; sub(/^func (\([^)]*\) )?/, "", fn); sub(/\(.*/, "", fn) }
/testcommon\.(SkipIfGotestShort|IsGotestShort|UsesAllTestSites)\(/ {
	if (fn ~ /^Test/) print dir, fn
	else { printf "%s:%d: call this from the Test function itself\n", FILENAME, FNR > "/dev/stderr"; bad = 1 }
}
/os\.Getenv\("GOTEST_SHORT"\)/ && fn != "TestMain" {
	printf "%s:%d: use testcommon.SkipIfGotestShort or IsGotestShort instead\n", FILENAME, FNR > "/dev/stderr"; bad = 1
}
END { exit bad }
')

case "${1:-}" in
"") ;;
pkg) tests=$(awk -v m=" $MOVE_TO_CMD " '$1 == "pkg" && !index(m, " " $2 " ")' <<<"$tests") ;;
cmd) tests=$(awk -v m=" $MOVE_TO_CMD " '$1 == "cmd" || index(m, " " $2 " ")' <<<"$tests") ;;
*)
	echo "usage: $0 [pkg|cmd]" >&2
	exit 2
	;;
esac

awk '{ print $2 }' <<<"$tests" | sort -u | paste -sd'|' | sed 's/.*/^(&)\\b/'
