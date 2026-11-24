#!/usr/bin/env node
// Weekly report of Go tests and subtests that ran in no CI job on main,
// built from the last-ran index collect-per-test.js keeps
// (ci/tests/last-ran.json on the performance-data branch, see
// last-ran-index.js) and the test functions in the checkout. Run by
// .github/workflows/tests-not-run.yml, which turns the report into a single
// tracking issue. Lists:
//   - Skipped everywhere: tests and subtests whose only outcomes in the
//     window are skips.
//   - Never ran: top-level `func TestXxx(t *testing.T)` in any package `go
//     list ./...` finds (for linux, darwin and windows) with no outcome at all
//     in the window -- including whole packages no CI job compiles.
//   - Allowlist entries in .github/tests-not-run-in-ci.yml that match no test.
// Tests in the allowlist are left out, unless their entry sets window_days,
// which judges them over that longer window instead (for legs that run
// rarely).
//
// Usage: node perf/collector/report-tests-not-run.js --index=<last-ran.json>
//   [--allowlist=.github/tests-not-run-in-ci.yml] [--window-days=14]
//   [--blob-base=https://github.com/ddev/ddev/blob/<sha>] [--run-url=<url>]
//   [--out=report.md] [--now=ISO8601] [--max-stale-days=3]
//
// Writes the markdown report to --out (stdout if omitted) and, under GitHub
// Actions, `count` (entries in the report) and `status` (ok, or incomplete
// when the index does not yet cover the window) to GITHUB_OUTPUT. Exits
// non-zero only when its inputs are broken: an unreadable index or
// allowlist, an allowlist entry without a reason, or an index not updated in
// --max-stale-days (the nightly collector has stopped).

const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');
const { INDEX_RETENTION_DAYS, loadIndex } = require('./last-ran-index');

const DAY_MS = 86400 * 1000;
// GitHub rejects an issue body over 65536 characters; leave room for the
// header and footer.
const MAX_BODY_CHARS = 60000;
const GOOS_LIST = ['linux', 'darwin', 'windows'];

function parseArgs(argv) {
  const opts = {};
  for (const a of argv) {
    const m = /^--([\w-]+)=(.*)$/.exec(a);
    if (!m) throw new Error(`Unexpected argument: ${a}`);
    opts[m[1]] = m[2];
  }
  return opts;
}

// Top-level Go test functions with their file and line. testing only runs
// TestXxx(t *testing.T), so TestMain(m *testing.M) and helpers that take
// other parameters do not match.
const TEST_FUNC_RE = /^func (Test[A-Z0-9_]\w*|Test)\(\s*\w+\s+\*testing\.T\s*\)/;

function testFuncsInFile(absPath) {
  const out = [];
  const lines = fs.readFileSync(absPath, 'utf8').split('\n');
  lines.forEach((line, i) => {
    const m = TEST_FUNC_RE.exec(line);
    if (m) out.push({ name: m[1], line: i + 1 });
  });
  return out;
}

// Every package with test files under any GOOS CI covers, so a test that
// builds only on Windows or macOS is still known. Keyed by import path, the
// same name test2json reports as Package.
function listStaticTests(repoRoot) {
  const modulePath = execFileSync('go', ['list', '-m'], { cwd: repoRoot, encoding: 'utf8' }).trim();
  const packages = new Map(); // import path -> { dir, tests: Map(name -> {file, line}) }
  for (const goos of GOOS_LIST) {
    const out = execFileSync(
      'go',
      ['list', '-e', '-f', '{{.ImportPath}}\t{{.Dir}}\t{{join .TestGoFiles ","}}\t{{join .XTestGoFiles ","}}', './...'],
      { cwd: repoRoot, encoding: 'utf8', env: { ...process.env, GOOS: goos, CGO_ENABLED: '0' }, maxBuffer: 64 * 1024 * 1024 }
    );
    for (const line of out.split('\n')) {
      if (!line.trim()) continue;
      const [importPath, dir, tfiles, xfiles] = line.split('\t');
      const files = [...tfiles.split(','), ...xfiles.split(',')].filter(Boolean);
      if (!files.length) continue;
      const rel = path.relative(repoRoot, dir).split(path.sep).join('/');
      const pkg = packages.get(importPath) || { dir: rel, tests: new Map() };
      packages.set(importPath, pkg);
      for (const f of files) {
        for (const t of testFuncsInFile(path.join(dir, f))) {
          if (!pkg.tests.has(t.name)) pkg.tests.set(t.name, { file: `${rel}/${f}`, line: t.line });
        }
      }
    }
  }
  return { modulePath, packages };
}

