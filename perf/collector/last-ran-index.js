// Compact index of when each Go test and subtest last ran (passed or failed)
// and last skipped in CI on main, kept as ci/tests/last-ran.json on the
// performance-data branch. collect-per-test.js folds every gotestsum artifact
// it already downloads into it; report-tests-not-run.js reads it to list
// tests that ran nowhere. Unlike per-test-history.ndjson, nothing is filtered
// out here (subtests, fast tests and skips all count), but each test keeps
// only its latest outcome, so the file stays at one entry per test name.
//
// Shape (one test entry per line, so the branch's git diffs stay readable):
//   {
//     "schema": 1,
//     "covers_since": "<timestamp of the oldest run folded in, or the
//                      retention cutoff once older entries are pruned>",
//     "updated": "<timestamp of the newest run folded in>",
//     "tests": {
//       "<package> <test>": {
//         "ran":     { "at", "leg", "outcome": "pass"|"fail", "url" },
//         "skipped": { "at", "leg", "message", "url" }
//       }
//     }
//   }
// "<package>" is the Go import path, "<test>" the test2json name (subtests
// include their "/" path). Either of ran/skipped may be absent.

const fs = require('fs');

const SCHEMA = 1;

// Entries whose newest outcome is older than this are dropped. Matches the
// 90-day retention of the gotestsum build artifacts, so a `fresh` rebuild
// can reproduce everything the index holds. The report's window must be
// shorter than this.
const INDEX_RETENTION_DAYS = 90;

// Longest skip message kept, so one verbose t.Skipf cannot bloat the file.
const MAX_MESSAGE_LEN = 200;

// A t.Skip/t.Log line as `go test -v` prints it: indented, then
// "file.go:123: text". Subtest output is indented further.
const LOG_LINE_RE = /^\s+([\w.-]+\.go:\d+): (.*)$/;

// Every terminal pass/fail/skip event in a test2json (gotestsum --jsonfile)
// stream, subtests included. A skip carries the last "file.go:N: text" line
// the same test printed, which is where t.Skip's message lands; with -v it is
// printed before the "--- SKIP" line, without -v after it, so lines that
// follow the skip are still attached until the package ends.
function extractAllTestEvents(ndjsonText) {
  const events = [];
  const lastLogLine = new Map(); // "<package> <test>" -> last log line
  const pendingSkip = new Map(); // "<package> <test>" -> skip event still missing a message
  for (const line of ndjsonText.split('\n')) {
    if (!line.trim()) continue;
    let event;
    try {
      event = JSON.parse(line);
    } catch {
      continue; // one corrupt line should not lose the rest of the file
    }
    if (!event.Test) continue;
    const key = `${event.Package} ${event.Test}`;
    if (event.Action === 'output') {
      const m = LOG_LINE_RE.exec((event.Output || '').replace(/\s+$/, ''));
      if (m) {
        const text = `${m[1]}: ${m[2]}`.slice(0, MAX_MESSAGE_LEN);
        lastLogLine.set(key, text);
        const skip = pendingSkip.get(key);
        if (skip) {
          skip.message = text;
          pendingSkip.delete(key);
        }
      }
      continue;
    }
    if (!['pass', 'fail', 'skip'].includes(event.Action)) continue;
    const out = { package: event.Package, test: event.Test, elapsed_s: event.Elapsed || 0, outcome: event.Action };
    if (event.Action === 'skip') {
      out.message = lastLogLine.get(key) || '';
      if (!out.message) pendingSkip.set(key, out);
    }
    lastLogLine.delete(key);
    events.push(out);
  }
  return events;
}

function emptyIndex() {
  return { schema: SCHEMA, covers_since: null, updated: null, tests: {} };
}

function loadIndex(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return emptyIndex();
  const index = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  if (index.schema !== SCHEMA) {
    // An older shape cannot be merged safely; start over. A `fresh` run of
    // perf-collect.yml backfills it from the retained artifacts.
    console.warn(`Ignoring ${filePath}: schema ${index.schema}, expected ${SCHEMA}`);
    return emptyIndex();
  }
  return index;
}

// One entry per line keeps the performance-data branch's diffs small.
function serializeIndex(index) {
  const keys = Object.keys(index.tests).sort();
  const lines = keys.map((k) => `    ${JSON.stringify(k)}: ${JSON.stringify(index.tests[k])}`);
  return (
    '{\n' +
    `  "schema": ${index.schema},\n` +
    `  "covers_since": ${JSON.stringify(index.covers_since)},\n` +
    `  "updated": ${JSON.stringify(index.updated)},\n` +
    `  "tests": {\n${lines.join(',\n')}${lines.length ? '\n' : ''}  }\n` +
    '}\n'
  );
}

function saveIndex(filePath, index) {
  fs.writeFileSync(filePath, serializeIndex(index));
}

// Folds one run's events into the index. `run` is { at, leg, url }. Only a
// newer run replaces an entry's ran/skipped, so runs may arrive in any order
// (a --since backfill over runs already folded in is harmless).
function updateIndex(index, events, run) {
  for (const e of events) {
    const key = `${e.package} ${e.test}`;
    const entry = index.tests[key] || (index.tests[key] = {});
    if (e.outcome === 'skip') {
      if (!entry.skipped || entry.skipped.at < run.at) {
        entry.skipped = { at: run.at, leg: run.leg, message: e.message || '', url: run.url };
      }
    } else if (!entry.ran || entry.ran.at < run.at || (entry.ran.at === run.at && e.outcome === 'fail')) {
      // Same run reporting both (a test in two make targets, or a retried
      // job): keep the failure, since it is the more useful outcome to see.
      entry.ran = { at: run.at, leg: run.leg, outcome: e.outcome, url: run.url };
    }
  }
  if (!index.covers_since || run.at < index.covers_since) index.covers_since = run.at;
  if (!index.updated || run.at > index.updated) index.updated = run.at;
}

// Drops entries with nothing newer than the retention cutoff, and moves
// covers_since up to the cutoff once anything older has been dropped.
function pruneIndex(index, now = Date.now()) {
  const cutoff = new Date(now - INDEX_RETENTION_DAYS * 86400 * 1000).toISOString();
  let pruned = 0;
  for (const [key, entry] of Object.entries(index.tests)) {
    const newest = [entry.ran?.at, entry.skipped?.at].filter(Boolean).sort().pop();
    if (!newest || newest < cutoff) {
      delete index.tests[key];
      pruned++;
      continue;
    }
    if (entry.ran && entry.ran.at < cutoff) delete entry.ran;
    if (entry.skipped && entry.skipped.at < cutoff) delete entry.skipped;
  }
  if (index.covers_since && index.covers_since < cutoff) index.covers_since = cutoff;
  return pruned;
}

module.exports = {
  INDEX_RETENTION_DAYS,
  extractAllTestEvents,
  emptyIndex,
  loadIndex,
  saveIndex,
  serializeIndex,
  updateIndex,
  pruneIndex,
};
