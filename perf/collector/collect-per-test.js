#!/usr/bin/env node
// Second pass over test-runtime-history.ndjson: for each run/build not yet
// processed, downloads its gotestsum per-test JSON artifact (written when
// GOTESTSUM_JSONFILE_DIR is set -- see Makefile, test.sh, test-reusable.yml)
// and appends one row per slow-or-failing top-level Go test to
// per-test-history.ndjson. Turns the per-test detail gotestsum already
// writes -- today only a 90-day build artifact -- into a lasting dataset, so
// a specific test's duration can be tracked as a trend instead of read off
// one run's artifact by hand. See perf/collector/README.md.
//
// Required env: same as collect-test-runtime.js (GITHUB_TOKEN,
// GITHUB_REPOSITORY, and BUILDKITE_API_TOKEN -- if unset, Buildkite rows in
// test-runtime-history.ndjson are skipped, not an error, same tolerance).
//
// Usage: node perf/collector/collect-per-test.js <path-to-test-runtime-history.ndjson> <path-to-per-test-history.ndjson> <path-to-cursor.json> [--since=ISO8601]
//
// --since is normally omitted: derived from cursor.json's
// processedThroughTimestamp, written at the end of every run. A run/build
// that produces no qualifying test row (see MIN_ELAPSED_S below) would
// otherwise leave no trace in per-test-history.ndjson and get reprocessed
// forever -- the cursor file tracks "attempted," not just "found something."

const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const args = process.argv.slice(2);
const positional = args.filter((a) => !a.startsWith('--'));
const [RUNTIME_HISTORY_PATH, PER_TEST_HISTORY_PATH, CURSOR_PATH] = positional;
const sinceArg = args.find((a) => a.startsWith('--since='));

let hadErrors = false;

const DEFAULT_BACKFILL_DAYS = 7;

// A top-level Go test suite is ~600 tests; most run in well under a second.
// Recording every test on every run would grow this dataset by orders of
// magnitude faster than test-runtime-history.ndjson for little benefit --
// the tests worth trend-tracking are the slow ones (#8696's whole premise)
// and any that fail, regardless of how fast they failed.
const MIN_ELAPSED_S = 5;

// Bounds file growth over time for a client-side-parsed dashboard. Double
// the 90-day retention gotestsum's raw build artifacts already get, so this
// is a real extension of that window, not a promise of true permanence --
// revisit if a longer horizon turns out to matter more than dashboard load
// time.
const RETENTION_DAYS = 180;

function readNdjson(filePath) {
  if (!fs.existsSync(filePath)) return [];
  const rows = [];
  for (const line of fs.readFileSync(filePath, 'utf8').split('\n')) {
    if (!line.trim()) continue;
    try {
      rows.push(JSON.parse(line));
    } catch (err) {
      console.warn(`Ignoring unparseable line in ${filePath}: ${err.message}`);
    }
  }
  return rows;
}

function computeSince() {
  if (sinceArg) return sinceArg.slice('--since='.length);
  if (!fs.existsSync(CURSOR_PATH)) {
    return new Date(Date.now() - DEFAULT_BACKFILL_DAYS * 86400 * 1000).toISOString();
  }
  const cursor = JSON.parse(fs.readFileSync(CURSOR_PATH, 'utf8'));
  return cursor.processedThroughTimestamp;
}

// gotestsum's --jsonfile is Go's test2json format: one JSON object per line,
// per test action (run/pause/cont/output/pass/fail/skip). Only the terminal
// pass/fail/skip event at the top level carries the test's total Elapsed
// seconds -- subtests (Test containing "/") are excluded here, matching the
// same top-level-only scope #8696 used, to keep cardinality manageable.
function extractTestEvents(ndjsonText) {
  const events = [];
  for (const line of ndjsonText.split('\n')) {
    if (!line.trim()) continue;
    let event;
    try {
      event = JSON.parse(line);
    } catch {
      continue; // gotestsum's jsonfile is one object per line; skip a corrupt one
    }
    if (!event.Test || event.Test.includes('/')) continue;
    if (!['pass', 'fail', 'skip'].includes(event.Action)) continue;
    events.push({ package: event.Package, test: event.Test, elapsed_s: event.Elapsed || 0, outcome: event.Action });
  }
  return events;
}

module.exports = { extractTestEvents, MIN_ELAPSED_S, RETENTION_DAYS };

