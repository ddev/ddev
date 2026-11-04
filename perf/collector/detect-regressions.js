#!/usr/bin/env node
// Scans history.ndjson for sustained per-leg regressions in the nightly perf
// benchmarks and opens (or comments on) one GitHub issue per regressed leg,
// labeled perf-regression. Run by .github/workflows/perf-collect.yml after
// collect.js. See "Regression signal" in perf/collector/README.md for the rule.
//
// Usage: node perf/collector/detect-regressions.js <history.ndjson> [--dry-run]
// --dry-run prints what it would file instead of calling `gh`.

const fs = require('fs');
const { execFileSync } = require('child_process');

const RECENT_NIGHTS = 3;
const BASELINE_NIGHTS = 14;
const MIN_RATIO = 1.2;
const MIN_SECONDS = 2;
const MAD_SIGMAS = 3;
// A leg whose newest point is older than this is not running; its last
// points say nothing about the current code.
const MAX_AGE_DAYS = 3;
const LABEL = 'perf-regression';

function median(values) {
  const s = [...values].sort((a, b) => a - b);
  const n = s.length;
  if (!n) return null;
  return n % 2 ? s[(n - 1) / 2] : (s[n / 2 - 1] + s[n / 2]) / 2;
}

function legKey(row) {
  return `${row.os}/${row.arch}/${row.docker_provider}`;
}

function readHistory(path) {
  const rows = [];
  for (const line of fs.readFileSync(path, 'utf8').split('\n')) {
    if (!line.trim()) continue;
    try {
      rows.push(JSON.parse(line));
    } catch (err) {
      console.warn(`Ignoring unparseable history line: ${err.message}`);
    }
  }
  return rows;
}

// Returns [{leg, metrics: [{metric, baseline, threshold, recent}], lastGood, firstBad}]
// for every leg whose last RECENT_NIGHTS points of some metric all exceed the
// threshold derived from the BASELINE_NIGHTS points before them.
function detect(rows, now = Date.now()) {
  const byLeg = new Map();
  for (const row of rows) {
    if (!row.metrics || !row.timestamp) continue;
    const key = legKey(row);
    if (!byLeg.has(key)) byLeg.set(key, []);
    byLeg.get(key).push(row);
  }

  const findings = [];
  for (const [leg, legRows] of [...byLeg].sort()) {
    legRows.sort((a, b) => (a.timestamp < b.timestamp ? -1 : a.timestamp > b.timestamp ? 1 : 0));
    const newest = legRows[legRows.length - 1];
    if (now - Date.parse(newest.timestamp) > MAX_AGE_DAYS * 86400e3) continue;

    const metricNames = new Set(legRows.flatMap((r) => Object.keys(r.metrics)));
    const regressed = [];
    let lastGood = null;
    let firstBad = null;
    for (const metric of [...metricNames].sort()) {
      const points = legRows.filter((r) => typeof r.metrics[metric] === 'number');
      if (points.length < RECENT_NIGHTS + BASELINE_NIGHTS) continue;
      const recent = points.slice(-RECENT_NIGHTS);
      const base = points.slice(-RECENT_NIGHTS - BASELINE_NIGHTS, -RECENT_NIGHTS);
      // A metric that stopped reporting (e.g. mutagen_settle_s turned null)
      // must not alert on its stale tail.
      if (recent[recent.length - 1] !== newest) continue;

      const baseValues = base.map((r) => r.metrics[metric]);
      const baseline = median(baseValues);
      const sigma = 1.4826 * median(baseValues.map((v) => Math.abs(v - baseline)));
      const threshold = Math.max(baseline * MIN_RATIO, baseline + MIN_SECONDS, baseline + MAD_SIGMAS * sigma);
      const recentValues = recent.map((r) => r.metrics[metric]);
      if (!recentValues.every((v) => v > threshold)) continue;

      regressed.push({ metric, baseline, threshold, recent: recentValues });
      if (!firstBad || recent[0].timestamp < firstBad.timestamp) {
        firstBad = recent[0];
        lastGood = base[base.length - 1];
      }
    }
    if (regressed.length) findings.push({ leg, metrics: regressed, lastGood, firstBad });
  }
  return findings;
}

function fmt(seconds) {
  return `${seconds.toFixed(1)}s`;
}

function shortSha(sha) {
  return sha ? sha.slice(0, 10) : 'unknown';
}

function issueTitle(leg) {
  return `Nightly perf regression: ${leg}`;
}