function readAllowlist(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return [];
  let data;
  if (/\.ya?ml$/.test(filePath)) {
    // yq (mikefarah) is on GitHub's Ubuntu runners; Node has no YAML parser.
    data = JSON.parse(execFileSync('yq', ['-o=json', '.', filePath], { encoding: 'utf8' }));
  } else {
    data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  }
  const entries = (data && data.tests) || [];
  const problems = [];
  entries.forEach((e, i) => {
    const where = `${filePath} entry ${i + 1}${e && e.test ? ` (${e.test})` : ''}`;
    if (!e || typeof e.test !== 'string' || !/^Test/.test(e.test)) problems.push(`${where}: "test" must name a Go test`);
    if (!e || typeof e.reason !== 'string' || !e.reason.trim()) problems.push(`${where}: "reason" is required`);
    if (e && e.window_days !== undefined) {
      const w = Number(e.window_days);
      if (!Number.isInteger(w) || w < 1 || w >= INDEX_RETENTION_DAYS) {
        problems.push(`${where}: "window_days" must be a whole number of days below ${INDEX_RETENTION_DAYS}`);
      }
    }
  });
  if (problems.length) throw new Error(`Invalid allowlist:\n${problems.join('\n')}`);
  return entries;
}

// An entry names a test or subtest, optionally limited to one package (a
// directory relative to the repo root, like pkg/ddevapp). A top-level name
// also covers its subtests.
function allowlistMatch(entries, relPkg, test) {
  return entries.find(
    (e) => (!e.package || e.package === relPkg) && (test === e.test || test.startsWith(`${e.test}/`))
  );
}

function shortDate(ts) {
  return ts ? ts.slice(0, 10) : '?';
}

// Inline code that survives backticks in the text.
function code(text) {
  const t = String(text).replace(/\s+/g, ' ').trim();
  const fence = t.includes('`') ? '``' : '`';
  return `${fence}${fence === '``' ? ' ' : ''}${t}${fence === '``' ? ' ' : ''}${fence}`;
}

