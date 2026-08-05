#!/usr/bin/env bash
# Times how long it takes a large, many-small-files tree to sync into the
# project and settle, using `ddev mutagen sync` (an explicit blocking flush)
# as the settle signal. This is deliberately independent of any CMS, since
# it isolates the file-sync mechanics that vary most across Docker providers
# (bind mount vs. Mutagen vs. NFS, etc.) from any particular app's install path.
# Prints one JSON metric line: {"metric":"mutagen_settle_s","value_s":N-or-null}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
FIXTURE_DIR="$DIR/../fixtures/large-tree"
TARGET_SUBDIR=".perf-fixture"

cd "$DDEV_PERF_PROJECT_DIR"

if ! ddev mutagen status >/dev/null 2>&1; then
  echo "Mutagen is not enabled on this project; skipping mutagen_settle_s" >&2
  emit_metric "mutagen_settle_s" "null"
  exit 0
fi

# On Windows, a file just touched by Mutagen/Docker Desktop can still be briefly
# held open, so `rm -rf` here can fail with "Device or resource busy" -- tolerate
# it, same as reset-drupal.sh does for the same reason: this is a throwaway
# fixture dir recreated (or re-cleaned) every run, so a leftover file or two
# doesn't affect correctness, only next run's `cp -R`/cleanup working a bit harder.
rm -rf "./$TARGET_SUBDIR" || true
mkdir -p "./$TARGET_SUBDIR"

start=$(now_ms)
cp -R "$FIXTURE_DIR/." "./$TARGET_SUBDIR/"
run_quiet "ddev mutagen sync" ddev mutagen sync
end=$(now_ms)

# Clean up and flush the deletion too, so the project is clean for the next metric.
rm -rf "./$TARGET_SUBDIR" || true
run_quiet "ddev mutagen sync (cleanup)" ddev mutagen sync

emit_metric "mutagen_settle_s" "$(( end - start ))"
