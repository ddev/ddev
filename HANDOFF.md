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
- **Just fixed, not yet committed**: bumped `puppeteer` in
  `perf/metrics/03-drupal-install/package.json` from `^24.11.2` to `^25.5.0`
  and regenerated `package-lock.json` — root-causes the macOS Puppeteer
  blocker below (upstream bug
  [#14957](https://github.com/puppeteer/puppeteer/issues/14957): unmaintained
  `extract-zip@2.0.1` silently truncates browser extraction on Node.js 26).
  See "The macOS blocker in detail" for the full writeup.

## Buildkite side: 6 pipelines created, all currently red

All created via `bk api --method POST /pipelines` (not `bk pipeline create`
— see gotchas below), `build_branches: false` (branch filters disabled) so
they don't fire on every push while this is still on a feature branch.

| Pipeline slug                          | Machine(s) behind it                                              | Status |
|-----------------------------------------|---------------------------------------------------------------------|--------|
| `ddev-perf-macos-docker-desktop-arm64`  | `tb-macos-arm64-4`, `macstadium-m1-1.local` (2-machine pool)        | Blocked — Puppeteer cache corruption, see below |
| `ddev-perf-macos-shared-providers`      | `tb-macos-arm64-5`, `-6`, `-7` (3-machine pool, 5 sequential legs)   | Blocked — same cache issue on `-6`; `-5`/`-7` untested |
| `ddev-perf-windows10-docker-desktop`    | `tb-win11-10` (single machine)                                       | Blocked — Chrome never downloads at all, no error surfaced. **This is the actual open question**: can headless Chrome even run under Buildkite's Windows service (Session 0) context? Not yet answered — need to run `npx puppeteer browsers install chrome` directly on that box to see the real error. |
| `ddev-perf-wsl2-docker-desktop`         | `tb-wsldd-16` (single machine)                                       | Failed with a corrupted-zip download (`end of central directory record signature not found`) — looks transient, not stale cache. Worth checking disk space (`df -h /var/lib/buildkite-agent`) and just retrying. |
| `ddev-perf-wsl2-docker-inside`          | shares the `wsl2-mirrored` pool tags                                 | Was stuck "scheduled" waiting for an agent — not a bug, just no free agent yet |
| `ddev-perf-wsl2-mirrored`               | `tb-wsl-12`, `tb-wsl-14`                                              | Got past the earlier missing-`libasound.so.2` issue; then hit the now-fixed silent-failure bug in `05-drush-install.sh`. Should retry now that `run_quiet` is in. |

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

**Fix (already applied, commit pending): bump the `puppeteer` dependency.**
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

1. The `puppeteer` version bump (see "The macOS blocker in detail") should
   make this self-healing on retry — no per-machine SSH/repair work expected.
   Retrigger a build on each of `macstadium-m1-1.local`, `tb-macos-arm64-4`,
   `-5`, `-6`, `-7` (via the round-robin pools) and confirm `npm ci` extracts
   both `chrome` and `chrome-headless-shell` cleanly; only fall back to
   investigating a specific machine if one still fails after this fix.
2. Once all 5 macOS machines are clean, retrigger
   `ddev-perf-macos-docker-desktop-arm64` and `ddev-perf-macos-shared-providers`
   (see `bk build create` above) and watch for a clean pass all the way
   through.
3. Separately (not blocking macOS): check disk space on `tb-wsldd-16` and
   retry `ddev-perf-wsl2-docker-desktop`; retry `ddev-perf-wsl2-mirrored` now
   that the silent-failure bug is fixed; check whether
   `ddev-perf-wsl2-docker-inside` ever picked up an agent.
4. Separately (the real open question): on `tb-win11-10`, run
   `npx puppeteer browsers install chrome` by hand to see why the Buildkite
   job's Chrome download produced no output/error at all. The `puppeteer`
   version bump may already fix this too if it was the same
   extract-zip-on-Node-26 bug, but confirm directly rather than assuming —
   that's the actual test of whether headless Chrome can run under
   Buildkite's Windows service context at all.
5. Once at least one pipeline is fully green, exercise the real end-to-end
   dashboard deploy before merging: `gh workflow run perf-linux.yml|perf-collect.yml|docs-publish.yml --ref 20260726_rfay_perf_testing`
   (all three already support `workflow_dispatch`).
