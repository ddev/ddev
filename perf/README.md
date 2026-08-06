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
| `ddev_rebuild_s` | `ddev utility rebuild` (forced no-cache image build + restart), single sample | Image-build-layer cost -- catches regressions like #8600, where a recursive chgrp/chmod added 90s+ to every project build without any existing metric noticing. `ddev_start_cold_s` below starts from an already-built image, so it can't see this class of regression |
| `ddev_start_cold_s` | `ddev poweroff`, then `ddev start` to ready | Docker-provider/container startup cost, independent of any CMS |
| `mutagen_settle_s` | Copy a ~5000-file tree in, time `ddev mutagen sync` (blocking flush) to settle | Isolates file-sync mechanics (bind mount vs. Mutagen vs. NFS) from any app's install logic. `null` when Mutagen isn't enabled on the project |
| `drupal_install_s` | Puppeteer drives the Drupal `demo_umami` web install wizard end-to-end | The flagship "realistic app" metric — parallels normal browser-based DDEV usage, exercising the DDEV router, webserver, and PHP-FPM on every step |
| `drush_install_s` | `ddev drush si demo_umami -y` (CLI, non-interactive) | Cheap diagnostic, not a replacement for `drupal_install_s` — see below |
| `ddev_stop_s` | `ddev poweroff` | Teardown cost |

Each metric script repeats noisy operations `DDEV_PERF_REPEAT` times (default 3) and reports the
median, in seconds to one decimal place. `ddev_rebuild_s` is the exception: it always runs once --
a no-cache rebuild is expensive, and each repeat busts its own image-layer cache, so repeating it
doesn't average out noise the way repeating a plain start or install does.

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
./perf/run-benchmark.sh \
  --project-dir ~/workspace/d11 \
  --site-url https://d11.ddev.site/ \
  > result.json
```

`~/workspace/d11` is a standard DDEV Drupal 11 project, already `ddev start`-ed once.
`--project-dir`/`-d` and `--site-url`/`-u` can also be set via the `DDEV_PERF_PROJECT_DIR` and
`DDEV_PERF_SITE_URL` env vars instead; run `./perf/run-benchmark.sh --help` for all options.

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
- Each leg uploads its JSON result as a build artifact named `perf-result-<DOCKER_TYPE>.json`
  (`perf-macos-shared-providers.yml` uploads several, one per job, within a single build).
  `perf/collector/collect.js` (run by `.github/workflows/perf-collect.yml` on a schedule) pulls
  every matching artifact and appends one line per leg to a versioned `history.ndjson` dataset on
  the `performance-data` branch. `perf/collector/dashboard.html` renders trend lines from that
  dataset, and is published under `/perf/` on the docs site by `.github/workflows/docs-publish.yml`
  — see `perf/collector/README.md`.

## Reading the numbers: what is and isn't comparable

- **Across legs, for the same metric**: this is the intended comparison, and the "Compare
  environments" view exists for it. It deliberately doesn't normalize for the fact that legs run on
  different machines — Docker-provider/platform overhead *is* the thing being measured.
- **The Linux leg is not directly comparable to the Buildkite legs.** GitHub-hosted runners are
  ephemeral, so `perf-linux.yml` provisions the Drupal codebase from scratch every night and starts
  with cold image, Composer, and OS page caches. Every Buildkite leg reuses a persistent codebase
  and warm caches. Trends *within* the Linux leg are meaningful; a Linux-vs-macOS bar comparison
  mostly measures that difference.
- **Over time, within one leg**: meaningful, with the caveat that the round-robin pools
  (`tb-macos-arm64-5/6/7`) can land a given provider on a different physical machine each night.

## Relationship to `perf-start-time.yml`

`.github/workflows/perf-start-time.yml` predates this harness and overlaps it: it also measures
`ddev start` and `ddev utility rebuild` on Linux nightly, and exists for the same regression
(#8600) that `ddev_rebuild_s` cites. The two are kept separate on purpose and are not redundant:

- `perf-start-time.yml` is a **gate**: it also runs on PRs touching the build-layer files, compares
  against a baseline ref in the same job on the same machine, and is meant to fail.
- This harness is a **trend record**: nightly, cross-provider, absolute numbers, no pass/fail.

Neither supersedes the other. If `ddev_rebuild_s`/`ddev_start_cold_s` here ever become a CI gate,
revisit — at that point one of them should go.

## Assumption: one Buildkite agent per testbot

The `perf-*.yml` pipelines reuse the same `agents:` tags as the correctness pipelines, and
`01-ddev-start.sh` runs `ddev poweroff` before each timed sample -- which stops every DDEV project
on the machine, not just this one. That is only safe because each testbot machine runs a single
`buildkite-agent` slot, so a perf job and a test job are never in flight on the same machine at once. `.buildkite/test.sh` already relies on
the same exclusivity (it does `ddev poweroff` and `docker rm -f` against every container on the
machine), so this adds no new assumption — but if a testbot is ever given more than one agent slot,
both suites break, not just this one. `perf-macos-shared-providers.yml` handles the related but
distinct problem of one nightly run monopolizing a shared *pool*; see the comment at its top.

## Retiring `ddev-puppeteer`

Once this harness is validated in CI, the [ddev-puppeteer](https://github.com/ddev/ddev-puppeteer)
repo should be archived with a README pointing here.