function qualifies(event) {
  return event.outcome === 'fail' || event.elapsed_s >= MIN_ELAPSED_S;
}

function ghApiPages(urlPath, params) {
  const repo = process.env.GITHUB_REPOSITORY;
  const fArgs = params.flatMap((p) => ['-f', p]);
  const out = execFileSync(
    'gh',
    ['api', '--method', 'GET', '--paginate', '--slurp', `repos/${repo}${urlPath}`, ...fArgs],
    { encoding: 'utf8', maxBuffer: 1024 * 1024 * 64 }
  );
  return JSON.parse(out);
}

// Multiple artifacts can share this prefix -- podman-rootless and
// podman-rootless-mutagen invoke the reusable workflow twice per run, once
// per make_target, each uploading its own artifact (see
// collect-test-runtime.js's stage_analysis comment for why).
function githubTestEvents(row) {
  const repo = process.env.GITHUB_REPOSITORY;
  let artifacts;
  try {
    artifacts = ghApiPages(`/actions/runs/${row.run_id}/artifacts`, ['per_page=100']).flatMap((p) => p.artifacts || []);
  } catch (err) {
    console.warn(`  Could not list artifacts for run ${row.run_id}: ${err.message}`);
    return [];
  }
  const prefix = `test-results-${row.run_id}-${row.run_attempt}-`;
  const matches = artifacts.filter((a) => a.name.startsWith(prefix));
  const rows = [];
  for (const artifact of matches) {
    const tmpDir = fs.mkdtempSync('/tmp/ddev-per-test-');
    try {
      execFileSync('gh', ['run', 'download', String(row.run_id), '--repo', repo, '-n', artifact.name, '-D', tmpDir]);
    } catch (err) {
      console.warn(`  Could not download artifact ${artifact.name} for run ${row.run_id}: ${err.message}`);
      continue;
    }
    for (const file of fs.readdirSync(tmpDir).filter((f) => f.endsWith('.ndjson'))) {
      const makeTarget = path.basename(file, '.ndjson');
      const text = fs.readFileSync(path.join(tmpDir, file), 'utf8');
      for (const event of extractTestEvents(text).filter(qualifies)) {
        rows.push({ ...event, make_target: makeTarget });
      }
    }
  }
  return rows;
}

// A missing/invalid token surfaces as a 401 here, caught by
// buildkiteTestEvents' own try/catch below -- same tolerance as
// collect-test-runtime.js: Buildkite rows just yield no per-test data until
// the token exists, without failing the run or blocking the cursor.
async function buildkiteApi(url) {
  const token = process.env.BUILDKITE_API_TOKEN;
  const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
  if (!res.ok) throw new Error(`Buildkite API error ${res.status} for ${url}`);
  return res.json();
}

// gotestsum's known output basenames (see Makefile's MAKE_TARGET_STAGES),
// matched on filename alone -- Buildkite's artifact `path` preserves the
// test-results/ subdirectory `buildkite-agent artifact upload` was given,
// but `filename` doesn't, so matching filename is robust either way.
const KNOWN_JSONFILE_NAMES = ['testnotddevapp.ndjson', 'testddevapp.ndjson', 'testcmd.ndjson'];

async function buildkiteTestEvents(row) {
  const org = require(path.join(__dirname, 'test-pipelines.json')).buildkiteOrg;
  let artifacts;
  try {
    artifacts = await buildkiteApi(
      `https://api.buildkite.com/v2/organizations/${org}/pipelines/${row.workflow}/builds/${row.run_id}/artifacts`
    );
  } catch (err) {
    console.warn(`  Could not list artifacts for ${row.workflow} build ${row.run_id}: ${err.message}`);
    return [];
  }
  const matches = artifacts.filter((a) => KNOWN_JSONFILE_NAMES.includes(a.filename));
  const rows = [];
  const token = process.env.BUILDKITE_API_TOKEN;
  for (const artifact of matches) {
    try {
      const res = await fetch(artifact.download_url, { headers: { Authorization: `Bearer ${token}` } });
      if (!res.ok) throw new Error(`download failed: HTTP ${res.status}`);
      const text = await res.text();
      const makeTarget = path.basename(artifact.filename, '.ndjson');
      for (const event of extractTestEvents(text).filter(qualifies)) {
        rows.push({ ...event, make_target: makeTarget });
      }
    } catch (err) {
      console.warn(`  Could not download ${artifact.filename} for ${row.workflow} build ${row.run_id}: ${err.message}`);
    }
  }
  return rows;
}

