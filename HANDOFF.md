# HANDOFF: CI test-runtime tracking

## Why this exists

#8695 and #8696 proposed trimming the Go test suite and restructuring CI to
cut redundant runtime. Both issues based their findings on a single
hand-collected snapshot (one `gh api`/Buildkite pull), and both explicitly
called out that a single sample can't show duration drift or flakiness over
time. This work builds the durable measurement those issues were missing,
so a future trimming effort can show a real before/after instead of another
one-off snapshot.

## What's here

- **`perf/collector/collect-test-runtime.js`** — nightly collector (wired
  into the existing `perf-collect.yml` cron) that pulls completed `main`
  runs for the GitHub Actions Test-\*/Quickstart workflows
  (`test-workflows.json`) and finished `main` builds for the 12 active
  Buildkite Test-\* pipelines (`test-pipelines.json`), appending one row per
  run/build to `test-runtime-history.ndjson` on the `performance-data`
  branch. Each row has `wall_clock_s`, `compute_s`, `queue_wait_s`
  (Buildkite only — see below), `conclusion`, `skip_marker`, and a per-job
  breakdown.
- **Two dashboard pages, not one with a dataset switcher.** An earlier
  version put both datasets behind one `<select>` on one page; that implied
  a relationship between them that doesn't exist (perf benchmarks are
  plausibly public-interesting, CI build times never are), so it was split.
  `perf/collector/dashboard-common.js`/`.css` hold the shared engine
  (trend/compare charts, checkbox boxes, CSV export); `perf-dashboard.html`
  (deploys as `/perf/`) and `ci-dashboard.html` (deploys as `/perf/ci/`)
  are each a thin page that calls `initDashboard(config)` with just its own
  dataset. The CI page has independent "Test type" and "Runner machine"
  checkbox boxes and a "Only successful full runs" filter (on by default).
  Live at `https://ddev.github.io/ddev/perf/` and
  `https://ddev.github.io/ddev/perf/ci/`.
- **Makefile / `test-reusable.yml` / `test-wsl2-reusable.yml`** — opt-in
  `gotestsum` wiring for GitHub Actions Go test runs (`GOTESTSUM_JSONFILE_DIR`,
  unset by default so local `make test` is unaffected). Installed via
  Homebrew (Linux) or a release tarball (WSL2), not `go install` — an
  earlier `go install`-pinned version failed to compile against the
  runners' current Go toolchain; installing the prebuilt binary sidesteps
  that whole class of problem.
- **`perf-collect.yml`** — gained `fresh`/`since` `workflow_dispatch` inputs
  to rebuild `test-runtime-history.ndjson` from scratch, needed whenever the
  row schema gains a field (existing rows never retroactively get a new
  field otherwise, since the collector only appends).

## Key findings so far (from real data, not hypothetical)

- **Cancellations**: every Test-\* workflow's `cancel-in-progress: true`
  concurrency group fires on ~20% of `main` pushes. Wasted compute per
  cancellation ranges from seconds to ~2 hours depending on how far the
  cancelled run had gotten.
- **`[skip ci]`/`[skip buildkite]`/`[skip github]` commits**: Buildkite
  records these as `conclusion: "passed"` in a few seconds (no separate
  "skipped" state), which — before this was caught and tagged
  (`skip_marker`) — produced wildly misleading duration swings.
  GitHub-side, the same underlying idea is handled by the existing
  `check-skip` job, which reports these as `conclusion: "skipped"` and is
  filtered the normal way.
- **Buildkite queue wait**: `wall_clock_s` for Buildkite used to include
  time spent waiting for a scarce self-hosted agent, not just real test
  time — confirmed on one build where the real job waited 5.4 hours for an
  agent but only ran 65 minutes once it got one. Fixed: `wall_clock_s` and
  `compute_s` are now both real execution time; the wait is its own
  `queue_wait_s` field.
- **Single-agent degradation**: one Buildkite machine
  (`tb-macos-arm64-4-1`, running `ddev-macos-arm64-mutagen`) got steadily
  slower build over build across ~3.5 weeks (99 min → 513 min), while a
  sibling agent on the same pipeline (`macstadium-m1-1.local-1`) stayed
  flat the whole time. Visible now in the dashboard's per-machine split;
  not yet investigated on the actual machine (disk usage, stale
  Colima/Docker state, orphaned processes — that's testbot-infra work, not
  something this tooling can fix).
