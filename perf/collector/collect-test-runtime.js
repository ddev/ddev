#!/usr/bin/env node
// Pulls completed `main`-branch runs for each workflow in test-workflows.json
// via `gh api`, plus finished `main`-branch builds for each Buildkite Test-*
// pipeline in test-pipelines.json, and appends one row per run/build (with a
// per-job breakdown) to test-runtime-history.ndjson on the performance-data
// branch. This is the "total runtime per test type" (golang-nginx-fpm,
// macos-lima, quickstart, etc.) dataset -- see perf/collector/README.md.
// Companion to collect.js, which tracks the narrower nightly perf-benchmark
// metrics.
//
// Required env:
//   GITHUB_TOKEN         - for `gh` CLI calls against this repo's Actions runs
//   GITHUB_REPOSITORY    - owner/repo, set automatically in GitHub Actions
//   BUILDKITE_API_TOKEN  - read access to builds; if unset, the Buildkite
//                          pipelines are skipped (not an error) -- same
//                          tolerance as collect.js, for the pre-setup state.
//
// Usage: node perf/collector/collect-test-runtime.js <path-to-test-runtime-history.ndjson> [--since=YYYY-MM-DD]
//
// --since is normally omitted: the incremental (nightly) run derives it from
// the newest timestamp already in the history file, with a 1-day overlap to
// tolerate any run that was still in progress the last time this collected.
// Pass --since explicitly to backfill further back than the existing history
// -- e.g. to seed the dataset for the first time, or to re-pull a known gap.

const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

// Parsed unconditionally (not just under the CLI guard below) so `main()`
// and `computeSince()` can close over them either way; only actually
// validated when this file is run directly rather than require()'d, e.g. by
// collect-test-runtime.test.js's pure-function tests.
const args = process.argv.slice(2);
const HISTORY_PATH = args.find((a) => !a.startsWith('--'));
const sinceArg = args.find((a) => a.startsWith('--since='));

const CONFIG = JSON.parse(fs.readFileSync(path.join(__dirname, 'test-workflows.json'), 'utf8'));
const BK_CONFIG = JSON.parse(fs.readFileSync(path.join(__dirname, 'test-pipelines.json'), 'utf8'));

// Set whenever any single workflow fails to collect, so the run can still
// finish, commit, and publish whatever DID come through clean -- mirrors
// collect.js's per-source error tolerance (see its header comment for why).
let hadErrors = false;

const DEFAULT_BACKFILL_DAYS = 7;
const OVERLAP_DAYS = 1;

function ghApiPages(urlPath, params) {
  const repo = process.env.GITHUB_REPOSITORY;
  if (!repo) throw new Error('GITHUB_REPOSITORY must be set');
  const fArgs = params.flatMap((p) => ['-f', p]);
  // --method GET is not optional here: gh api silently switches to POST as
  // soon as any -f flag is present unless the method is given explicitly,
  // and POST against these (GET-only) endpoints fails as a plain 404 rather
  // than a method-not-allowed error -- easy to misread as a bad path/token.
  const out = execFileSync(
    'gh',
    ['api', '--method', 'GET', '--paginate', '--slurp', `repos/${repo}${urlPath}`, ...fArgs],
    { encoding: 'utf8', maxBuffer: 1024 * 1024 * 64 }
  );
  return JSON.parse(out);
}

function fetchRuns(workflowFile, since) {
  console.log(`Fetching ${workflowFile} runs on main since ${since}...`);
  const pages = ghApiPages(`/actions/workflows/${workflowFile}/runs`, [
    'branch=main',
    'status=completed',
    'per_page=100',
    `created=>=${since}`,
  ]);
  const runs = pages.flatMap((p) => p.workflow_runs || []);
  console.log(`  Found ${runs.length} completed run(s)`);
  return runs;
}

function fetchJobs(runId) {
  const pages = ghApiPages(`/actions/runs/${runId}/jobs`, ['per_page=100']);
  return pages.flatMap((p) => p.jobs || []);
}

