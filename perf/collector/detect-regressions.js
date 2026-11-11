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

// Collapses a leg's rows into one point per UTC date for a metric, so the
// windows below count nights even when a leg records more than one run a night.
function nightly(legRows, metric) {
  const byDate = new Map();
  for (const row of legRows) {
    if (typeof row.metrics[metric] !== 'number') continue;
    const date = row.timestamp.slice(0, 10);
    if (!byDate.has(date)) byDate.set(date, []);
    byDate.get(date).push(row);
  }
  return [...byDate].map(([date, dayRows]) => ({
    date,
    value: median(dayRows.map((r) => r.metrics[metric])),
    runners: [...new Set(dayRows.map((r) => r.runner).filter(Boolean))],
    first: dayRows[0],
    last: dayRows[dayRows.length - 1],
  }));
}

// Returns [{leg, metrics: [{metric, baseline, threshold, recent}], lastGood, firstBad}]
// for every leg whose last RECENT_NIGHTS nights of some metric all exceed the
// threshold derived from the BASELINE_NIGHTS nights before them.
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
      const nights = nightly(legRows, metric);
      if (nights.length < RECENT_NIGHTS + BASELINE_NIGHTS) continue;
      const recent = nights.slice(-RECENT_NIGHTS);
      const base = nights.slice(-RECENT_NIGHTS - BASELINE_NIGHTS, -RECENT_NIGHTS);
      // A metric that stopped reporting (e.g. mutagen_settle_s turned null)
      // must not alert on its stale tail.
      if (recent[recent.length - 1].last !== newest) continue;

      const baseValues = base.map((n) => n.value);
      const baseline = median(baseValues);
      const sigma = 1.4826 * median(baseValues.map((v) => Math.abs(v - baseline)));
      const threshold = Math.max(baseline * MIN_RATIO, baseline + MIN_SECONDS, baseline + MAD_SIGMAS * sigma);
      const recentValues = recent.map((n) => n.value);
      if (!recentValues.every((v) => v > threshold)) continue;

      regressed.push({
        metric,
        baseline,
        threshold,
        recent: recentValues,
        recentRunners: recent.map((n) => n.runners),
        baseRunners: base.flatMap((n) => n.runners),
      });
      // The slowdown may have begun before the recent window, with a single
      // noisy fast night breaking the streak, so walk back past slow nights
      // and past any lone fast night that has a slow night before it.
      const slow = (j) => j >= 0 && nights[j].value > threshold;
      let i = nights.length - RECENT_NIGHTS;
      while (slow(i - 1) || slow(i - 2)) i -= slow(i - 1) ? 1 : 2;
      const metricFirstBad = nights[i].first;
      if (!firstBad || metricFirstBad.timestamp < firstBad.timestamp) {
        firstBad = metricFirstBad;
        lastGood = i > 0 ? nights[i - 1].last : null;
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

// "201.1s" or, when the night recorded where it ran, "201.1s on tb-wsl-12-1".
function fmtNight(value, runners) {
  return runners && runners.length ? `${fmt(value)} on ${runners.join('+')}` : fmt(value);
}

// "tb-wsl-14-1 ×8, tb-wsl-12-1 ×6", most frequent first; '' when none recorded.
function runnerCounts(runners) {
  const counts = new Map();
  for (const r of runners) counts.set(r, (counts.get(r) || 0) + 1);
  return [...counts]
    .sort((a, b) => b[1] - a[1] || (a[0] < b[0] ? -1 : 1))
    .map(([r, n]) => `${r} ×${n}`)
    .join(', ');
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
      const nights = m.recent.map((v, i) => fmtNight(v, m.recentRunners && m.recentRunners[i])).join(', ');
      return `| \`${m.metric}\` | ${fmt(m.baseline)} | ${fmt(m.threshold)} | ${nights} (+${pct}%) |`;
    }),
    '',
  ];
  // A leg's metrics come from the same nightly runs, so the first regressed
  // metric's runners stand for all of them.
  const baseRunners = runnerCounts(metrics[0].baseRunners || []);
  if (baseRunners) {
    lines.push(
      `The baseline nights ran on ${baseRunners}. If the last nights all ran on one machine and the baseline mostly on another, compare against that machine's own history on the dashboard (runner filter) before looking for a code change.`,
      ''
    );
  }
  const goodSha = lastGood && lastGood.commit_sha;
  const badSha = firstBad.commit_sha;
  const badDate = firstBad.timestamp.slice(0, 10);
  if (!lastGood) {
    lines.push(`Every night in the history is above the threshold back to ${badDate}, so there is no last good night to compare against.`);
  } else if (!goodSha || !badSha) {
    lines.push(
      `Last good night ${lastGood.timestamp.slice(0, 10)} (\`${shortSha(goodSha)}\`), first slow night ${badDate} (\`${shortSha(badSha)}\`): ` +
        'a commit is missing from the history, so no commit range is available.'
    );
  } else if (goodSha !== badSha) {
    lines.push(
      `Last good night ${lastGood.timestamp.slice(0, 10)} (\`${shortSha(goodSha)}\`), first slow night ${badDate} (\`${shortSha(badSha)}\`): ` +
        `[commits in between](https://github.com/${repo}/compare/${goodSha}...${badSha}).`
    );
  } else {
    lines.push(
      `Last good night ${lastGood.timestamp.slice(0, 10)} and first slow night ${badDate} ran the same commit (\`${shortSha(badSha)}\`), so DDEV code did not change: look at the machine, its Docker provider, or upstream images.`
    );
  }
  lines.push(
    '',
    'Also check other legs at the same dates on the [dashboard](https://ddev.github.io/ddev/perf/): a code change usually shows on several legs, a machine change on one.',
    '',
    `Close this issue once it is explained or fixed. The nightly check then files nothing more for these metrics on this leg until a new slowdown begins; while the issue is open, it comments here only when another metric on this leg regresses.`,
    `<!-- perf-regression-metrics: ${metrics.map((m) => m.metric).join(',')} -->`
  );
  return lines.join('\n');
}

