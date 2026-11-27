// Drives the dashboard inside the app's own window and reports the result
// to /selftest/result. selftest.go prepends DDEV_SELFTEST_DIR, an empty
// directory to configure as a project.
(async () => {
  const $ = (sel) => document.querySelector(sel);
  const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
  const until = async (what, check, ms = 30000) => {
    const end = Date.now() + ms;
    for (;;) {
      const v = check();
      if (v) return v;
      if (Date.now() > end) throw new Error('timed out waiting for ' + what);
      await sleep(100);
    }
  };
  const setValue = (sel, value) => {
    const el = $(sel);
    el.value = value;
    el.dispatchEvent(new Event('input', { bubbles: true }));
  };
  const addProject = async (dir) => {
    $('#btn-add').click();
    await until('the Add project dialog', () => $('#dialog').open && $('#f-dir'));
    setValue('#f-dir', dir);
    setValue('#f-name', 'selftest');
    $('#f-type').value = 'php';
    $('#dialog-ok').click();
    await until('the dialog to close', () => !$('#dialog').open);
  };
  const report = (result) => fetch('/selftest/result', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-DDEV-Dashboard': '1' },
    body: JSON.stringify(result),
  });

  let step = 'load';
  try {
    step = 'status stream';
    // #status-text leaves Connecting… once the stream opens. Without Docker
    // it then reads "Docker unreachable", which is enough here.
    await until('the status stream', () => !/^(Connecting|Disconnected)/.test($('#status-text').textContent));

    step = 'refused directory';
    await addProject(DDEV_SELFTEST_DIR + '/missing');
    await until('an error toast', () => $('#toasts .toast.error'));

    step = 'ddev config job';
    await addProject(DDEV_SELFTEST_DIR);
    await until('the job output', () => $('#job-output').textContent.includes('$ ddev config'));
    await until('the job to finish', () => /✓ Done|✗ Failed/.test($('#job-output').textContent), 120000);
    if (!$('#job-output').textContent.includes('✓ Done')) throw new Error('the job failed');

    // With Docker, a detail starts host commands such as git, which hang on
    // Windows if they inherit the app's stdin pipe. Without Docker it fails
    // before them, so only an answer is required.
    step = 'project detail';
    await fetch('/api/projects/selftest', { signal: AbortSignal.timeout(30000) });

    await report({ ok: true });
  } catch (e) {
    await report({ ok: false, step, error: String(e), output: ($('#job-output') || {}).textContent || '' });
  }
})();
