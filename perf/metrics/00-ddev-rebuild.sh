#!/usr/bin/env bash
# Times `ddev utility rebuild`: a forced, no-cache rebuild of the project's
# web image (plus restart). This isolates image-build-layer regressions --
# a change to WriteBuildDockerfile, app_compose_template.yaml, or a base
# Dockerfile -- from ddev_start_cold_ms, which starts from an already-built
# image and would not notice a build-time regression. See #8600, where a
# recursive chgrp/chmod added 90s+ to every project build without any
# existing metric or test noticing.
# Prints one JSON metric line: {"metric":"ddev_rebuild_ms","value_ms":N}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
REPEAT="${DDEV_PERF_REPEAT:-3}"

cd "$DDEV_PERF_PROJECT_DIR"

samples=()
for _ in $(seq 1 "$REPEAT"); do
  start=$(now_ms)
  ddev utility rebuild >/dev/null 2>&1
  end=$(now_ms)

  samples+=("$(( end - start ))")
done

value=$(printf '%s\n' "${samples[@]}" | median)
emit_metric "ddev_rebuild_ms" "$value"
