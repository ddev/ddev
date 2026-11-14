// Drives a real `ddev serve` through one project's life: add, start, logs,
// Xdebug, snapshot, restore, delete. It needs Docker, so it runs only with
// DDEV_WEBUI_E2E=true. DDEV_BINARY names the ddev to test (default: ddev
// on PATH). Its projects live in a separate DDEV_XDG_CONFIG_HOME, so the
// user's own project list is not touched.
'use strict';

const { test, expect } = require('@playwright/test');
const { spawn, spawnSync } = require('node:child_process');
const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

test.skip(process.env.DDEV_WEBUI_E2E !== 'true', 'set DDEV_WEBUI_E2E=true to run against Docker');
test.describe.configure({ mode: 'serial' });

const ddev = process.env.DDEV_BINARY || 'ddev';
const id = crypto.randomBytes(3).toString('hex');
const name = `webui-e2e-${id}`;
// Docker providers share the home directory, not always the system temp dir.
const root = path.join(os.homedir(), 'tmp', `webui-e2e-${id}`);
const projectDir = path.join(root, name);
const env = {
  ...process.env,
  DDEV_XDG_CONFIG_HOME: path.join(root, 'xdg'),
  DDEV_NO_INSTRUMENTATION: 'true',
};

let serve;
let dashboardURL;

test.beforeAll(async () => {
  fs.mkdirSync(path.join(projectDir, 'web'), { recursive: true });
  fs.writeFileSync(path.join(projectDir, 'web', 'index.php'), `<?php echo "hello from ${name}";\n`);
  serve = spawn(ddev, ['serve', '--no-open'], { env, cwd: root, stdio: ['ignore', 'pipe', 'pipe'] });
  let output = '';
  dashboardURL = await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error('ddev serve printed no URL:\n' + output)), 60000);
    const read = (chunk) => {
      output += chunk;
      const m = output.match(/http:\/\/127\.0\.0\.1:\d+\/\?token=[0-9a-f]+/);
      if (m) {
        clearTimeout(timer);
        resolve(m[0]);
      }
    };
    serve.stdout.on('data', read);
    serve.stderr.on('data', read);
    serve.on('exit', (code) => reject(new Error(`ddev serve exited with ${code}:\n${output}`)));
  });
});

test.afterAll(async () => {
  if (serve) serve.kill();
  // A failed run can leave the project behind.
  spawnSync(ddev, ['delete', '-Oy', name], { env, stdio: 'inherit' });
  fs.rmSync(root, { recursive: true, force: true });
});

// runAndWait clicks, then waits for the job titled title to end, and fails
// with its output when the job fails.
async function runAndWait(page, title, click, timeout = 10 * 60 * 1000) {
  await click();
  await expect(page.locator('#activity-title')).toHaveText(title);
  const out = page.locator('#job-output');
  await expect(out).toContainText(/✓ Done|✗ Failed/, { timeout });
  const text = await out.textContent();
  expect(text, text).toContain('✓ Done');
}

test('manages a project from the dashboard', async ({ page, request }) => {
  await page.goto(dashboardURL);
  await expect(page).toHaveURL(/\/$/);
  await expect(page.locator('#conn-status')).toHaveText('connected');

  await test.step('add the project', async () => {
    await runAndWait(page, `Configuring ${projectDir}`, async () => {
      await page.getByRole('button', { name: 'Add project' }).click();
      const dialog = page.locator('#dialog');
      await dialog.getByLabel('Project directory').fill(projectDir);
      await dialog.getByLabel('Project type').selectOption('php');
      await dialog.getByLabel('Docroot').fill('web');
      await dialog.getByRole('button', { name: 'Configure' }).click();
    });
    await page.locator('#project-list li', { hasText: name }).click();
    await expect(page.locator('.detail-head h1')).toContainText(name);
    await expect(page.locator('.detail-head')).toContainText('stopped');
  });

  const toolbar = page.locator('.toolbar');
  await test.step('start it and open the site', async () => {
    await runAndWait(page, `Starting ${name}`, () => toolbar.getByRole('button', { name: 'Start', exact: true }).click());
    await expect(page.locator('.detail-head')).toContainText('running');
    const site = toolbar.getByRole('link', { name: 'Open site ↗' });
    await expect(site).toBeVisible();
    const res = await request.get(await site.getAttribute('href'));
    expect(await res.text()).toContain(`hello from ${name}`);
    await expect(page.locator('#detail table.kv')).toContainText('mariadb');
  });

  await test.step('follow the web logs', async () => {
    await page.getByRole('button', { name: 'Logs' }).click();
    await expect(page.locator('.logs-output')).toContainText(/nginx|php-fpm/);
    await page.getByRole('button', { name: 'Overview' }).click();
  });

  await test.step('toggle Xdebug', async () => {
    await runAndWait(page, `Turning Xdebug on in ${name}`, () => page.getByRole('checkbox').check());
    await expect(page.locator('.switch')).toContainText('enabled');
    await runAndWait(page, `Turning Xdebug off in ${name}`, () => page.getByRole('checkbox').uncheck());
    await expect(page.locator('.switch')).toContainText('disabled');
  });

  await test.step('snapshot and restore the database', async () => {
    await page.getByRole('button', { name: 'Snapshots' }).click();
    await runAndWait(page, `Making a snapshot of ${name}`, () => page.getByRole('button', { name: 'Create snapshot' }).click());
    const row = page.locator('table.grid tr', { hasText: name });
    await expect(row).toHaveCount(1);
    await runAndWait(page, /^Restoring snapshot /, async () => {
      await row.getByRole('button', { name: 'Restore' }).click();
      await page.locator('#dialog').getByRole('button', { name: 'Restore' }).click();
    });
  });

  await test.step('delete it', async () => {
    await runAndWait(page, `Deleting ${name} without a snapshot`, async () => {
      await toolbar.getByRole('button', { name: 'Delete' }).click();
      await page.locator('#dialog').getByText('Skip the database snapshot').click();
      await page.locator('#dialog').getByRole('button', { name: 'Delete' }).click();
    });
    await expect(page.locator('#project-list li', { hasText: name })).toHaveCount(0);
  });
});
