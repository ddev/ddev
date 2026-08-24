# Collector and dashboard

Run by `.github/workflows/perf-collect.yml` on a schedule, after the nightly
`perf-*.yml` (Buildkite) and `perf-linux.yml` (GitHub Actions) benchmark jobs
should have finished.

- `collect.js` pulls every `perf-result*.json` artifact from the latest passed
  build of each Buildkite pipeline listed in `pipelines.json` (a build can hold
  more than one -- `perf-macos-shared-providers.yml` runs five providers as
  separate jobs in the same build), plus the latest successful `perf-linux.yml`
  run's artifact, and appends one line per leg to `history.ndjson` on the
  `performance-data` branch.
- `collect-test-runtime.js` is a separate dataset: total runtime per CI test
  type (`golang-nginx-fpm`, `macos-lima`, `quickstart`, etc.), not narrow
  synthetic benchmarks. It pulls completed `main`-branch runs for every
  GitHub Actions workflow in `test-workflows.json` via `gh api`, plus finished
  `main`-branch builds for every Buildkite pipeline in `test-pipelines.json`
  (the actual Go-test-suite pipelines -- `macos-lima`, `wsl2-mirrored`, etc. --
  not the nightly `ddev-perf-*` benchmarks in `pipelines.json`). No test
  instrumentation required for this dataset. It appends one row per run/build
  to `test-runtime-history.ndjson`, with `wall_clock_s` (queue-excluded
  duration) and `compute_s` (summed job duration, so a run split into more
  than one job -- podman-rootless's testpkg/testcmd, WSL2 NAT -- reports real
  CPU-minutes, not just the longest job) plus a `jobs[]` breakdown with each
  job's runner labels (GitHub Actions labels or Buildkite agent `meta_data`
  tags, e.g. `os=macos`, `lima=true`). Incremental by default (derives
  `--since` from the newest timestamp already in the history file, with a
  1-day overlap); pass `--since=YYYY-MM-DD` to backfill further back. Reruns
  are recorded as distinct points (keyed on source + workflow + run/build
  number + attempt), not deduped away, since a flaky-test rerun is itself a
  real measurement. `BUILDKITE_API_TOKEN` is the same one `collect.js` uses;
  if unset, the Buildkite side is skipped (not an error), same tolerance.
  Complements the per-test data gotestsum writes when `GOTESTSUM_JSONFILE_DIR`
  is set (see `Makefile`) -- that's per-Go-test detail uploaded as a build
  artifact (GitHub Actions via `actions/upload-artifact`, Buildkite via
  `buildkite-agent artifact upload` in `test.sh`) by the workflows that run
  `make test`; this script is the per-test-type total that also covers
  non-Go workflows like Quickstart.
  `test-pipelines.json`'s comment explains how its 12 slugs were confirmed
  live (`bk pipeline list --org ddev`) against several retired/legacy
  pipelines the Buildkite API still returns -- see #8695.

  `perf-collect.yml`'s `workflow_dispatch` has a `fresh` input (wipes
  test-runtime-history.ndjson before collecting, instead of appending) and a
  `since` input (overrides the default 7-day backfill window). Since the
  collector dedupes by run/build identity, adding a field to the row schema
  only appears on rows collected *after* that change -- `fresh=true` (with
  `since` set far enough back) re-derives every existing row from GitHub's
  and Buildkite's own retained history, rather than leaving old rows
  permanently missing the new field until enough new data dilutes them.

  For GitHub Actions runs GitHub itself marked `failure`, the collector also
  downloads that run's gotestsum artifact and checks which of
  `testnotddevapp.ndjson`/`testddevapp.ndjson`/`testcmd.ndjson` exist, adding
  a `stage_analysis` object to the matching job in `jobs[]`. The Makefile's
  target chain (`test: testpkg testcmd`; `testpkg: testnotddevapp
  testddevapp`) means a failure in an earlier target aborts the rest --
  GNU Make's default behavior on a non-zero recipe -- so a target whose
  jsonfile never got written never ran. `stage_analysis.truncated` is true
  when that happened (the run's `wall_clock_s`/`compute_s` reflect only part
  of a full run, not a real completion -- exclude these from duration trends
  the same way cancelled runs already are); `failed_at_stage` names the
  target that never ran; `failing_tests` lists the actual failing test names
  from whichever stage did run last. No artifact match (the common case
  before this was added; also still true for Buildkite, since this collector
  doesn't fetch its gotestsum artifacts yet -- see HANDOFF.md) just means no
  `stage_analysis` -- not an error.
  `makeTargetFromArtifactName`/`analyzeStageDir` in the script are pure
  functions verified against fixture data (no jest/mocha dependency here, so
  ad hoc rather than a committed test file -- same as `collect.js`).
- **Two separate dashboard pages, not one page with a dataset switcher.**
  The nightly perf benchmarks and CI test runtime are unrelated datasets for
  unrelated audiences -- the perf numbers are plausibly public-interesting
  (DDEV's speed across Docker providers); CI build times never are (see
  HANDOFF.md). An earlier version of this put both behind one `<select>` on
  one page; that implied a relationship between them that doesn't exist, so
  it was split.
  - `dashboard-common.js` + `dashboard-common.css` are the shared engine and
    styling both pages use -- everything dataset-agnostic (trend/compare
    chart drawing, the workflow/runner checkbox boxes, CSV export). A page
    calls `initDashboard(config)` once with its own small config object
    (which file to fetch, how to key/filter/format its rows); there's no
    dataset switching inside the engine itself.
  - `perf-dashboard.html` is the nightly perf benchmarks page (deploys as
    `/perf/index.html`): filterable by metric and by leg
    (platform/arch/docker provider). Two views: a per-metric trend line over
    time (the default), and a "Compare environments" bar chart -- one bar
    per leg, the median of its last 7 nightly runs for the selected metric,
    sorted fastest to slowest. The bar chart deliberately doesn't normalize
    for the fact that legs run on different physical/virtual machines --
    it's comparing Docker-provider/platform overhead, which is the point,
    not controlling for hardware.
  - `ci-dashboard.html` is the CI test-runtime page (deploys as
    `/perf/ci/index.html`): same trend/compare views, filterable by test
    type and (for Buildkite pipelines) by runner machine -- see
    `collect-test-runtime.js`'s section above for what the data means.
  - Both pages have a "Download CSV" link exporting the rows behind whichever
    chart and filters are currently on screen -- for slide decks or
    spreadsheets, so numbers quoted from a presentation match what the
    dashboard shows.

Neither collector step deploys to GitHub Pages directly. A repo has only one
Pages site, and `.github/workflows/docs-publish.yml` already deploys the docs
there on every push to `main`/`stable` -- deploying the dashboards separately
would race it and the two would overwrite each other. Instead,
`perf-collect.yml` commits the perf-benchmark files (`history.ndjson`,
`index.html`, `dashboard-common.js`, `dashboard-common.css`) at the root of
the `performance-data` branch, and the CI-test-runtime files
(`ci/test-runtime-history.ndjson`, `ci/index.html`) under its `ci/`
subdirectory, then (if anything changed) triggers `docs-publish.yml`, which
checks out that branch and folds both into the site under `/perf/` and
`/perf/ci/` respectively.

## One-time manual setup

These can't be done from the repo itself:

1. Add a `BUILDKITE_API_TOKEN` item (read access to builds and artifacts;
   Buildkite dashboard → your user → API Access Tokens) to the `test-secrets`
   1Password vault used by the `TESTS_SERVICE_ACCOUNT_TOKEN` repo secret.
2. Once the `.buildkite/perf-*.yml` pipelines exist in the Buildkite dashboard
   (see `perf/README.md`), update `pipelines.json` with their real slugs —
   the ones checked in are suggested names, not guaranteed to match what
   was actually created.

## Regression signal

Kept intentionally simple for now: `dashboard-common.js` computes, per leg
and metric, the trailing median of the last 10 points and marks any point
more than 20% above it in red -- on both pages. This is a visual flag only —
nothing fails CI or pages anyone on it yet. Revisit once the signal has been
observed to be trustworthy over a few weeks (see the main plan/README for
why).
