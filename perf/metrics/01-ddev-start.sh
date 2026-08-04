#!/usr/bin/env bash
# Times a cold `ddev start`: power off, prune, then start-to-ready.
# Prints one JSON metric line: {"metric":"ddev_start_cold_ms","value_ms":N}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
REPEAT="${DDEV_PERF_REPEAT:-3}"

cd "$DDEV_PERF_PROJECT_DIR"

samples=()
for _ in $(seq 1 "$REPEAT"); do
  ddev poweroff >/dev/null 2>&1 || true
  docker system prune -f >/dev/null 2>&1 || true

  start=$(now_ms)
  ddev start -y >/dev/null 2>&1
  end=$(now_ms)

  samples+=("$(( end - start ))")
done

value=$(printf '%s\n' "${samples[@]}" | median)
emit_metric "ddev_start_cold_ms" "$value"
