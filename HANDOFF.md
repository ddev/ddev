# HANDOFF: bringing up the perf/ nightly benchmark harness on Buildkite

Temporary note — not part of the ddev codebase, delete once resolved (same
convention as the last HANDOFF.md, removed in `23fdaaa47`). Context for
picking this back up in a different session/environment.

Branch: `20260726_rfay_perf_testing` (pushed to `upstream`, not yet merged).

## What this is

Bringing up `perf/` — a nightly performance-benchmark harness that runs the
same DDEV benchmark battery (`perf/run-benchmark.sh`) across 6 Buildkite
pipelines (one per Docker-provider/OS combination) plus 2 GitHub Actions
workflows, and publishes results to a dashboard under `/perf/` on the docs
site. See `perf/README.md` and `perf/collector/README.md` for the
architecture; this file is just "where are we right now."

## Status as of 2026-08-05: 4 of 6 pipelines fully passing

| Pipeline slug                          | Status |
|-----------------------------------------|--------|
| `ddev-perf-macos-docker-desktop-arm64`  | **Passing** (build #7, `macstadium-m1-1.local`) |
| `ddev-perf-macos-shared-providers`      | **Passing, all 5 legs** (`colima_vz`, `lima`, `orbstack`, `podman-rootless`, `rancher-desktop`, build #4, `tb-macos-arm64-5`) |
| `ddev-perf-wsl2-docker-desktop`         | **Passing** (build #6, `tb-wsldd-16`) |
| `ddev-perf-wsl2-mirrored`               | **Passing** (build #7, `tb-wsl-14`) |
| `ddev-perf-windows10-docker-desktop`    | Retrying (build #5) against the Windows `rm`-busy fix below; queued behind an unrelated job on the single `tb-win11-10` agent |
| `ddev-perf-wsl2-docker-inside`          | **Deprioritized** — no connected agent has ever matched its required tags (`os=wsl2` + `dockertype=wsl2`); `tb-wsl-12`/`-14` are tagged `os=wsl2-mirrored`, `tb-wsldd-16` is tagged `dockertype=dockerforwindows`. `wsl2-mirrored` is the canonical pipeline for "docker native inside WSL2" — stop retriggering this one. |

Getting here took 5 separate real bugs, each confirmed against a real
Buildkite run (not just guessed at):

1. **Puppeteer/Chrome silently failed to install** on Node.js 26 (macOS,
   confirmed also would have hit Windows). Root cause:
   [puppeteer/puppeteer#14957](https://github.com/puppeteer/puppeteer/issues/14957) —
   `@puppeteer/browsers`' pinned `extract-zip@2.0.1` dependency silently
   truncates zip extraction on Node 26 without raising an error (`npm ci`
   exits 0, but the browser executable is missing). **Not** a stale-cache
   problem — a clean `rm -rf ~/.cache/puppeteer` + fresh `npm ci` reproduces
   it every time. Fix: bumped `puppeteer` from `^24.11.2` to `^25.5.0`
   (commit `824261ab9`) — Puppeteer's own fix for #14957 dropped
   `extract-zip` for shelling out to system `unzip`, landing in
   `@puppeteer/browsers@3.0.2`. This also answered the long-standing "does
   headless Chrome even run under Buildkite's Windows service context"
   question: yes, once this bug is out of the way.
2. **Puppeteer 25 requires Node >=22 and dropped CommonJS support**, so
   `require('puppeteer')` threw `ERR_REQUIRE_ESM` on the WSL2/Linux testbots,
   which were still on Node.js v18.19.1. Not a code fix — genuinely needed
   Node upgraded on `tb-wsldd-16`/`tb-wsl-14`/`tb-wsl-12`. Done: upgraded to
   Node 22+ via the `nodesource` apt method already documented in
   `docs/content/developers/buildkite-testmachine-setup.md` (not the `n`
   version manager — that installs under a user's home dir, invisible to
   `buildkite-agent`'s systemd service since it never sources `.bashrc`; see
   `/etc/buildkite-agent/hooks/environment`, the same mechanism already used
   for `CAROOT`/`WSLENV`, if a similar problem shows up again).
3. **`docker_provider` reported `"unknown"`** for any run without `DOCKER_TYPE`
   set (i.e. every local run). Fix (commit `860b44190`): fall back to `ddev
   version -j`'s own `docker-platform` field instead of a hardcoded string.
4. **`drush` missing on the reused benchmark project** — `perf.sh` provisions
   `$PERF_PROJECT_DIR` once via `ddev composer create-project
   drupal/recommended-project`, which doesn't include `drush/drush`, so
   `05-drush-install.sh` could never succeed. Fix (commit `815b33501`): added
   a check-and-add step (`ddev drush --version` fails → `ddev composer
   require drush/drush`) that self-heals both fresh and already-provisioned
   projects.
5. **Windows `rm -rf` on the mutagen sync fixture failed with "Device or
   resource busy"** — a file just touched by Mutagen/Docker Desktop can
   still be briefly held open on Windows. Fix (commit `8e2466b5e`): tolerate
   it with `|| true`, same pattern `reset-drupal.sh` already used for this
   exact reason — it's a throwaway fixture dir recreated every run.

Also added along the way: `run-benchmark.sh` now prints a readable results
table (metric name + value + unit) to stderr, not just the raw JSON (commit
`dfbe46615`); `perf/collector/dashboard.html` got a "Compare environments"
bar-chart view alongside the existing trend lines (commit `dcb667851`).

## To resume

1. Check `ddev-perf-windows10-docker-desktop` build #5 — still in flight as
   of this update, queued on the single `tb-win11-10` agent. If it passes,
   that's 5/6 — `ddev-perf-wsl2-docker-inside` is the only one intentionally
   not being chased (see status table above).
2. Once everything passing is confirmed stable across a couple of retries,
   exercise the real end-to-end dashboard deploy before merging: `gh workflow
   run perf-linux.yml|perf-collect.yml|docs-publish.yml --ref
   20260726_rfay_perf_testing` (all three already support
   `workflow_dispatch`).
3. Update `perf/collector/pipelines.json` with the real pipeline slugs
   created below, if not already done — needed for `collect.js` to find
   these pipelines' artifacts.

## `bk` CLI cheat sheet

Assumes `bk` is already installed and authorized (`bk --help` works without
prompting). Org is `ddev` throughout.

```bash
# List pipelines / agents
bk pipeline list --output json
bk agent list --output json          # NOTE: no --org flag, unlike other subcommands

# Inspect a build (state, jobs, which agent ran which job)
bk api /pipelines/<slug>/builds/<n>
# bk api auto-prepends /organizations/ddev -- do NOT include it yourself,
# or you get a 404 from the doubled path.

# Trigger a manual build on a specific branch (needed here since all perf/
# work is on a feature branch, and branch filters are disabled on these
# pipelines -- a plain `bk build create` defaults to the pipeline's default
# branch and 404s looking for a .buildkite/*.yml that only exists on our branch)
#
# Use --branch alone, WITHOUT a pinned --commit: the build's `.commit` field
# shows the literal string "HEAD" until the pipeline-upload step resolves it
# against the branch tip server-side (normal, resolves within seconds) --
# pinning a commit yourself risks the local checkout's HEAD not actually
# being what's pushed, or not being fetchable if it isn't the branch tip.
bk build create --pipeline <slug> \
  --branch 20260726_rfay_perf_testing \
  --message "why you're retrying" \
  --ignore-branch-filters \
  --yes                              # skips the interactive Y/N prompt (needed non-interactively)

# Tail a specific job's raw log (get the job id from `bk api .../builds/<n>`)
bk job log <job-id>

# Create/manage pipelines and schedules directly via the REST API when the
# `bk pipeline`/`bk build` subcommands don't expose enough (e.g. full
# `configuration` YAML, or the Schedules API):
bk api --method POST /pipelines --data '<json>'      # --data wants a literal
                                                        # JSON string, NOT
                                                        # @file.json like curl
bk api --method POST /pipelines/<slug>/schedules --data '<json>'
```

Gotchas hit while doing this work:

- `bk pipeline create` can't set the full pipeline config (steps, agent
  tags, etc.) — had to go through `bk api POST /pipelines` directly instead.
- `bk api` prepends `/organizations/<org>` itself; passing it yourself
  doubles the path and 404s.
- `bk agent list` doesn't accept `--org` (unlike most other subcommands).
- `bk build create` refuses to run on a pipeline with branch filters
  disabled unless you pass `--ignore-branch-filters`.
- `--yes` is required to skip the confirmation prompt when running
  non-interactively (otherwise it EOFs waiting for input).
- Since `docker-desktop-arm64`'s 2-machine pool and `shared-providers`' 3-machine
  pool round-robin across machines, a pipeline can look "fixed" on one retry
  and "broken again" on the next just because a different machine in its pool
  picked up the job — always check `bk api /pipelines/<slug>/builds/<n>` for
  which agent actually ran the work job, not just whether it passed.
- A single-machine agent (e.g. `tb-win11-10`) can be busy running an unrelated
  pipeline's job when you trigger a perf build — it'll queue and run once free,
  not a bug.
