#!/usr/bin/env bash
# Times how long it takes a large, many-small-files tree to sync into the
# project and settle, using `ddev mutagen sync` (an explicit blocking flush)
# as the settle signal. This is deliberately independent of any CMS, since
# it isolates the file-sync mechanics that vary most across Docker providers
# (bind mount vs. Mutagen vs. NFS, etc.) from any particular app's install path.
# Prints one JSON metric line: {"metric":"mutagen_settle_ms","value_ms":N-or-null}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
FIXTURE_DIR="$DIR/../fixtures/large-tree"
TARGET_SUBDIR=".perf-fixture"

cd "$DDEV_PERF_PROJECT_DIR"

if ! ddev mutagen status >/dev/null 2>&1; then
  echo "Mutagen is not enabled on this project; skipping mutagen_settle_ms" >&2
  emit_metric "mutagen_settle_ms" "null"
  exit 0
fi

rm -rf "./$TARGET_SUBDIR"
mkdir -p "./$TARGET_SUBDIR"

start=$(now_ms)
cp -R "$FIXTURE_DIR/." "./$TARGET_SUBDIR/"
ddev mutagen sync >/dev/null
end=$(now_ms)

# Clean up and flush the deletion too, so the project is clean for the next metric.
rm -rf "./$TARGET_SUBDIR"
ddev mutagen sync >/dev/null

emit_metric "mutagen_settle_ms" "$(( end - start ))"
