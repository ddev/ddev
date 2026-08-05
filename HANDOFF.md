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

## Code-side work: done

- Metrics report seconds, not ms; bash-3/macOS compatible.
- `run-benchmark.sh` has `--help`, `--project-dir`, `--site-url` flags with
  all-missing-values-at-once error reporting.
- Fixed `collect.js` silently dropping 4 of 5 results from the
  shared-providers leg (was matching only the first `perf-result.json`
  artifact; now collects every `perf-result-*.json` per build).
- Fixed the GitHub Pages collision: `perf-collect.yml` no longer deploys to
  Pages itself — it commits to the `performance-data` branch and dispatches
  `docs-publish.yml`, which folds the dashboard into `/perf/` on the one real
  Pages deploy.
- `BUILDKITE_API_TOKEN` moved from a plain repo secret to the `test-secrets`
  1Password vault, matching the rest of the repo's workflows.
- **Just fixed (commit `e0f17748a`)**: several metric scripts redirected
  their core timed `ddev` command straight to `/dev/null 2>&1`, which under
  `set -euo pipefail` meant any real failure exited with **zero diagnostic
  output** — this is what made the `05-drush-install.sh` failure on
  `wsl2-mirrored` build #4 undiagnosable. Added `run_quiet()` to
  `perf/lib/common.sh` (silent on success, prints captured output on
  failure) and applied it to the actual measured commands in
  `00-ddev-rebuild.sh`, `01-ddev-start.sh`, `02-mutagen-sync-settle.sh`,
  `04-ddev-stop.sh`, `05-drush-install.sh`, and `reset-drupal.sh`. Not yet
  exercised by a real failure since landing it — worth watching the next
  failure to confirm it actually surfaces something useful.
- `docs/content/developers/buildkite-testmachine-setup.md` documents the
  Node.js + Chrome-runtime-shared-lib apt packages WSL2/Linux testbots need
  (committed `2c63f3ae2`).