function gh(args) {
  return execFileSync('gh', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] });
}

function metricsIn(issue) {
  const known = new Set();
  for (const text of [issue.body, ...issue.comments.map((c) => c.body)]) {
    const m = /<!-- perf-regression-metrics: ([^ ]*) -->/.exec(text || '');
    if (m) m[1].split(',').forEach((name) => known.add(name));
  }
  return known;
}

function fileIssues(findings, repo) {
  gh(['label', 'create', LABEL, '--repo', repo, '--force', '--color', 'd93f0b',
    '--description', 'Opened by perf-collect.yml when a nightly benchmark regresses']);
  const issues = JSON.parse(gh(['issue', 'list', '--repo', repo, '--label', LABEL, '--state', 'all',
    '--limit', '100', '--json', 'number,title,body,comments,state,closedAt']));

  for (const finding of findings) {
    const title = issueTitle(finding.leg);
    const body = report(finding, repo);
    const sameLeg = issues.filter((i) => i.title === title);
    const open = sameLeg.find((i) => i.state === 'OPEN');
    // An issue closed after this slowdown began was closed with it in view,
    // so it still covers it; one closed earlier was about a different one.
    const closedDuring = (i) => i.state === 'CLOSED' && Date.parse(i.closedAt) >= Date.parse(finding.firstBad.timestamp);
    const known = new Set();
    for (const issue of sameLeg.filter((i) => i === open || closedDuring(i))) {
      metricsIn(issue).forEach((name) => known.add(name));
    }
    const added = finding.metrics.filter((m) => !known.has(m.metric));
    if (!added.length) {
      console.log(`An open issue, or one closed since this slowdown began, already covers ${finding.leg}; not filing`);
      continue;
    }
    if (open) {
      gh(['issue', 'comment', String(open.number), '--repo', repo, '--body', body]);
      console.log(`Commented on #${open.number} for ${added.map((m) => m.metric).join(', ')}`);
    } else {
      const url = gh(['issue', 'create', '--repo', repo, '--title', title, '--label', LABEL, '--body', body]).trim();
      console.log(`Opened ${url}`);
    }
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

module.exports = { detect, report, fileIssues };