function buildReport(opts) {
  const now = opts.now ? Date.parse(opts.now) : Date.now();
  const windowDays = Number(opts['window-days'] || 14);
  if (!Number.isInteger(windowDays) || windowDays < 1 || windowDays >= INDEX_RETENTION_DAYS) {
    throw new Error(`--window-days must be a whole number of days below ${INDEX_RETENTION_DAYS}`);
  }
  const maxStaleDays = Number(opts['max-stale-days'] || 3);
  const repoRoot = path.resolve(opts['repo-root'] || '.');
  const blobBase = opts['blob-base'] || '';
  const index = loadIndex(opts.index);
  if (!index.updated) throw new Error(`${opts.index} holds no runs`);
  if (now - Date.parse(index.updated) > maxStaleDays * DAY_MS) {
    throw new Error(`${opts.index} was last updated ${index.updated}, over ${maxStaleDays} days ago; is perf-collect.yml running?`);
  }
  const allowlist = readAllowlist(opts.allowlist);
  const { modulePath, packages } = listStaticTests(repoRoot);
  const relPkg = (importPath) =>
    importPath === modulePath ? '.' : importPath.startsWith(`${modulePath}/`) ? importPath.slice(modulePath.length + 1) : importPath;

  const windowStartFor = (days) => new Date(now - days * DAY_MS).toISOString();
  const defaultStart = windowStartFor(windowDays);
  const covered = (start) => index.covers_since && index.covers_since <= start;
  // For each test, the window it is judged over: the default, or a longer
  // one from its allowlist entry. null leaves it out of the report.
  const judgeStart = (rel, test) => {
    const entry = allowlistMatch(allowlist, rel, test);
    if (!entry) return defaultStart;
    if (entry.window_days === undefined) return null;
    const start = windowStartFor(Number(entry.window_days));
    return covered(start) ? start : null; // not enough history yet to judge it
  };
  const link = (loc, label) => (loc && blobBase ? `[${label}](${blobBase}/${loc.file}#L${loc.line})` : label);

  const status = covered(defaultStart) ? 'ok' : 'incomplete';

  // Which packages and tests had any outcome inside their window.
  const seen = new Map(); // import path -> Set(top-level test names with any outcome)
  const skippedEverywhere = [];
  for (const [key, entry] of Object.entries(index.tests)) {
    const sp = key.indexOf(' ');
    const pkgPath = key.slice(0, sp);
    const test = key.slice(sp + 1);
    const rel = relPkg(pkgPath);
    const top = test.split('/')[0];
    const start = judgeStart(rel, test) || defaultStart;
    const ranIn = entry.ran && entry.ran.at >= start;
    const skippedIn = entry.skipped && entry.skipped.at >= start;
    if (ranIn || skippedIn) {
      if (!seen.has(pkgPath)) seen.set(pkgPath, new Set());
      seen.get(pkgPath).add(top);
    }
    if (status !== 'ok' || ranIn || !skippedIn) continue;
    if (!judgeStart(rel, test)) continue;
    const pkg = packages.get(pkgPath);
    const loc = pkg && pkg.tests.get(top);
    if (!loc) continue; // deleted or renamed since it last ran
    skippedEverywhere.push({ rel, test, loc, skipped: entry.skipped });
  }

  const neverRan = [];
  if (status === 'ok') {
    for (const [pkgPath, pkg] of packages) {
      const rel = relPkg(pkgPath);
      const seenTests = seen.get(pkgPath) || new Set();
      for (const [name, loc] of pkg.tests) {
        if (seenTests.has(name) || !judgeStart(rel, name)) continue;
        neverRan.push({ rel, test: name, loc, wholePackage: seenTests.size === 0 });
      }
    }
  }

  const staleAllowlist = allowlist.filter((e) => {
    const top = e.test.split('/')[0];
    for (const [pkgPath, pkg] of packages) {
      if (e.package && relPkg(pkgPath) !== e.package) continue;
      if (pkg.tests.has(top)) return false;
    }
    return true;
  });

  const byPkgThenTest = (a, b) => (a.rel === b.rel ? (a.test < b.test ? -1 : 1) : a.rel < b.rel ? -1 : 1);
  skippedEverywhere.sort(byPkgThenTest);
  neverRan.sort(byPkgThenTest);

  const sections = [];
  const grouped = (items, fmt) => {
    const lines = [];
    let current = null;
    for (const item of items) {
      if (item.rel !== current) {
        current = item.rel;
        const whole = items.filter((i) => i.rel === current);
        const note = whole[0].wholePackage ? ' — no test in this package ran or skipped anywhere' : '';
        lines.push('', `### \`${current}\` (${whole.length})${note}`, '');
      }
      lines.push(fmt(item));
    }
    return lines;
  };
  if (skippedEverywhere.length) {
    sections.push({
      head: [
        `## Skipped everywhere (${skippedEverywhere.length})`,
        '',
        'Every CI job that reached these skipped them. Subtests link to their top-level test.',
      ],
      lines: grouped(skippedEverywhere, (i) => {
        const msg = i.skipped.message ? ` — ${code(i.skipped.message)}` : '';
        const where = i.skipped.url ? `[${i.skipped.leg}](${i.skipped.url})` : i.skipped.leg;
        return `- ${link(i.loc, code(i.test))}${msg} (last skipped ${shortDate(i.skipped.at)} in ${where})`;
      }),
    });
  }
  if (neverRan.length) {
    sections.push({
      head: [
        `## Never ran (${neverRan.length})`,
        '',
        'No CI job reported these at all, not even as skipped. A whole package listed here is usually one that no `make` target passes to `go test`; a single test is usually one that returns early or whose parent never calls `t.Run` for it.',
      ],
      lines: grouped(neverRan, (i) => `- ${link(i.loc, code(i.test))}`),
    });
  }
  if (staleAllowlist.length) {
    sections.push({
      head: [
        `## Allowlist entries that match no test (${staleAllowlist.length})`,
        '',
        'Remove these from `.github/tests-not-run-in-ci.yml`, or fix the name if the test was renamed.',
      ],
      lines: [
        '',
        ...staleAllowlist.map((e) => `- ${e.package ? `\`${e.package}\` ` : ''}${code(e.test)}: ${e.reason.replace(/\s+/g, ' ').trim()}`),
      ],
    });
  }

  const count = skippedEverywhere.length + neverRan.length + staleAllowlist.length;
  const header = [];
  const asOf = new Date(now).toISOString().slice(0, 10);
  if (status !== 'ok') {
    header.push(
      `The last-ran index covers CI runs only since ${shortDate(index.covers_since)}, less than the ${windowDays}-day window ending ${asOf}, so tests that ran nowhere cannot be told apart from tests outside the index yet. Re-run this once the index has ${windowDays} days of runs, or rebuild it with a \`fresh\` run of \`perf-collect.yml\` with \`since\` set ${windowDays} or more days back.`
    );
  } else {
    header.push(
      `${count} Go test(s), subtest(s) or allowlist entries need a look: in the ${windowDays} days before ${asOf}, the tests below did not run (pass or fail) in any CI job on \`main\` that uploads gotestsum results (GitHub Actions test workflows and Buildkite pipelines; the last-ran index was updated ${shortDate(index.updated)}).`,
      '',
      'For each test, either make a CI job run it, delete it, or add it to `.github/tests-not-run-in-ci.yml` with the reason it cannot run in CI. Tests that only run in a rarely-run job can get a `window_days` there instead, so they are judged over a longer window.'
    );
  }

  const footer = ['', '---', opts['run-url'] ? `[Report run](${opts['run-url']})` : 'Generated by `perf/collector/report-tests-not-run.js`.'];
  let body = [...header];
  let budget = MAX_BODY_CHARS - header.join('\n').length - footer.join('\n').length;
  for (const s of sections) {
    const head = ['', ...s.head];
    budget -= head.join('\n').length;
    body.push(...head);
    let omitted = 0;
    for (const line of s.lines) {
      if (budget - line.length - 1 < 200) {
        if (line.startsWith('- ')) omitted++;
        continue;
      }
      budget -= line.length + 1;
      body.push(line);
    }
    if (omitted) body.push('', `…and ${omitted} more, left out to fit GitHub's issue size limit.`);
  }
  body.push(...footer);
  return { markdown: body.join('\n') + '\n', count, status };
}

if (require.main === module) {
  let opts;
  try {
    opts = parseArgs(process.argv.slice(2));
    if (!opts.index) throw new Error('--index=<last-ran.json> is required');
    const { markdown, count, status } = buildReport(opts);
    if (opts.out) fs.writeFileSync(opts.out, markdown);
    else process.stdout.write(markdown);
    if (process.env.GITHUB_OUTPUT) fs.appendFileSync(process.env.GITHUB_OUTPUT, `count=${count}\nstatus=${status}\n`);
    console.error(`status=${status} count=${count}`);
  } catch (err) {
    console.error(`ERROR: ${err.message}`);
    process.exit(1);
  }
}

module.exports = { buildReport, testFuncsInFile, allowlistMatch, readAllowlist };