- **Fixed, committed `5be828481`/`824261ab9`**: bumped `puppeteer` in
  `perf/metrics/03-drupal-install/package.json` from `^24.11.2` to `^25.5.0`
  and regenerated `package-lock.json` — root-causes the macOS Puppeteer
  blocker below (upstream bug
  [#14957](https://github.com/puppeteer/puppeteer/issues/14957): unmaintained
  `extract-zip@2.0.1` silently truncates browser extraction on Node.js 26).
  **Confirmed working on real Buildkite runs** (2026-08-05 retrigger, see
  "Retrigger results" below) on both macOS (`macstadium-m1-1.local`) and
  Windows (`tb-win11-10`) — Puppeteer/Chrome now installs and runs cleanly on
  both. **But introduced a regression on the Node 18 WSL2/Linux testbots**:
  Puppeteer 25 is ESM-only and requires Node >=22, so `require('puppeteer')`
  now throws `ERR_REQUIRE_ESM` there. See "The macOS blocker in detail" and
  "Retrigger results" for the full writeup — fix is to upgrade Node.js on
  those testbots to 22+, which `docs/content/developers/buildkite-testmachine-setup.md`
  already documents how to do; not yet applied to the actual machines.
- **Fixed, committed `860b44190`**: `run-benchmark.sh`'s `docker_provider`
  field silently reported the literal string `"unknown"` for any run that
  didn't have `DOCKER_TYPE` set — true for every local run, since only CI
  sets it. Now falls back to `ddev version -j`'s own `docker-platform` field
  (e.g. `"orbstack"`, `"docker-desktop"`) instead. `DOCKER_TYPE` still wins
  when set, since CI's label carries detail ddev can't see for itself (e.g.
  `"colima_vz"` vs `"colima_qemu"`).

## Buildkite side: 6 pipelines created, all currently red

All created via `bk api --method POST /pipelines` (not `bk pipeline create`
— see gotchas below), `build_branches: false` (branch filters disabled) so
they don't fire on every push while this is still on a feature branch.

| Pipeline slug                          | Machine(s) behind it                                              | Status as of 2026-08-05 retrigger (commit `824261ab9`) |
|-----------------------------------------|---------------------------------------------------------------------|--------|
| `ddev-perf-macos-docker-desktop-arm64`  | `tb-macos-arm64-4`, `macstadium-m1-1.local` (2-machine pool)        | **Puppeteer fix confirmed working** (`macstadium-m1-1.local`, build #6) — Chrome/chrome-headless-shell installed and ran cleanly. Now fails later, at `drush si`: `drush is not available` — see "New bug: missing drush" below. |
| `ddev-perf-macos-shared-providers`      | `tb-macos-arm64-5`, `-6`, `-7` (3-machine pool, 5 sequential legs)   | Build #3: `colima (vz)`, `lima`, and `orbstack` legs all failed on `tb-macos-arm64-7` with `start VM: timed out waiting for VM to start` — orbstack itself won't start a VM on that machine (used as the shared reset-to-known-state step between legs, so it blocks every leg that runs there). Infra flakiness on that specific machine, unrelated to Puppeteer. `podman rootless`/`rancher desktop` legs untested this run (never reached, or ran on a different machine in the pool). |
| `ddev-perf-windows10-docker-desktop`    | `tb-win11-10` (single machine)                                       | **The "real open question" is answered: yes** — headless Chrome runs fine under Buildkite's Windows service context once Puppeteer's extraction bug is fixed. Build #3: Puppeteer/Chrome step passed cleanly; failed later at `drush si` with the same missing-drush bug as macOS. |
| `ddev-perf-wsl2-docker-desktop`         | `tb-wsldd-16` (single machine)                                       | **New regression from the puppeteer bump**: this machine runs Node.js v18.19.1; Puppeteer 25 is ESM-only and requires Node >=22, so `require('puppeteer')` throws `ERR_REQUIRE_ESM` in `install-timer.js`. Needs Node upgraded per `docs/content/developers/buildkite-testmachine-setup.md` (already documents `nodesource` setup for Node 22). |
| `ddev-perf-wsl2-docker-inside`          | shares the `wsl2-mirrored` pool tags                                 | Still stuck "scheduled", no free agent, across two separate trigger attempts (build #2 from earlier in the day, build #3 from this retrigger) — genuine pool-capacity issue, not something a retrigger fixes. |
| `ddev-perf-wsl2-mirrored`               | `tb-wsl-12`, `tb-wsl-14`                                              | Same Node 18 `ERR_REQUIRE_ESM` regression as `wsl2-docker-desktop` (ran on `tb-wsl-14`). The earlier `05-drush-install.sh` silent-failure bug is fixed (`run_quiet` did its job — this failure now shows a clear Node.js stack trace instead of nothing), but never got far enough this run to hit that step again. |

### New bug found this run: missing `drush` on the reused benchmark project

Both machines that got past the Puppeteer step (`macstadium-m1-1.local`,
`tb-win11-10`) now fail at the `drush si demo_umami` step with `drush is not
available. You may need to 'ddev composer require drush/drush'`. `perf.sh`
provisions `$PERF_PROJECT_DIR` once via `ddev composer create-project
drupal/recommended-project` and reuses it on every subsequent nightly run —
that project template doesn't include `drush/drush`, so the CLI-diagnostic
metric (`05-drush-install.sh`) can never succeed as provisioned. Needs either
`ddev composer require drush/drush` added to `perf.sh`'s one-time
provisioning step, or added to the reused project by hand on each affected
machine. Not yet fixed.

### The macOS blocker in detail — root-caused and fixed (2026-08-05)

Reproduced locally on a macOS arm64 checkout: `rm -rf ~/.cache/puppeteer` +
`npm ci` alone does NOT fix this — cache-cleared and re-downloaded from
scratch, the exact same broken state came back immediately. This is a known
upstream bug, not stale/leftover machine state:
[puppeteer/puppeteer#14957](https://github.com/puppeteer/puppeteer/issues/14957) —
`@puppeteer/browsers`'s (at the version we had pinned) transitive dependency
`extract-zip@2.0.1` (unmaintained since 2020) silently aborts mid-extraction
on Node.js 26 without raising an error. The zip downloads fine (verified with
`unzip -l` — all entries present, valid archive); only the small metadata
files (`ABOUT`, `LICENSE.*`) get extracted, the large binaries
(`chrome-headless-shell`, and `Google Chrome for Testing Framework` inside
the full Chrome bundle) are silently dropped, and `npm ci` exits 0. Symptom
is one of:

```
dlopen ... Google Chrome for Testing Framework ... (no such file)
```

or

```
npm error Error: ERROR: Failed to set up chrome-headless-shell v148.0.7778.97!
npm error   [cause]: Error: All providers failed for chrome-headless-shell ...
npm error     - DefaultProvider: The browser folder (...) exists but the
npm error       executable (...) is missing
```

**Fix (applied, commit `5be828481`/`824261ab9`): bump the `puppeteer` dependency.**
Puppeteer's own fix for #14957 was to drop `extract-zip` entirely and shell
out to the system `unzip` — that landed in `@puppeteer/browsers@3.0.2`, first
pulled in by `puppeteer@25.0.2`. `perf/metrics/03-drupal-install/package.json`
was pinned to `puppeteer: "^24.11.2"`, stuck on the broken `2.13.x`
`@puppeteer/browsers` line. Bumped to `^25.5.0` (latest at the time) and
regenerated `package-lock.json`. `install-timer.js`'s `require('puppeteer')`
still works unchanged against Puppeteer 25's ESM-only package, since Node
22+ natively supports `require(esm)` for synchronous ESM modules — no code
changes needed there.

Verified locally, 3x from a fully clean `~/.cache/puppeteer` + fresh
`node_modules` (matching exactly what CI does): `npm ci` now extracts both
`chrome` and `chrome-headless-shell` completely every time, and the full
`drupal_install_s` metric runs clean end-to-end. **No manual `unzip` repair,
no maintenance-script changes, no per-machine SSH work needed** — this was a
plain outdated-dependency bug, already fixed upstream.

Given the version bump is the real fix, the 5 macOS machines almost certainly
don't need individual repair once this branch's `package-lock.json` update is
in place — the next `npm ci` on each will just pull the fixed
`@puppeteer/browsers`. `tb-macos-arm64-4`, previously marked "fixed" via a
cache-clear that (per this finding) shouldn't have worked, is worth a retry
to confirm, but no special-casing should be needed.

**Confirmed on real Buildkite runs, 2026-08-05**: retriggered all 6 pipelines
against commit `824261ab9` (the puppeteer bump). On `macstadium-m1-1.local`
(macOS) and `tb-win11-10` (Windows), `npm ci` installed Puppeteer 25.5.0
cleanly and the Puppeteer/Chrome install step ran to completion with no
errors — both machines then failed later, at an unrelated step (missing
`drush`, see "New bug found this run" below). This also answers the
long-standing Windows open question: headless Chrome does run fine under
Buildkite's Windows service context once the extraction bug is out of the
way. However, the version bump broke the WSL2/Linux legs (`tb-wsldd-16`,
`tb-wsl-14`), which run Node.js v18.19.1 — see "Regression: Node 18 vs
Puppeteer 25" below.

### Regression: Node 18 vs Puppeteer 25 (found 2026-08-05, not yet fixed)

Puppeteer 25 dropped CommonJS support (ESM-only package) and requires
Node.js >=22. The macOS/Windows testbots apparently already run a new enough
Node (Puppeteer/Chrome installed and ran fine there), but the WSL2/Linux
testbots (`tb-wsldd-16`, `tb-wsl-14`) are still on Node.js v18.19.1, so
`install-timer.js`'s `require('puppeteer')` now throws:

```
Error [ERR_REQUIRE_ESM]: require() of ES Module .../node_modules/puppeteer/lib/puppeteer/puppeteer.js
from .../install-timer.js not supported.
```

This didn't show up in local testing because the reproduction machine ran
Node 26. No code-side fix exists for this — Puppeteer 25 genuinely requires
Node >=22 at runtime, and downgrading puppeteer would bring back the
extract-zip/Node-26 bug wherever a machine's Node is new enough to trigger
it. `docs/content/developers/buildkite-testmachine-setup.md` already
documents installing Node.js 22 via nodesource for exactly this reason (see
"1. Install Node.js 22, matching the version `perf-linux.yml` uses") — that
guidance just hasn't been applied to `tb-wsldd-16`/`tb-wsl-14` yet. This is
infrastructure work (upgrade Node on those specific boxes), not something
fixable from this checkout — no SSH access to testbots from this environment.

Since two pipelines (`docker-desktop-arm64`'s 2-machine pool,
`shared-providers`' 3-machine pool) round-robin across machines, a pipeline
can look "fixed" on one retry and "broken again" on the next just because a
different machine in its pool picked up the job — always check
`bk api /pipelines/<slug>/builds/<n>` for which agent actually ran the work
job, not just whether it passed.

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
bk build create --pipeline <slug> \
  --branch 20260726_rfay_perf_testing \
  --commit HEAD \
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

## To resume

Puppeteer/Chrome itself is no longer the blocker on macOS or Windows — both
got past that step on the 2026-08-05 retrigger. Remaining work:

1. **Upgrade Node.js to 22+ on the WSL2/Linux testbots** (`tb-wsldd-16`,
   `tb-wsl-14`, and presumably `tb-wsl-12`) per
   `docs/content/developers/buildkite-testmachine-setup.md`. Requires SSH
   access this environment doesn't have — user is handling this directly.
2. **Fix the missing-`drush` bug**: add `ddev composer require drush/drush`
   to `perf.sh`'s one-time project provisioning (or run it by hand on
   `$PERF_PROJECT_DIR` on each machine that already provisioned the project —
   `~/tmp/ddev-perf-drupal11` on the machines checked so far). Blocks
   `05-drush-install.sh` on every platform, not just one.
3. **`tb-macos-arm64-7` orbstack won't start a VM** — `start VM: timed out
   waiting for VM to start`, blocking every leg of `macos-shared-providers`
   that lands on that machine (orbstack is used as a shared reset step
   between legs, not just the leg actually testing orbstack). Needs
   investigation on that machine directly (disk space, orbstack version,
   whether a stuck VM process needs killing) — not something a retrigger
   fixes, and not related to anything fixed so far.
4. Once (1) and (2) are done, retrigger all 6 pipelines again (see `bk build
   create` above) and check whether any of them get a fully clean pass.
   `ddev-perf-wsl2-docker-inside` specifically has had a build stuck
   "scheduled" waiting for an agent across two separate trigger attempts —
   don't just retrigger it again without checking agent availability first
   (`bk agent list --output json`), or it'll just queue a third one.
5. Once at least one pipeline is fully green, exercise the real end-to-end
   dashboard deploy before merging: `gh workflow run perf-linux.yml|perf-collect.yml|docs-publish.yml --ref 20260726_rfay_perf_testing`
   (all three already support `workflow_dispatch`).
