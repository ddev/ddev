#!/usr/bin/env bash
# Times `ddev poweroff`. Prints one JSON metric line:
# {"metric":"ddev_stop_ms","value_ms":N}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
cd "$DDEV_PERF_PROJECT_DIR"

start=$(now_ms)
ddev poweroff >/dev/null
end=$(now_ms)

emit_metric "ddev_stop_ms" "$(( end - start ))"