// Order matches the Makefile's dependency chain (test: testpkg testcmd;
// testpkg: testnotddevapp testddevapp), which is also Make's default
// execution order. A failure in an earlier target aborts the chain --
// GNU Make's default behavior on a non-zero recipe -- so later targets'
// gotestsum jsonfiles never get written. See perf/collector/README.md.
const MAKE_TARGET_STAGES = {
  test: ['testnotddevapp', 'testddevapp', 'testcmd'],
  testpkg: ['testnotddevapp', 'testddevapp'],
  testcmd: ['testcmd'],
};

function fetchRunArtifacts(runId) {
  const pages = ghApiPages(`/actions/runs/${runId}/artifacts`, ['per_page=100']);
  return pages.flatMap((p) => p.artifacts || []);
}

// For a failed run, the gotestsum jsonfiles already uploaded (one per
// Makefile target -- see Makefile's `gotest` macro) tell us, for free,
// exactly how far the run got before Make aborted the chain: a target
// whose jsonfile never showed up never ran. Requires no Makefile or
// workflow change -- it's pulled from an artifact we already upload.
// Match by suffix, not a prefix capture: test-reusable.yml names artifacts
// test-results-<run>-<attempt>-<make_target>, but test-wsl2-reusable.yml
// inserts extra segments first -- test-results-<run>-<attempt>-wsl2-
// <networking>-<make_target> -- so a naive "everything after the 2nd dash
// group" capture would grab "wsl2-nat-testcmd" instead of "testcmd". Check
// testpkg/testcmd before test since neither ends in the bare "-test".
function makeTargetFromArtifactName(artifactName) {
  return ['testpkg', 'testcmd', 'test'].find((t) => artifactName.endsWith(`-${t}`)) || null;
}

// Pure over an already-downloaded directory, so it's testable without a real
// `gh run download` -- see perf/collector/collect-test-runtime.test.js.
function analyzeStageDir(makeTarget, dir) {
  const expectedStages = MAKE_TARGET_STAGES[makeTarget];
  if (!expectedStages) return null;

  const stagesRun = expectedStages.filter((stage) => fs.existsSync(path.join(dir, `${stage}.ndjson`)));
  const truncated = stagesRun.length < expectedStages.length;
  const lastStage = stagesRun[stagesRun.length - 1] || null;

  let failingTests = [];
  if (lastStage) {
    const seen = new Set();
    for (const line of fs.readFileSync(path.join(dir, `${lastStage}.ndjson`), 'utf8').split('\n')) {
      if (!line.trim()) continue;
      try {
        const event = JSON.parse(line);
        if (event.Action === 'fail' && event.Test) seen.add(event.Test);
      } catch {
        // gotestsum's jsonfile is one JSON object per line; an unparseable
        // line here would mean a corrupted upload, not a real event -- skip.
      }
    }
    failingTests = [...seen];
  }

  return {
    make_target: makeTarget,
    stages_run: stagesRun,
    truncated,
    // Null when every expected stage ran -- the failure is a real test
    // failure in lastStage, not a chain abort before it.
    failed_at_stage: truncated ? expectedStages[stagesRun.length] : null,
    failing_tests: failingTests,
  };
}

function analyzeStageArtifact(repo, runId, artifactName) {
  const makeTarget = makeTargetFromArtifactName(artifactName);
  if (!makeTarget) {
    console.warn(`  Unrecognized make_target from artifact "${artifactName}"; skipping stage analysis`);
    return null;
  }

  const tmpDir = fs.mkdtempSync('/tmp/ddev-test-results-');
  try {
    execFileSync('gh', ['run', 'download', String(runId), '--repo', repo, '-n', artifactName, '-D', tmpDir]);
  } catch (err) {
    console.warn(`  Could not download artifact ${artifactName} for run ${runId}: ${err.message}`);
    return null;
  }

  return analyzeStageDir(makeTarget, tmpDir);
}

module.exports = { makeTargetFromArtifactName, analyzeStageDir, MAKE_TARGET_STAGES, hasSkipMarker };

