#!/usr/bin/env bash
# Cheap CLI-only diagnostic alongside the Puppeteer web install (see perf/README.md
# for why this isn't a replacement for it): `ddev drush si` runs entirely as a
# single non-interactive PHP CLI process inside the web container and never
# touches the DDEV router, webserver, or PHP-FPM, so it isolates PHP-execution +
# filesystem-I/O time from the router/webserver overhead the web install adds.
# Prints one JSON metric line: {"metric":"drush_install_ms","value_ms":N}
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../lib/common.sh
source "$DIR/../lib/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"

bash "$DIR/../lib/reset-drupal.sh"

cd "$DDEV_PERF_PROJECT_DIR"

start=$(now_ms)
ddev drush si demo_umami -y >/dev/null 2>&1
end=$(now_ms)

emit_metric "drush_install_ms" "$(( end - start ))"
