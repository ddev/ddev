#!/usr/bin/env bash
# Orchestrates the DDEV performance benchmark battery against an already-running
# DDEV project and prints one JSON result line to stdout. See perf/README.md for
# the full schema and how to run this locally.
#
# Required (flag or env):
#   -d, --project-dir DIR / DDEV_PERF_PROJECT_DIR - path to the DDEV project to benchmark against
#   -u, --site-url URL    / DDEV_PERF_SITE_URL    - the project's URL, e.g. https://d11.ddev.site/
#                                                    (used for the Puppeteer-driven Drupal install metric)
# Optional env:
#   DDEV_PERF_REPEAT      - repeat count for noisy metrics, median is reported (default 3)
#   DOCKER_TYPE           - docker provider label; set by CI, e.g. "colima_vz" (default: ddev's own
#                                                    "docker-platform", e.g. "orbstack", "docker-desktop")
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
  cat <<EOF
Usage: $(basename "${BASH_SOURCE[0]}") [options]

Orchestrates the DDEV performance benchmark battery against an already-running
DDEV project and prints one JSON result line to stdout. See perf/README.md for
the full schema.

Options:
  -d, --project-dir DIR  Path to the DDEV project to benchmark against
                         (or set DDEV_PERF_PROJECT_DIR)
  -u, --site-url URL     The project's URL, e.g. https://d11.ddev.site/, used
                         for the Puppeteer-driven Drupal install metric
                         (or set DDEV_PERF_SITE_URL)
  -h, --help             Show this help and exit

Optional env:
  DDEV_PERF_REPEAT   Repeat count for noisy metrics, median is reported (default 3)
  DOCKER_TYPE        Docker provider label; set by CI, e.g. "colima_vz" (default: ddev's own
                     "docker-platform", e.g. "orbstack", "docker-desktop")
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    -d|--project-dir)
      if [ $# -lt 2 ]; then
        echo "FATAL: $1 requires a value" >&2
        exit 1
      fi
      DDEV_PERF_PROJECT_DIR="$2"
      shift 2
      ;;
    --project-dir=*)
      DDEV_PERF_PROJECT_DIR="${1#*=}"
      shift
      ;;
    -u|--site-url)
      if [ $# -lt 2 ]; then
        echo "FATAL: $1 requires a value" >&2
        exit 1
      fi
      DDEV_PERF_SITE_URL="$2"
      shift 2
      ;;
    --site-url=*)
      DDEV_PERF_SITE_URL="${1#*=}"
      shift
      ;;
    *)
      echo "FATAL: unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

missing=()
if [ -z "${DDEV_PERF_PROJECT_DIR:-}" ]; then
  missing+=("DDEV_PERF_PROJECT_DIR (or --project-dir/-d)")
fi
if [ -z "${DDEV_PERF_SITE_URL:-}" ]; then
  missing+=("DDEV_PERF_SITE_URL (or --site-url/-u)")
fi
if [ "${#missing[@]}" -gt 0 ]; then
  {
    echo "FATAL: missing required value(s):"
    for m in "${missing[@]}"; do
      echo "  - $m"
    done
    echo
    usage
  } >&2
  exit 1
fi

# shellcheck source=lib/common.sh
source "$DIR/lib/common.sh"

export DDEV_PERF_PROJECT_DIR DDEV_PERF_SITE_URL
export DDEV_PERF_REPEAT="${DDEV_PERF_REPEAT:-3}"

metrics_json='{}'
add_metric() {
  local line="$1"
  if ! jq -e . >/dev/null 2>&1 <<<"$line"; then
    echo "FATAL: expected a JSON metric line but got: '$line'" >&2
    exit 1
  fi
  metrics_json=$(jq --argjson m "$line" '. + {($m.metric): $m.value_s}' <<<"$metrics_json")
}

echo "--- Generating sync-settle fixture tree (if needed)" >&2
bash "$DIR/fixtures/generate-large-tree.sh" >&2

echo "--- ddev utility rebuild (no-cache image build)" >&2
add_metric "$(bash "$DIR/metrics/00-ddev-rebuild.sh")"

echo "--- ddev start (cold)" >&2
add_metric "$(bash "$DIR/metrics/01-ddev-start.sh")"

echo "--- Mutagen sync settle" >&2
add_metric "$(bash "$DIR/metrics/02-mutagen-sync-settle.sh")"

echo "--- Drupal demo_umami web install (Puppeteer)" >&2
bash "$DIR/lib/reset-drupal.sh"
( cd "$DIR/metrics/03-drupal-install" && npm ci --no-progress )
add_metric "$(cd "$DIR/metrics/03-drupal-install" && DDEV_PERF_SITE_URL="$DDEV_PERF_SITE_URL" node install-timer.js)"

echo "--- drush si demo_umami (CLI diagnostic)" >&2
add_metric "$(bash "$DIR/metrics/05-drush-install.sh")"

echo "--- ddev stop" >&2
add_metric "$(bash "$DIR/metrics/04-ddev-stop.sh")"

echo "--- Assembling result" >&2
version_json=$(cd "$DDEV_PERF_PROJECT_DIR" && ddev version -j 2>/dev/null || echo '{}')
ddev_version=$(jq -r '.raw["DDEV version"] // "unknown"' <<<"$version_json")
os_name=$(jq -r '.raw.os // "unknown"' <<<"$version_json")
arch_name=$(jq -r '.raw.architecture // "unknown"' <<<"$version_json")

commit_sha="${BUILDKITE_COMMIT:-${GITHUB_SHA:-$(git -C "$DIR" rev-parse HEAD 2>/dev/null || echo unknown)}}"
branch="${BUILDKITE_BRANCH:-${GITHUB_REF_NAME:-$(git -C "$DIR" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)}}"
# DOCKER_TYPE (CI) carries more detail than ddev can see for itself, e.g.
# "colima_vz" vs "colima_qemu" or "podman-rootless" -- prefer it when set.
# Falling back to ddev's own "docker-platform" (e.g. "orbstack", "docker-desktop")
# instead of a bare "unknown" lets local runs report a real provider too.
docker_provider="${DOCKER_TYPE:-$(jq -r '.raw["docker-platform"] // "unknown"' <<<"$version_json")}"

jq -n \
  --argjson metrics "$metrics_json" \
  --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg commit_sha "$commit_sha" \
  --arg branch "$branch" \
  --arg ddev_version "$ddev_version" \
  --arg os "$os_name" \
  --arg arch "$arch_name" \
  --arg docker_provider "$docker_provider" \
  '{
    timestamp: $timestamp,
    commit_sha: $commit_sha,
    branch: $branch,
    ddev_version: $ddev_version,
    os: $os,
    arch: $arch,
    docker_provider: $docker_provider,
    metrics: $metrics
  }'
