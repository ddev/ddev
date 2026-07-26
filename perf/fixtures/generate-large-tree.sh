#!/usr/bin/env bash
# Generates perf/fixtures/large-tree/ on demand: a representative many-small-files
# tree (like a vendor/ or node_modules/ directory) used by 02-mutagen-sync-settle.sh.
# Not committed to git (see .gitignore) -- regenerated the first time it's needed.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$DIR/large-tree"
FILE_COUNT="${PERF_FIXTURE_FILE_COUNT:-5000}"
DIR_COUNT="${PERF_FIXTURE_DIR_COUNT:-50}"

existing_count=0
if [ -d "$TARGET" ]; then
  existing_count=$(find "$TARGET" -type f | wc -l | tr -d ' ')
fi
if [ "$existing_count" -ge "$FILE_COUNT" ]; then
  echo "Fixture tree already present at $TARGET ($existing_count files)"
  exit 0
fi

rm -rf "$TARGET"
mkdir -p "$TARGET"

files_per_dir=$(( FILE_COUNT / DIR_COUNT ))
for d in $(seq 1 "$DIR_COUNT"); do
  subdir="$TARGET/dir_$d"
  mkdir -p "$subdir"
  for f in $(seq 1 "$files_per_dir"); do
    printf '<?php // perf fixture file %s/%s\n' "$d" "$f" > "$subdir/file_$f.php"
  done
done

echo "Generated $(find "$TARGET" -type f | wc -l) files under $TARGET"