function pointKey(row) {
  return `${row.source}:${row.workflow}:${row.run_id}|${row.run_attempt}:${row.package}/${row.test}`;
}

async function main() {
  const since = computeSince();
  const runtimeRows = readNdjson(RUNTIME_HISTORY_PATH)
    .filter((r) => r.timestamp > since)
    .sort((a, b) => (a.timestamp < b.timestamp ? -1 : 1));

  if (!runtimeRows.length) {
    console.warn(`No test-runtime-history rows newer than ${since}; nothing to process`);
    return;
  }

  const seen = new Set(readNdjson(PER_TEST_HISTORY_PATH).map(pointKey));
  const fresh = [];
  let latestProcessed = since;

  for (const runtimeRow of runtimeRows) {
    // Cancelled runs and skip-tagged commits never really ran the suite --
    // same exclusion test-runtime-history.ndjson's own "only successful full
    // runs" dashboard filter already applies, see README.md.
    if (runtimeRow.conclusion === 'cancelled' || runtimeRow.skip_marker) {
      latestProcessed = runtimeRow.timestamp;
      continue;
    }

    console.log(`Processing ${runtimeRow.source} ${runtimeRow.workflow} ${runtimeRow.run_id}#${runtimeRow.run_attempt}...`);
    let events;
    try {
      events = runtimeRow.source === 'github' ? githubTestEvents(runtimeRow) : await buildkiteTestEvents(runtimeRow);
    } catch (err) {
      console.error(`ERROR extracting per-test events for ${runtimeRow.source} ${runtimeRow.workflow} ${runtimeRow.run_id}: ${err.message}`);
      hadErrors = true;
      continue; // don't advance latestProcessed past a run we failed to read -- retry it next time
    }

    for (const event of events) {
      const row = {
        timestamp: runtimeRow.timestamp,
        source: runtimeRow.source,
        workflow: runtimeRow.workflow,
        run_id: runtimeRow.run_id,
        run_attempt: runtimeRow.run_attempt,
        make_target: event.make_target,
        package: event.package,
        test: event.test,
        elapsed_s: event.elapsed_s,
        outcome: event.outcome,
      };
      const key = pointKey(row);
      if (seen.has(key)) continue;
      seen.add(key);
      fresh.push(row);
    }
    latestProcessed = runtimeRow.timestamp;
  }

  if (fresh.length) {
    fs.appendFileSync(PER_TEST_HISTORY_PATH, fresh.map((r) => JSON.stringify(r)).join('\n') + '\n');
    console.log(`Appended ${fresh.length} per-test row(s) to ${PER_TEST_HISTORY_PATH}`);
  } else {
    console.log('No qualifying per-test rows found in this batch');
  }

  const cutoff = new Date(Date.now() - RETENTION_DAYS * 86400 * 1000).toISOString();
  const allRows = readNdjson(PER_TEST_HISTORY_PATH);
  const kept = allRows.filter((r) => r.timestamp >= cutoff);
  if (kept.length < allRows.length) {
    fs.writeFileSync(PER_TEST_HISTORY_PATH, kept.map((r) => JSON.stringify(r)).join('\n') + (kept.length ? '\n' : ''));
    console.log(`Pruned ${allRows.length - kept.length} row(s) older than ${RETENTION_DAYS} days`);
  }

  fs.writeFileSync(CURSOR_PATH, JSON.stringify({ processedThroughTimestamp: latestProcessed }, null, 2) + '\n');
}

if (require.main === module) {
  if (!RUNTIME_HISTORY_PATH || !PER_TEST_HISTORY_PATH || !CURSOR_PATH) {
    console.error('Usage: collect-per-test.js <test-runtime-history.ndjson> <per-test-history.ndjson> <cursor.json> [--since=ISO8601]');
    process.exit(1);
  }

  main()
    .then(() => {
      if (process.env.GITHUB_OUTPUT) {
        fs.appendFileSync(process.env.GITHUB_OUTPUT, `per_test_had_errors=${hadErrors}\n`);
      }
      if (hadErrors) {
        console.error('Completed with errors on one or more runs (see ERROR lines above). Any clean results were still collected and written.');
      }
    })
    .catch((err) => {
      console.error(err);
      if (process.env.GITHUB_OUTPUT) {
        fs.appendFileSync(process.env.GITHUB_OUTPUT, 'per_test_had_errors=true\n');
      }
      process.exit(1);
    });
}