- **Failed runs are not necessarily short**: the Makefile's target chain
  (`test: testpkg testcmd`; `testpkg: testnotddevapp testddevapp`) aborts on
  the first non-zero recipe (GNU Make's default behavior), so a failure in
  an early stage skips the rest of the chain and produces a short,
  unrepresentative duration — but a failure in the *last* stage to run
  looks like (and mostly is) a normal-length run. The dashboard's default
  "only successful full runs" filter drops all failures regardless of which
  case it was, since neither belongs in a "how long does this normally
  take" measurement.

## TODOs

1. ~~**Split the dataset presentations.**~~ Done: the dashboard was one page
   with a dataset switcher between "Nightly perf benchmarks" and "CI test
   runtime"; now `/perf/` (perf benchmarks) and `/perf/ci/` (CI test
   runtime) are two separate pages sharing a common chart engine
   (`dashboard-common.js`/`.css`), each with a one-line cross-link to the
   other. Old performance-data-branch files from before the split
   (`test-runtime-history.ndjson` at branch root, the old combined
   `dashboard.html`) are left as orphaned data on that branch — harmless,
   not copied into the deployed site by `docs-publish.yml` anymore, but
   worth a manual cleanup pass on `performance-data` at some point.
2. **No Buildkite per-test instrumentation.** `.buildkite/test.sh`/`test.cmd`
   still run plain `go test -v`. Only the GitHub Actions Linux wrappers +
   WSL2 workflow get gotestsum's per-test JSON. Buildkite only has the
   total-runtime (per-run) dataset, nothing per-test.
3. **No permanent per-test time series.** The gotestsum per-test JSON is
   uploaded as a 90-day build artifact per job (GitHub Actions side only —
   see #2) but nothing aggregates it into a lasting dataset or dashboard
   view. Only per-workflow *totals* are tracked long-term today.
4. **`stage_analysis` is unverified against a real failure.** The
   truncated-vs-late-failure detection (reads which gotestsum jsonfiles a
   failed GitHub Actions run actually produced) has never fired on live
   data — it only activates the next time a real GitHub Actions job fails
   after this merges. Worth checking the first time that happens.
5. **No GitHub Actions runner-generation tracking.** Investigated and set
   aside: the Jobs API doesn't expose runner image/generation, only
   `runner_id`/`runner_name`/`labels` (and GH-hosted runners are one-shot
   VMs anyway, so there's no persistent-machine concept to track the way
   Buildkite has). Detecting a silent GitHub-side fleet change would need
   scraping the "Runner Image" line out of raw job logs — a much heavier
   lift, not started.
6. **Open discussion, not decided**: whether to switch gotestsum's console
   format from `--format=standard-verbose` (byte-for-byte like today's
   `go test -v`, chosen deliberately for continuity) to something quieter
   like `--format=pkgname` or `--format=github-actions`, which would shrink
   green-run CI log noise since gotestsum only prints a passing test's full
   output when using verbose formats — quieter formats print one line per
   test and the full dump only on failure. Would change what a normal
   passing-run log looks like; raised for discussion, not acted on.
7. **The actual #8695/#8696 trimming work hasn't started.** This PR is the
   measurement infrastructure the trimming work needs as a baseline/backfill
   before-and-after comparison — the trimming itself is a separate,
   not-yet-begun effort.
8. **No write-up posted back to #8695/#8696 yet.** The dashboard and the
   findings above (cancellation rate, skip-marker contamination, Buildkite
   queue wait, the single-agent degradation) haven't been shared back to
   those issues.

## Where to look

- Live CI test-runtime dashboard: `https://ddev.github.io/ddev/perf/ci/`
- Live nightly perf-benchmark dashboard: `https://ddev.github.io/ddev/perf/`
- Raw CI test-runtime data: `https://ddev.github.io/ddev/perf/ci/test-runtime-history.ndjson`
- Collector: `perf/collector/collect-test-runtime.js`,
  `perf/collector/README.md` has the fuller technical writeup
- To backfill after a schema change: `gh workflow run perf-collect.yml
  --ref <branch> -f fresh=true -f since=YYYY-MM-DD`, then `gh workflow run
  docs-publish.yml --ref <branch>` to redeploy (two separate hops — the
  dashboard HTML only refreshes on `performance-data` when
  `perf-collect.yml` runs; `docs-publish.yml` just redeploys whatever is
  currently there)