async function buildkiteApiPaginated(urlPath, params) {
  const token = process.env.BUILDKITE_API_TOKEN;
  if (!token) throw new Error('BUILDKITE_API_TOKEN must be set');
  const perPage = 100;
  let page = 1;
  const all = [];
  for (;;) {
    const qs = new URLSearchParams({ ...params, page: String(page), per_page: String(perPage) });
    const url = `https://api.buildkite.com/v2/${urlPath}?${qs}`;
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
    if (!res.ok) throw new Error(`Buildkite API error ${res.status} for ${url}`);
    const batch = await res.json();
    all.push(...batch);
    if (batch.length < perPage) break;
    page++;
  }
  return all;
}

async function fetchBuildkiteBuilds(pipeline, since) {
  console.log(`Fetching Buildkite pipeline ${pipeline} builds on main since ${since}...`);
  const builds = await buildkiteApiPaginated(
    `organizations/${BK_CONFIG.buildkiteOrg}/pipelines/${pipeline}/builds`,
    { branch: 'main', created_from: `${since}T00:00:00Z` }
  );
  // Buildkite has no "status=completed" filter -- exclude builds still in
  // flight (no finished_at yet) client-side instead.
  const finished = builds.filter((b) => b.finished_at);
  console.log(`  Found ${finished.length} finished build(s) of ${builds.length} total`);
  return finished;
}

function durationSeconds(startIso, endIso) {
  if (!startIso || !endIso) return null;
  const ms = new Date(endIso).getTime() - new Date(startIso).getTime();
  return Number.isFinite(ms) ? Math.round(ms / 1000) : null;
}

// [skip buildkite]/[skip ci] (checked by .buildkite/test.sh, before any real
// setup) and [skip github] (checked by test-reusable.yml's check-skip job)
// all make the run/build exit almost instantly while still recording a
// success conclusion -- Buildkite has no separate "skipped" state the way a
// GitHub job does, so without this tag these near-zero-duration runs mix
// into the same success bucket as a real multi-hour run, producing the
// "vast swings" the dashboard showed before this was added. Checked against
// all three markers on both sources rather than just the one each side's
// own script looks for, since what matters here is "was this run real,"
// not which specific tool would have honored the marker.
const SKIP_MARKERS = ['[skip ci]', '[skip buildkite]', '[skip github]'];
function hasSkipMarker(message) {
  if (!message) return false;
  const lower = message.toLowerCase();
  return SKIP_MARKERS.some((marker) => lower.includes(marker));
}

function toRow(run, jobs) {
  const jobRows = jobs.map((j) => ({
    name: j.name,
    labels: j.labels || [],
    runner_name: j.runner_name || null,
    conclusion: j.conclusion,
    duration_s: durationSeconds(j.started_at, j.completed_at),
  }));
  return {
    timestamp: run.run_started_at || run.created_at,
    source: 'github',
    workflow: run.name,
    workflow_file: path.basename(run.path || ''),
    conclusion: run.conclusion,
    skip_marker: hasSkipMarker(run.head_commit && run.head_commit.message),
    // Wall-clock: when the run actually started executing (post-queue) to
    // completion. What "total runtime for golang-nginx-fpm" means day to
    // day -- queue time is infra noise, not the thing #8695/#8696 restructure.
    wall_clock_s: durationSeconds(run.run_started_at, run.updated_at),
    // Compute cost: sum of every job's own duration, so workflows split into
    // more than one job (podman-rootless's testpkg/testcmd, WSL2 NAT) still
    // report the real CPU-minutes spent, not just the wall-clock of whichever
    // job happened to run longest.
    compute_s: jobRows.reduce((sum, j) => sum + (j.duration_s || 0), 0),
    commit_sha: run.head_sha,
    branch: run.head_branch,
    run_id: run.id,
    run_attempt: run.run_attempt,
    html_url: run.html_url,
    jobs: jobRows,
  };
}

