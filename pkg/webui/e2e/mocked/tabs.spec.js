'use strict';

const { test, expect, nthPost, reload } = require('./fixtures');

test('toggles Xdebug', async ({ page, backend }) => {
  const toggle = page.getByRole('switch');
  await expect(toggle).not.toBeChecked();
  await toggle.check();
  expect((await nthPost(backend, 1)).path).toBe('/api/projects/d11/actions/xdebug-on');
  // While the job runs, the switch keeps the requested state.
  await expect(page.locator('.xdebug-switch')).toContainText('turning on…');
  await expect(page.locator('#activity')).toHaveClass(/collapsed/);
  await expect(toggle).toBeChecked();
  await expect(toggle).toBeDisabled();
  backend.details.d11.xdebug = 'on';
  backend.endJob(backend.jobs[0].id);
  await expect(page.locator('.xdebug-switch')).toHaveText(/Xdebug on$/);
  // The job's end and the status event each rebuild the panel; the enabled
  // Stop button shows both have arrived.
  await expect(page.locator('.toolbar').getByRole('button', { name: 'Stop', exact: true })).toBeEnabled();
  await expect(toggle).toBeChecked();
  await toggle.uncheck();
  expect((await nthPost(backend, 2)).path).toBe('/api/projects/d11/actions/xdebug-off');
});

test('resets the Xdebug switch when its job is refused', async ({ page, backend }) => {
  backend.postReply = () => ({ status: 409, body: 'already running: Stopping all projects' });
  const toggle = page.getByRole('switch');
  // check() would fail when the switch flips back before it looks.
  await toggle.click();
  await expect(page.locator('#toasts .toast.error')).toHaveText('already running: Stopping all projects');
  await expect(page.locator('.xdebug-switch')).toHaveText(/Xdebug off$/);
  await expect(toggle).not.toBeChecked();
});

test('installs, updates and removes add-ons', async ({ page, backend }) => {
  await page.getByRole('button', { name: 'Add-ons' }).click();
  const rows = page.locator('table.grid tr');
  await expect(rows.nth(1)).toContainText('redis');
  await expect(rows.nth(1).getByRole('link')).toHaveAttribute('href', 'https://github.com/ddev/ddev-redis');

  // An empty name sends nothing.
  await page.getByRole('button', { name: 'Install' }).click();
  // The registry's add-ons are suggested as the name is typed.
  const repo = page.getByLabel('Add-on repository');
  await repo.click();
  await expect(page.locator('.menu').getByRole('menuitem')).toHaveCount(3);
  await repo.fill('sol');
  await expect(page.locator('.menu').getByRole('menuitem')).toHaveText(['ddev/ddev-solr']);
  await expect(repo).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Enter');
  await expect(repo).toHaveValue('ddev/ddev-solr');
  // The picked add-on's list stays closed.
  await expect(page.locator('.menu')).toHaveCount(0);
  // The list matches fuzzily, and allows for a typo, best match first.
  for (const q of ['rds', 'rediss', 'reids']) {
    await repo.fill(q);
    await expect(page.locator('.menu').getByRole('menuitem')).toHaveText(['ddev/ddev-redis', 'ddev/ddev-redis-insight']);
  }
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Enter');
  await expect(page.locator('.menu')).toHaveCount(0);
  await repo.fill('ddev/ddev-solr');
  await page.keyboard.press('Escape');
  await repo.press('Enter');
  expect(await nthPost(backend, 1)).toEqual({ path: '/api/projects/d11/actions/addon-get', form: { repo: 'ddev/ddev-solr' } });
  backend.endJob(backend.jobs[0].id);

  await page.getByRole('button', { name: 'Update all' }).click();
  expect((await nthPost(backend, 2)).path).toBe('/api/projects/d11/actions/addon-update');
  backend.endJob(backend.jobs[1].id);

  await rows.nth(1).getByRole('button', { name: 'Remove' }).click();
  await expect(page.locator('#dialog h2')).toHaveText('Remove redis?');
  await page.locator('#dialog').getByRole('button', { name: 'Remove' }).click();
  expect(await nthPost(backend, 3)).toEqual({ path: '/api/projects/d11/actions/addon-remove', form: { addon: 'redis' } });
});

test('shows when no add-ons are installed', async ({ page, backend }) => {
  backend.details.d11.addons = [];
  await reload(page);
  await page.getByRole('button', { name: 'Add-ons' }).click();
  await expect(page.locator('#detail')).toContainText('No add-ons installed.');
  await expect(page.getByRole('button', { name: 'Update all' })).toBeDisabled();
});

