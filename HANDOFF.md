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

## Status as of 2026-08-05: 5 of 6 pipelines fully passing

| Pipeline slug                          | Status |
|-----------------------------------------|--------|
| `ddev-perf-macos-docker-desktop-arm64`  | **Passing** |
| `ddev-perf-macos-shared-providers`      | **Passing, all 5 legs** (`colima_vz`, `lima`, `orbstack`, `podman-rootless`, `rancher-desktop`) |
| `ddev-perf-wsl2-docker-desktop`         | **Passing** |
| `ddev-perf-wsl2-mirrored`               | **Passing** |
| `ddev-perf-windows10-docker-desktop`    | **Passing** |
| `ddev-perf-wsl2-docker-inside`          | **Deprioritized** — no connected agent has ever matched its required tags (`os=wsl2` + `dockertype=wsl2`); `wsl2-mirrored` is the canonical pipeline for "docker native inside WSL2" instead. Stop retriggering this one. |

The bugs that got in the way (Puppeteer/Node version mismatches, a missing
`drush` dependency, a Windows file-lock race) are captured in their own
commit messages on this branch — no need to duplicate that history here.

## To resume

1. Added an "orbstack, no Mutagen" leg to `ddev-perf-macos-shared-providers`
   (its own isolated project dir via `PERF_PROJECT_DIR_SUFFIX`, `DOCKER_TYPE`
   stays `orbstack` for provider bring-up, `DOCKER_PROVIDER_LABEL` gives it a
   distinct reported name) — not yet exercised against a real Buildkite run.
   Along the way, fixed a real bug this would have hit immediately:
   `ddev mutagen status`/`sync` both exit 0 even when Mutagen is disabled, so
   `02-mutagen-sync-settle.sh`/`reset-drupal.sh`'s guards never actually
   detected "disabled" correctly — now check `ddev describe -j`'s
   `mutagen_enabled` field instead (`mutagen_enabled_for_project` in
   `perf/lib/common.sh`).
2. Worth one or two more retriggers of each pipeline to confirm the current
   green state isn't a lucky run (especially the two round-robin pools,
   which can land on a different machine each time).
3. Update `perf/collector/pipelines.json` with the real pipeline slugs for
   all 6 pipelines, if not already done — needed for `collect.js` to find
   these pipelines' artifacts.
4. Exercise the real end-to-end dashboard deploy before merging: `gh workflow
   run perf-linux.yml|perf-collect.yml|docs-publish.yml --ref
   20260726_rfay_perf_testing` (all three already support
   `workflow_dispatch`). Nothing so far has actually exercised `collect.js`
   pulling real artifacts, `history.ndjson` accumulating, or the dashboard
   rendering real data.

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
