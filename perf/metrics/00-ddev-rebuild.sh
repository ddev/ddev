#!/usr/bin/env bash
# Times `ddev utility rebuild`: a forced, no-cache rebuild of the project's
# web image (plus restart). This isolates image-build-layer regressions --
# a change to WriteBuildDockerfile, app_compose_template.yaml, or a base
# Dockerfile -- from ddev_start_cold_s, which starts from an already-built
# image and would not notice a build-time regression. See #8600, where a
# recursive chgrp/chmod added 90s+ to every project build without any
# existing metric or test noticing.
# Prints one JSON metric line: {"metric":"ddev_rebuild_s","value_s":N}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
# Deliberately not DDEV_PERF_REPEAT (3): a no-cache rebuild is expensive and
# each repeat busts its own image-layer cache, so repeating it doesn't average
# out noise the way repeating a plain start/install does -- it just triples
# the cost for no real benefit.
REPEAT=1

cd "$DDEV_PERF_PROJECT_DIR"

samples=()
for _ in $(seq 1 "$REPEAT"); do
  start=$(now_ms)
  run_quiet "ddev utility rebuild" ddev utility rebuild
  end=$(now_ms)

  samples+=("$(( end - start ))")
done

value=$(printf '%s\n' "${samples[@]}" | median)
emit_metric "ddev_rebuild_s" "$value"