function report(finding, repo) {
  const { leg, metrics, lastGood, firstBad } = finding;
  const lines = [
    `The nightly benchmark on \`${leg}\` has been slower for ${RECENT_NIGHTS} nights in a row than the median of the ${BASELINE_NIGHTS} nights before.`,
    '',
    '| Metric | Baseline median | Threshold | Last nights |',
    '|---|---|---|---|',
    ...metrics.map((m) => {
      const pct = ((median(m.recent) / m.baseline - 1) * 100).toFixed(0);
      return `| \`${m.metric}\` | ${fmt(m.baseline)} | ${fmt(m.threshold)} | ${m.recent.map(fmt).join(', ')} (+${pct}%) |`;
    }),
    '',
  ];
  if (lastGood.commit_sha && firstBad.commit_sha && lastGood.commit_sha !== firstBad.commit_sha) {
    lines.push(
      `Last good night ${lastGood.timestamp.slice(0, 10)} (\`${shortSha(lastGood.commit_sha)}\`), first slow night ${firstBad.timestamp.slice(0, 10)} (\`${shortSha(firstBad.commit_sha)}\`): ` +
        `[commits in between](https://github.com/${repo}/compare/${lastGood.commit_sha}...${firstBad.commit_sha}).`
    );
  } else {
    lines.push(
      `Last good night ${lastGood.timestamp.slice(0, 10)} and first slow night ${firstBad.timestamp.slice(0, 10)} ran the same commit (\`${shortSha(firstBad.commit_sha)}\`), so DDEV code did not change: look at the machine, its Docker provider, or upstream images.`
    );
  }
  lines.push(
    '',
    'Also check other legs at the same dates on the [dashboard](https://ddev.github.io/ddev/perf/): a code change usually shows on several legs, a machine change on one.',
    '',
    `Close this issue once it is explained or fixed. While it is open, the nightly check comments here only when another metric on this leg regresses.`,
    `<!-- perf-regression-metrics: ${metrics.map((m) => m.metric).join(',')} -->`
  );
  return lines.join('\n');
}

function gh(args) {
  return execFileSync('gh', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] });
}

function fileIssues(findings, repo) {
  gh(['label', 'create', LABEL, '--repo', repo, '--force', '--color', 'd93f0b',
    '--description', 'Opened by perf-collect.yml when a nightly benchmark regresses']);
  const open = JSON.parse(gh(['issue', 'list', '--repo', repo, '--label', LABEL, '--state', 'open',
    '--limit', '100', '--json', 'number,title,body,comments']));

  for (const finding of findings) {
    const title = issueTitle(finding.leg);
    const body = report(finding, repo);
    const existing = open.find((i) => i.title === title);
    if (!existing) {
      const url = gh(['issue', 'create', '--repo', repo, '--title', title, '--label', LABEL, '--body', body]).trim();
      console.log(`Opened ${url}`);
      continue;
    }
    const known = new Set();
    for (const text of [existing.body, ...existing.comments.map((c) => c.body)]) {
      const m = /<!-- perf-regression-metrics: ([^ ]*) -->/.exec(text || '');
      if (m) m[1].split(',').forEach((name) => known.add(name));
    }
    const added = finding.metrics.filter((m) => !known.has(m.metric));
    if (!added.length) {
      console.log(`#${existing.number} already covers ${finding.leg}; not commenting`);
      continue;
    }
    gh(['issue', 'comment', String(existing.number), '--repo', repo, '--body', body]);
    console.log(`Commented on #${existing.number} for ${added.map((m) => m.metric).join(', ')}`);
  }
}

function main() {
  const args = process.argv.slice(2);
  const dryRun = args.includes('--dry-run');
  const historyPath = args.find((a) => !a.startsWith('--'));
  if (!historyPath) {
    console.error('Usage: detect-regressions.js <history.ndjson> [--dry-run]');
    process.exit(1);
  }
  const repo = process.env.GITHUB_REPOSITORY || 'ddev/ddev';
  const findings = detect(readHistory(historyPath));

  const summary = findings.length
    ? findings.map((f) => `### ${issueTitle(f.leg)}\n\n${report(f, repo)}`).join('\n\n')
    : 'No sustained perf regressions detected.';
  console.log(summary);
  if (process.env.GITHUB_STEP_SUMMARY) {
    fs.appendFileSync(process.env.GITHUB_STEP_SUMMARY, `## Perf regression check\n\n${summary}\n`);
  }

  if (findings.length && !dryRun) fileIssues(findings, repo);
}

if (require.main === module) {
  main();
}

module.exports = { detect, report };