function toBuildkiteRow(pipeline, build) {
  // Only "script" jobs (Buildkite's term for an actual command step) have
  // real runtime -- "waiter"/"trigger"/"manual" job types are structural and
  // would otherwise dilute compute_s with near-zero entries.
  const scriptJobs = (build.jobs || []).filter((j) => j.type === 'script');
  const jobRows = scriptJobs.map((j) => ({
    name: j.name || j.step_key || null,
    labels: (j.agent && j.agent.meta_data) || [],
    runner_name: (j.agent && j.agent.name) || null,
    conclusion: j.state,
    duration_s: durationSeconds(j.started_at, j.finished_at),
  }));
  return {
    timestamp: build.started_at || build.created_at,
    source: 'buildkite',
    workflow: pipeline,
    workflow_file: null,
    conclusion: build.state,
    skip_marker: hasSkipMarker(build.message),
    wall_clock_s: durationSeconds(build.started_at, build.finished_at),
    compute_s: jobRows.reduce((sum, j) => sum + (j.duration_s || 0), 0),
    commit_sha: build.commit,
    branch: build.branch,
    // Buildkite build numbers are only unique per pipeline (unlike GH's
    // repo-wide run ids) -- see pointKey(), which folds `workflow` in too.
    run_id: build.number,
    run_attempt: 1,
    html_url: build.web_url,
    jobs: jobRows,
  };
}

// Identity of one data point. For GitHub, a specific attempt of a specific
// run -- run_attempt increments on a manual rerun, which is a distinct, real
// measurement, not a duplicate (flaky-test reruns are exactly the kind of
// thing this dataset should be able to show). `workflow` is folded in because
// Buildkite build numbers (run_id here) are unique only within their own
// pipeline, not across the dataset the way GitHub's run ids are.
function pointKey(row) {
  return `${row.source}:${row.workflow}:${row.run_id}|${row.run_attempt}`;
}

function existingPointKeys(historyPath) {
  if (!fs.existsSync(historyPath)) return new Set();
  const keys = new Set();
  for (const line of fs.readFileSync(historyPath, 'utf8').split('\n')) {
    if (!line.trim()) continue;
    try {
      keys.add(pointKey(JSON.parse(line)));
    } catch (err) {
      console.warn(`Ignoring unparseable history line: ${err.message}`);
    }
  }
  return keys;
}

function latestTimestamp(historyPath) {
  if (!fs.existsSync(historyPath)) return null;
  let latest = null;
  for (const line of fs.readFileSync(historyPath, 'utf8').split('\n')) {
    if (!line.trim()) continue;
    try {
      const ts = JSON.parse(line).timestamp;
      if (ts && (!latest || ts > latest)) latest = ts;
    } catch {
      // already warned about in existingPointKeys; ignore here
    }
  }
  return latest;
}

function computeSince(historyPath) {
  if (sinceArg) return sinceArg.slice('--since='.length);
  const latest = latestTimestamp(historyPath);
  if (!latest) {
    const d = new Date(Date.now() - DEFAULT_BACKFILL_DAYS * 86400 * 1000);
    return d.toISOString().slice(0, 10);
  }
  const d = new Date(new Date(latest).getTime() - OVERLAP_DAYS * 86400 * 1000);
  return d.toISOString().slice(0, 10);
}

// Only for runs GitHub itself marked "failure" -- success and cancelled runs
// have nothing to explain here, and this is the one part of collection that
// costs a real artifact download per call, so it's worth keeping scoped to
// where it actually answers a question (was this failure's duration
// truncated by an early Make-chain abort, or a real failure late in the
// chain -- see the #8695/#8696 "quit on first failure" discussion).
function enrichFailedGithubRuns(results, repo) {
  for (const row of results) {
    if (row.source !== 'github' || row.conclusion !== 'failure') continue;
    let artifacts;
    try {
      artifacts = fetchRunArtifacts(row.run_id);
    } catch (err) {
      console.warn(`Could not list artifacts for run ${row.run_id}: ${err.message}`);
      continue;
    }
    const prefix = `test-results-${row.run_id}-${row.run_attempt}-`;
    const matches = artifacts.filter((a) => a.name.startsWith(prefix));
    for (const artifact of matches) {
      const analysis = analyzeStageArtifact(repo, row.run_id, artifact.name);
      if (!analysis) continue;
      // Crude but effective: a job's own name (e.g. "podman-rootless-testcmd
      // / test") contains its make_target ("testcmd") for every workflow in
      // test-workflows.json, including the single-job ones where make_target
      // is just "test".
      const job = row.jobs.find((j) => j.name && j.name.includes(analysis.make_target));
      if (job) job.stage_analysis = analysis;
    }
  }
}

