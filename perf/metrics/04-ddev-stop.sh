#!/usr/bin/env bash
# Times `ddev poweroff`. Prints one JSON metric line:
# {"metric":"ddev_stop_s","value_s":N}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
cd "$DDEV_PERF_PROJECT_DIR"

start=$(now_ms)
run_quiet "ddev poweroff" ddev poweroff
end=$(now_ms)

emit_metric "ddev_stop_s" "$(( end - start ))"
