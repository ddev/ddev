# DDEV performance benchmark harness

Systematized replacement for the old manual [ddev-puppeteer](https://github.com/ddev/ddev-puppeteer)
script (e.g. used for the [2023 Docker performance blog post](https://ddev.com/blog/docker-performance-2023/)).
Runs the same benchmark battery, unattended, across every platform/Docker-provider
combination already covered by `.buildkite/*.yml` and `.github/workflows/test-reusable.yml`,
on a nightly schedule (see `.buildkite/perf-*.yml` and `.github/workflows/perf-linux.yml`),
and accumulates results so trends are visible over time instead of living only in a blog post.

## Metrics collected

One JSON result object per run per leg (platform + Docker provider combination):

```json
{
  "timestamp": "2026-07-27T03:00:12Z",
  "commit_sha": "abc1234...",
  "branch": "main",
  "ddev_version": "v1.24.0",
  "os": "darwin",
  "arch": "arm64",
  "docker_provider": "colima_vz",
  "metrics": {
    "ddev_rebuild_s": 33.8,
    "ddev_start_cold_s": 8.3,
    "mutagen_settle_s": 4.1,
    "drupal_install_s": 41.2,
    "drush_install_s": 22.9,
    "ddev_stop_s": 2.0
  }
}
```

| Metric | What it measures | Why it's here |
|---|---|---|
| `ddev_rebuild_s` | `ddev utility rebuild` (forced no-cache image build + restart) | Image-build-layer cost -- catches regressions like #8600, where a recursive chgrp/chmod added 90s+ to every project build without any existing metric noticing. `ddev_start_cold_s` below starts from an already-built image, so it can't see this class of regression |
| `ddev_start_cold_s` | `ddev poweroff` + prune, then `ddev start` to ready | Docker-provider/container startup cost, independent of any CMS |
| `mutagen_settle_s` | Copy a ~5000-file tree in, time `ddev mutagen sync` (blocking flush) to settle | Isolates file-sync mechanics (bind mount vs. Mutagen vs. NFS) from any app's install logic. `null` when Mutagen isn't enabled on the project |
| `drupal_install_s` | Puppeteer drives the Drupal `demo_umami` web install wizard end-to-end | The flagship "realistic app" metric — parallels normal browser-based DDEV usage, exercising the DDEV router, webserver, and PHP-FPM on every step |
| `drush_install_s` | `ddev drush si demo_umami -y` (CLI, non-interactive) | Cheap diagnostic, not a replacement for `drupal_install_s` — see below |
| `ddev_stop_s` | `ddev poweroff` | Teardown cost |

Each metric script repeats noisy operations `DDEV_PERF_REPEAT` times (default 3) and reports the median, in seconds to one decimal place.

### Why both `drupal_install_s` and `drush_install_s`?

Investigated directly in Drupal core (`web/core/includes/install.core.inc`) rather than assumed
equivalent. `ddev drush si` passes settings, so Drupal's installer runs non-interactively:
every batch step (module installs, config import, demo content creation) runs straight through
in one continuous PHP CLI process, and `ddev drush` itself execs PHP directly inside the web
container — it never touches the DDEV router, the webserver, or PHP-FPM.

The browser-driven install is interactive, so Drupal deliberately breaks each batch chunk into
its own HTTP request/page-reload. Every one of those round trips goes through the DDEV router,
the webserver, PHP-FPM, and a fresh Drupal bootstrap — exactly the layer where Docker-provider
differences (bind mount vs. Mutagen vs. NFS, gRPC-FUSE vs. virtiofs, etc.) most plausibly show
up, and exactly what normal browser-based DDEV usage exercises on every page load.

So `drush_install_s` isolates pure PHP-execution + filesystem-I/O time, while `drupal_install_s`
additionally captures router/webserver/PHP-FPM overhead. Keep both: if they track together across
providers, that confirms filesystem I/O dominates; if they diverge, the gap itself isolates
router/webserver overhead as the real differentiator.

## Running locally

```sh
export DDEV_PERF_PROJECT_DIR=~/workspace/d11   # a standard DDEV Drupal 11 project, already `ddev start`-ed once
export DDEV_PERF_SITE_URL=https://d11.ddev.site/
./perf/run-benchmark.sh > result.json
```

Requires `jq` and Node.js (for the Puppeteer step) on `PATH`, in addition to the usual DDEV/Docker
prerequisites. The large-file fixture tree used by `mutagen_settle_s` is generated on first use
under `perf/fixtures/large-tree/` (gitignored) and reused on subsequent runs.

## CI wiring

- **macOS**: `.buildkite/perf-macos-docker-desktop-arm64.yml` is its own pipeline, since Docker
  Desktop has dedicated machines (`macstadium-m1`, `tb-macos-arm64-4`). The other five macOS
  providers (colima, lima, orbstack, podman-rootless, rancher-desktop) currently share a single
  3-machine pool (`tb-macos-arm64-5/6/7`), so they're combined into one pipeline instead of five:
  `.buildkite/perf-macos-shared-providers.yml` runs all five as sequential steps (`wait` between
  each), guaranteeing only one is ever active against that pool at a time — five independent
  nightly pipelines could otherwise grab up to three of the five providers at once and starve the
  pool for correctness tests during the run.
- **Windows/WSL2**: `.buildkite/perf-windows10dockerforwindows.yml` and `.buildkite/perf-wsl2-*.yml`,
  one per existing leg in `.buildkite/*.yml`, reusing the same `agents:` tags.
- All of the above must be configured in the Buildkite dashboard as pipelines with a **nightly
  Scheduled Build** (not a push/PR trigger) — that step can't be done from this repo; see the
  comment at the top of each `perf-*.yml` file.
- **Linux**: `.github/workflows/perf-linux.yml`, triggered by `schedule:` (nightly cron) and
  `workflow_dispatch:` (manual runs).
- Each leg uploads its single JSON result as a build artifact. `perf/collector/collect.sh`
  (run by a separate scheduled workflow) pulls all of a run's artifacts and appends one line per
  leg to a versioned `history.ndjson` dataset. `perf/collector/dashboard.html` renders trend lines
  from that dataset — see `perf/collector/README.md`.

## Retiring `ddev-puppeteer`

Once this harness is validated in CI, the [ddev-puppeteer](https://github.com/ddev/ddev-puppeteer)
repo should be archived with a README pointing here.