test('creates and restores snapshots', async ({ page, backend }) => {
  await page.getByRole('button', { name: 'Snapshots' }).click();
  const rows = page.locator('table.grid tr');
  await expect(rows).toHaveCount(3);
  await expect(rows.nth(1)).toContainText('d11_new');
  await expect(rows.nth(1)).toContainText('2.5 MB');

  await page.getByRole('button', { name: 'Create snapshot' }).click();
  expect((await nthPost(backend, 1)).path).toBe('/api/projects/d11/actions/snapshot');
  backend.endJob(backend.jobs[0].id);

  await rows.nth(2).getByRole('button', { name: 'Restore' }).click();
  await expect(page.locator('#dialog h2')).toHaveText('Restore d11_old?');
  await page.locator('#dialog').getByRole('button', { name: 'Restore' }).click();
  expect(await nthPost(backend, 2)).toEqual({ path: '/api/projects/d11/actions/restore', form: { snapshot: 'd11_old' } });
  backend.endJob(backend.jobs[1].id);

  await page.getByRole('button', { name: 'Restore latest' }).click();
  await page.locator('#dialog').getByRole('button', { name: 'Restore' }).click();
  expect((await nthPost(backend, 3)).form).toEqual({ snapshot: '' });
});

test('snapshots need a database', async ({ page, backend }) => {
  backend.details.d11.describe.dbinfo = undefined;
  backend.details.d11.snapshots = [];
  await reload(page);
  await page.getByRole('button', { name: 'Snapshots' }).click();
  await expect(page.getByRole('button', { name: 'Create snapshot' })).toBeDisabled();
  await expect(page.getByRole('button', { name: 'Restore latest' })).toBeDisabled();
  await page.getByRole('button', { name: 'Database' }).click();
  await expect(page.locator('#detail')).toContainText('This project has no database container.');
});

test('follows logs and stops when leaving the tab', async ({ page, backend }) => {
  await page.getByRole('button', { name: 'Logs' }).click();
  await expect.poll(() => backend.logRequests).toEqual([{ project: 'd11', service: 'web' }]);
  backend.logLines(['nginx started', 'php-fpm ready']);
  const out = page.locator('.logs-output');
  await expect(out).toContainText('php-fpm ready');

  // A status update must not restart the stream or clear the output.
  backend.pushStatus();
  backend.logLines(['after status']);
  await expect(out).toContainText('nginx started');
  await expect(out).toContainText('after status');
  expect(backend.logRequests).toHaveLength(1);

  await page.getByLabel('Service').click();
  await page.getByRole('option', { name: 'db', exact: true }).click();
  await expect.poll(() => backend.logRequests.at(-1)).toEqual({ project: 'd11', service: 'db' });
  await expect(out).not.toContainText('nginx started');

  await page.getByRole('button', { name: 'Overview' }).click();
  await expect.poll(() => backend.logClients.size).toBe(0);
});

test('logs of a stopped project', async ({ page, backend }) => {
  await page.locator('#project-list li', { hasText: 'wp' }).click();
  await page.getByRole('button', { name: 'Logs' }).click();
  await expect(page.locator('.logs-output')).toHaveText('The project is not running.');
  expect(backend.logRequests).toHaveLength(0);
});

test('adds a project', async ({ page, backend }) => {
  const dialog = page.locator('#dialog');
  await page.locator('#btn-add').click();
  await expect(dialog.locator('h2')).toHaveText('Add project');
  await dialog.getByLabel('Project type').click();
  await expect(dialog.getByRole('option')).toHaveCount(5);
  await dialog.getByLabel('Project type').click();

  // The directory is required.
  await dialog.getByRole('button', { name: 'Configure' }).click();
  await expect(dialog).toBeVisible();
  expect(backend.posts).toHaveLength(0);

  await dialog.getByLabel('Project directory').fill('/home/me/projects/new');
  await dialog.getByLabel('Project type').click();
  await dialog.getByRole('option', { name: 'laravel', exact: true }).click();
  await dialog.getByLabel('Project name').fill('newsite');
  await dialog.getByLabel('Docroot').fill('public');
  await dialog.getByRole('button', { name: 'Configure' }).click();
  await expect(dialog).toBeHidden();
  expect(await nthPost(backend, 1)).toEqual({
    path: '/api/projects',
    form: { dir: '/home/me/projects/new', type: 'laravel', name: 'newsite', docroot: 'public' },
  });
});

test('reports a refused project directory', async ({ page, backend }) => {
  backend.postReply = () => ({ status: 400, body: 'no such directory: /nope' });
  await page.locator('#btn-add').click();
  await page.getByLabel('Project directory').fill('/nope');
  await page.getByLabel('Project directory').press('Enter');
  await expect(page.locator('#toasts .toast.error')).toHaveText('no such directory: /nope');
});
