#!/usr/bin/env node
// Pulls every perf-result*.json artifact from the latest passed build of each
// Buildkite pipeline in pipelines.json, plus the latest successful Linux GH
// Actions perf run, and appends one line per leg to history.ndjson. Run by
// .github/workflows/perf-collect.yml on a schedule, offset later than the
// perf-* jobs so they've had time to finish. See perf/README.md.
//
// Required env:
//   BUILDKITE_API_TOKEN - a Buildkite API token with read access to builds/artifacts
//   GITHUB_TOKEN         - for `gh` CLI calls against this repo's Actions runs
//   GITHUB_REPOSITORY    - owner/repo, set automatically in GitHub Actions
//
// Usage: node perf/collector/collect.js <path-to-history.ndjson>

const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const HISTORY_PATH = process.argv[2];
if (!HISTORY_PATH) {
  console.error('Usage: collect.js <path-to-history.ndjson>');
  process.exit(1);
}

const CONFIG = JSON.parse(fs.readFileSync(path.join(__dirname, 'pipelines.json'), 'utf8'));

// Set whenever any single pipeline/artifact fails to collect, so the run can
// still finish, commit, and publish whatever DID come through clean -- one
// broken leg shouldn't blank out the other five -- while still reporting
// failure at the end (see main()) so a broken leg doesn't go unnoticed the
// way a fully-silent skip would.
let hadErrors = false;

async function buildkiteApi(url) {
  const token = process.env.BUILDKITE_API_TOKEN;
  if (!token) throw new Error('BUILDKITE_API_TOKEN must be set');
  console.log(`  GET ${url}`);
  const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
  if (!res.ok) throw new Error(`Buildkite API error ${res.status} for ${url}`);
  return res.json();
}

async function latestBuildkiteResults(pipeline) {
  console.log(`Checking pipeline ${pipeline} for its latest passed build...`);
  const org = CONFIG.buildkiteOrg;
  const builds = await buildkiteApi(
    `https://api.buildkite.com/v2/organizations/${org}/pipelines/${pipeline}/builds?per_page=1&state=passed`
  );
  if (!builds.length) {
    console.warn(`No passed builds found for pipeline ${pipeline}`);
    return [];
  }
  const build = builds[0];
  console.log(`  Latest passed build: #${build.number} (${build.web_url})`);

  const artifacts = await buildkiteApi(
    `https://api.buildkite.com/v2/organizations/${org}/pipelines/${pipeline}/builds/${build.number}/artifacts`
  );
  // A build can hold more than one perf-result*.json artifact -- e.g.
  // perf-macos-shared-providers.yml runs five Docker providers as separate
  // jobs within the same build, each uploading its own result. Collect every
  // match, not just the first, or four of the five legs get silently dropped.
  const matches = artifacts.filter((a) => /^perf-result.*\.json$/.test(a.filename));
  if (!matches.length) {
    console.warn(`No perf-result*.json artifact found on latest passed build of ${pipeline}`);
    return [];
  }
  console.log(`  Found ${matches.length} perf-result*.json artifact(s): ${matches.map((a) => a.filename).join(', ')}`);

  const token = process.env.BUILDKITE_API_TOKEN;
  const results = [];
  for (const artifact of matches) {
    // Caught per-artifact, not left to propagate: one bad upload in a build
    // that holds several (e.g. macos-shared-providers' 6 legs) must not
    // discard the other, valid ones.
    try {
      console.log(`  Downloading ${artifact.filename} from ${artifact.download_url}`);
      const res = await fetch(artifact.download_url, { headers: { Authorization: `Bearer ${token}` } });
      if (!res.ok) throw new Error(`download failed: HTTP ${res.status}`);
      const text = await res.text();
      try {
        results.push(JSON.parse(text));
      } catch (err) {
        // Bare JSON.parse errors don't say which artifact failed, what it
        // actually contained, or where it came from -- all three are needed
        // to tell "wrong scope/HTML error page" apart from "a script wrote
        // non-JSON to stdout ahead of the result line" (the actual cause the
        // first time this fired) without pulling raw Buildkite logs by hand.
        throw new Error(
          `not valid JSON: ${err.message}\n    URL: ${artifact.download_url}\n    First 200 chars: ${text.slice(0, 200)}`
        );
      }
    } catch (err) {
      console.error(`ERROR collecting ${pipeline}'s ${artifact.filename}: ${err.message}`);
      hadErrors = true;
    }
  }
  return results;
}

