#!/usr/bin/env bash
# Resets a Drupal project to a pre-install state so install-time metrics start
# from the same place every run: drops the db, clears public files, drops the
# PHP opcache, and (if enabled) flushes Mutagen so those changes are visible
# in the container before the next metric runs.
#
# Deliberately does NOT remove settings.php: DDEV only regenerates that file
# (with the required include of settings.ddev.php, which is what wires in the
# DB credentials) during `ddev start`/`restart`, not on every drush/install
# invocation. Removing it here would break site:install's ability to find the
# database without a full project restart -- keep it, same as the original
# ddev-puppeteer.js, which only ever dropped the db and cleared files.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=./common.sh
source "$DIR/common.sh"

: "${DDEV_PERF_PROJECT_DIR:?DDEV_PERF_PROJECT_DIR must be set}"
cd "$DDEV_PERF_PROJECT_DIR"

run_quiet "ddev mysql reset" ddev mysql -e "DROP DATABASE IF EXISTS db; CREATE DATABASE db;"
rm -rf web/sites/default/files/* 2>/dev/null || true
ddev exec "killall -USR2 php-fpm" >/dev/null 2>&1 || true

if mutagen_enabled_for_project; then
  run_quiet "ddev mutagen sync (reset)" ddev mutagen sync
fi