async function main() {
  const since = computeSince(HISTORY_PATH);
  const results = [];

  for (const workflowFile of CONFIG.workflows) {
    try {
      const runs = fetchRuns(workflowFile, since);
      for (const run of runs) {
        try {
          const jobs = fetchJobs(run.id);
          results.push(toRow(run, jobs));
        } catch (err) {
          console.error(`ERROR fetching jobs for ${workflowFile} run ${run.id}: ${err.message}`);
          hadErrors = true;
        }
      }
    } catch (err) {
      console.error(`ERROR collecting ${workflowFile}: ${err.message}`);
      hadErrors = true;
    }
  }

  enrichFailedGithubRuns(results, process.env.GITHUB_REPOSITORY);

  // A missing token is the expected, tolerated state before the one-time
  // BUILDKITE_API_TOKEN vault setup (shared with collect.js's perf-* legs) --
  // skip Buildkite entirely and still keep whatever GitHub Actions rows came
  // through above, without counting it as an error.
  if (!process.env.BUILDKITE_API_TOKEN) {
    console.warn('BUILDKITE_API_TOKEN not set; skipping Buildkite pipelines');
  } else {
    for (const pipeline of BK_CONFIG.pipelines) {
      try {
        const builds = await fetchBuildkiteBuilds(pipeline, since);
        for (const build of builds) {
          results.push(toBuildkiteRow(pipeline, build));
        }
      } catch (err) {
        console.error(`ERROR collecting Buildkite pipeline ${pipeline}: ${err.message}`);
        hadErrors = true;
      }
    }
  }

  if (!results.length) {
    console.warn('No results collected this run; leaving test-runtime-history.ndjson unchanged');
    return;
  }

  const seen = existingPointKeys(HISTORY_PATH);
  const fresh = [];
  for (const row of results) {
    const key = pointKey(row);
    if (seen.has(key)) {
      console.log(`Skipping already-recorded run: ${key}`);
      continue;
    }
    seen.add(key);
    fresh.push(row);
  }

  if (!fresh.length) {
    console.warn('All collected runs were already recorded; leaving test-runtime-history.ndjson unchanged');
    return;
  }

  const lines = fresh.map((r) => JSON.stringify(r));
  fs.appendFileSync(HISTORY_PATH, lines.join('\n') + '\n');
  console.log(`Appended ${lines.length} new run(s) of ${results.length} collected to ${HISTORY_PATH}`);
}

// Guarded so collect-test-runtime.test.js can require() this file for its
// pure functions (analyzeStageDir, makeTargetFromArtifactName) without
// triggering a real collection run or the "no HISTORY_PATH" exit below.
if (require.main === module) {
  if (!HISTORY_PATH) {
    console.error('Usage: collect-test-runtime.js <path-to-history.ndjson> [--since=YYYY-MM-DD]');
    process.exit(1);
  }

  main()
    .then(() => {
      // Exit 0 even when hadErrors, so the caller's later workflow steps
      // (commit/push, docs-publish trigger) still run and publish whatever
      // clean results came through -- mirrors collect.js.
      if (process.env.GITHUB_OUTPUT) {
        fs.appendFileSync(process.env.GITHUB_OUTPUT, `test_runtime_had_errors=${hadErrors}\n`);
      }
      if (hadErrors) {
        console.error('Completed with errors on one or more workflows (see ERROR lines above). Any clean results were still collected and written.');
      }
    })
    .catch((err) => {
      console.error(err);
      if (process.env.GITHUB_OUTPUT) {
        fs.appendFileSync(process.env.GITHUB_OUTPUT, 'test_runtime_had_errors=true\n');
      }
      process.exit(1);
    });
}
