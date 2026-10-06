#!/usr/bin/env bash
# Prints the `go test -run` pattern for the tests that call the testcommon
# GOTEST_SHORT helpers, which the all-project-types job runs with GOTEST_SHORT
# unset. Fails on a test that reads GOTEST_SHORT itself, which would be missed.
# The pattern ends in \b rather than $, which make would expand.
set -euo pipefail
cd "$(dirname "$0")/../.."

find pkg cmd -name '*_test.go' -print0 | xargs -0 awk '
FNR == 1 { fn = "" }
/^func / { fn = $0; sub(/^func (\([^)]*\) )?/, "", fn); sub(/\(.*/, "", fn) }
/testcommon\.(SkipIfGotestShort|IsGotestShort|UsesAllTestSites)\(/ {
	if (fn ~ /^Test/) print fn
	else { printf "%s:%d: call this from the Test function itself\n", FILENAME, FNR > "/dev/stderr"; bad = 1 }
}
/os\.Getenv\("GOTEST_SHORT"\)/ && fn != "TestMain" {
	printf "%s:%d: use testcommon.SkipIfGotestShort or IsGotestShort instead\n", FILENAME, FNR > "/dev/stderr"; bad = 1
}
END { exit bad }
' | sort -u | paste -sd'|' | sed 's/.*/^(&)\\b/'