function latestLinuxResult() {
  const repo = process.env.GITHUB_REPOSITORY;
  if (!repo) throw new Error('GITHUB_REPOSITORY must be set');
  console.log('Checking perf-linux.yml for its latest successful run...');
  const runsJson = execFileSync(
    'gh',
    ['run', 'list', '--repo', repo, '--workflow', 'perf-linux.yml', '--status', 'success', '--limit', '1', '--json', 'databaseId,url'],
    { encoding: 'utf8' }
  );
  const runs = JSON.parse(runsJson);
  if (!runs.length) {
    console.warn('No successful perf-linux.yml runs found');
    return null;
  }
  const runId = runs[0].databaseId;
  console.log(`  Latest successful run: ${runs[0].url}`);
  const tmpDir = fs.mkdtempSync('/tmp/ddev-perf-linux-');
  execFileSync('gh', [
    'run', 'download', String(runId),
    '--repo', repo,
    '-n', 'perf-result-linux-docker-ce',
    '-D', tmpDir,
  ]);
  const resultPath = path.join(tmpDir, 'perf-result.json');
  const text = fs.readFileSync(resultPath, 'utf8');
  try {
    return JSON.parse(text);
  } catch (err) {
    throw new Error(
      `perf-result-linux-docker-ce from ${runs[0].url} is not valid JSON: ${err.message}\n  First 200 chars: ${text.slice(0, 200)}`
    );
  }
}

// Identity of one data point, matching dashboard.html's legKey() plus the run's
// timestamp. Used to dedup: each source below hands back the latest *passed*
// build, so a leg that failed (or didn't run) tonight yields last night's record
// again, unchanged. Appending it blind would double-count that measurement in
// the trend line, the trailing median, and the regression flag.
function pointKey(row) {
  return `${row.timestamp}|${row.os}/${row.arch}/${row.docker_provider}`;
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

async function main() {
  const results = [];

  // A missing token is the expected, tolerated state before the one-time
  // BUILDKITE_API_TOKEN vault setup (see this file's header comment) --
  // skip Buildkite entirely and still collect the Linux leg below, without
  // counting it as an error. Once a token is present, a request failure
  // (wrong scope, revoked, rate-limited) means the setup is broken, not "not
  // done yet" -- but one pipeline failing that way must not discard the
  // other pipelines' perfectly good results, so it's caught here (per
  // pipeline) the same way latestBuildkiteResults() catches per artifact:
  // logged loudly, counted in hadErrors, and the loop moves on.
  if (!process.env.BUILDKITE_API_TOKEN) {
    console.warn('BUILDKITE_API_TOKEN not set; skipping Buildkite pipelines');
  } else {
    for (const pipeline of CONFIG.pipelines) {
      try {
        results.push(...(await latestBuildkiteResults(pipeline)));
      } catch (err) {
        console.error(`ERROR collecting ${pipeline}: ${err.message}`);
        hadErrors = true;
      }
    }
  }

  try {
    const linuxResult = latestLinuxResult();
    if (linuxResult) results.push(linuxResult);
  } catch (err) {
    console.error(`ERROR collecting Linux: ${err.message}`);
    hadErrors = true;
  }

  if (!results.length) {
    console.warn('No results collected this run; leaving history.ndjson unchanged');
    return;
  }

  const seen = existingPointKeys(HISTORY_PATH);
  const fresh = [];
  for (const row of results) {
    const key = pointKey(row);
    if (seen.has(key)) {
      console.log(`Skipping already-recorded result: ${key}`);
      continue;
    }
    seen.add(key);
    fresh.push(row);
  }

  if (!fresh.length) {
    console.warn('All collected results were already recorded; leaving history.ndjson unchanged');
    return;
  }

  const lines = fresh.map((r) => JSON.stringify(r));
  fs.appendFileSync(HISTORY_PATH, lines.join('\n') + '\n');
  console.log(`Appended ${lines.length} new result(s) of ${results.length} collected to ${HISTORY_PATH}`);
}

main()
  .then(() => {
    // Exit 0 even when hadErrors, so the caller's later workflow steps
    // (commit/push, docs-publish trigger) still run and publish whatever
    // clean results came through -- one broken source shouldn't block the
    // others. Recorded via GITHUB_OUTPUT so a later step in the same
    // workflow can still fail the run visibly, AFTER publishing, instead of
    // this going unnoticed the way a fully-silent skip would.
    if (process.env.GITHUB_OUTPUT) {
      fs.appendFileSync(process.env.GITHUB_OUTPUT, `had_errors=${hadErrors}\n`);
    }
    if (hadErrors) {
      console.error('Completed with errors on one or more sources (see ERROR lines above). Any clean results were still collected and written; the workflow will report failure after publishing them.');
    }
  })
  .catch((err) => {
    // A genuine crash (malformed pipelines.json, a required env var missing)
    // reaches here, distinct from a per-source error, which main() already
    // caught and logged without letting it propagate. Nothing here is
    // trustworthy enough to publish, so fail immediately.
    console.error(err);
    if (process.env.GITHUB_OUTPUT) {
      fs.appendFileSync(process.env.GITHUB_OUTPUT, 'had_errors=true\n');
    }
    process.exit(1);
  });
