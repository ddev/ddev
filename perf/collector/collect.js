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

async function buildkiteApi(url) {
  const token = process.env.BUILDKITE_API_TOKEN;
  if (!token) throw new Error('BUILDKITE_API_TOKEN must be set');
  const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } });
  if (!res.ok) throw new Error(`Buildkite API error ${res.status} for ${url}`);
  return res.json();
}

async function latestBuildkiteResults(pipeline) {
  const org = CONFIG.buildkiteOrg;
  const builds = await buildkiteApi(
    `https://api.buildkite.com/v2/organizations/${org}/pipelines/${pipeline}/builds?per_page=1&state=passed`
  );
  if (!builds.length) {
    console.warn(`No passed builds found for pipeline ${pipeline}`);
    return [];
  }
  const build = builds[0];

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

  const token = process.env.BUILDKITE_API_TOKEN;
  const results = [];
  for (const artifact of matches) {
    const res = await fetch(artifact.download_url, { headers: { Authorization: `Bearer ${token}` } });
    if (!res.ok) throw new Error(`Failed to download artifact ${artifact.filename} for ${pipeline}: ${res.status}`);
    results.push(JSON.parse(await res.text()));
  }
  return results;
}

function latestLinuxResult() {
  const repo = process.env.GITHUB_REPOSITORY;
  if (!repo) throw new Error('GITHUB_REPOSITORY must be set');
  const runsJson = execFileSync(
    'gh',
    ['run', 'list', '--repo', repo, '--workflow', 'perf-linux.yml', '--status', 'success', '--limit', '1', '--json', 'databaseId'],
    { encoding: 'utf8' }
  );
  const runs = JSON.parse(runsJson);
  if (!runs.length) {
    console.warn('No successful perf-linux.yml runs found');
    return null;
  }
  const runId = runs[0].databaseId;
  const tmpDir = fs.mkdtempSync('/tmp/ddev-perf-linux-');
  execFileSync('gh', [
    'run', 'download', String(runId),
    '--repo', repo,
    '-n', 'perf-result-linux-docker-ce',
    '-D', tmpDir,
  ]);
  return JSON.parse(fs.readFileSync(path.join(tmpDir, 'perf-result.json'), 'utf8'));
}

async function main() {
  const results = [];

  for (const pipeline of CONFIG.pipelines) {
    try {
      results.push(...(await latestBuildkiteResults(pipeline)));
    } catch (err) {
      console.warn(`Skipping ${pipeline}: ${err.message}`);
    }
  }

  try {
    const linuxResult = latestLinuxResult();
    if (linuxResult) results.push(linuxResult);
  } catch (err) {
    console.warn(`Skipping Linux: ${err.message}`);
  }

  if (!results.length) {
    console.warn('No results collected this run; leaving history.ndjson unchanged');
    return;
  }

  const lines = results.map((r) => JSON.stringify(r));
  fs.appendFileSync(HISTORY_PATH, lines.join('\n') + '\n');
  console.log(`Appended ${lines.length} result(s) to ${HISTORY_PATH}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
