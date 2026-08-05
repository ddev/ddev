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
- `dashboard.html` is a dependency-free static page: it `fetch()`es
  `history.ndjson` client-side and renders per-metric trend lines, filterable
  by metric and by leg (platform/arch/docker provider).

Neither step deploys to GitHub Pages directly. A repo has only one Pages site,
and `.github/workflows/docs-publish.yml` already deploys the docs there on
every push to `main`/`stable` -- deploying the dashboard separately would race
it and the two would overwrite each other. Instead, `perf-collect.yml` commits
`history.ndjson` and a rendered `index.html` to the `performance-data` branch,
and (if anything changed) triggers `docs-publish.yml`, which checks out that
branch and folds the dashboard into the site under `/perf/`.

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

Kept intentionally simple for now: `dashboard.html` computes, per leg and
metric, the trailing median of the last 10 points and marks any point more
than 20% above it in red. This is a visual flag only — nothing fails CI or
pages anyone on it yet. Revisit once the signal has been observed to be
trustworthy over a few weeks (see the main plan/README for why).
