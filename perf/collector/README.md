# Collector and dashboard

Run by `.github/workflows/perf-collect.yml` on a schedule, after the nightly
`perf-*.yml` (Buildkite) and `perf-linux.yml` (GitHub Actions) benchmark jobs
should have finished.

- `collect.js` pulls the latest passed build's `perf-result.json` artifact from
  each Buildkite pipeline listed in `pipelines.json`, plus the latest
  successful `perf-linux.yml` run's artifact, and appends one line per leg to
  `history.ndjson` on the `performance-data` branch.
- `dashboard.html` is a dependency-free static page: it `fetch()`es
  `history.ndjson` client-side and renders per-metric trend lines, filterable
  by metric and by leg (platform/arch/docker provider). Published as
  `index.html` on the `performance-data` branch via GitHub Pages.

## One-time manual setup

These can't be done from the repo itself:

1. Create a `BUILDKITE_API_TOKEN` repository secret with read access to builds
   and artifacts (Buildkite dashboard → your user → API Access Tokens).
2. Enable GitHub Pages for `ddev/ddev` with source **GitHub Actions**
   (repo Settings → Pages).
3. Once the `.buildkite/perf-*.yml` pipelines exist in the Buildkite dashboard
   (see `perf/README.md`), update `pipelines.json` with their real slugs —
   the ones checked in are suggested names, not guaranteed to match what
   was actually created.

## Regression signal

Kept intentionally simple for now: `dashboard.html` computes, per leg and
metric, the trailing median of the last 10 points and marks any point more
than 20% above it in red. This is a visual flag only — nothing fails CI or
pages anyone on it yet. Revisit once the signal has been observed to be
trustworthy over a few weeks (see the main plan/README for why).
