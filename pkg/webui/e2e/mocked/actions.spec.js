'use strict';

const { test, expect, nthPost } = require('./fixtures');

test('streams a job into the activity panel', async ({ page, backend }) => {
  await page.locator('.toolbar').getByRole('button', { name: 'Stop', exact: true }).click();
  const post = await nthPost(backend, 1);
  expect(post.path).toBe('/api/projects/d11/actions/stop');

  const activity = page.locator('#activity');
  await expect(activity).not.toHaveClass(/collapsed/);
  await expect(page.locator('#activity-title')).toHaveText('stop d11');
  await expect(page.locator('#activity-count')).toHaveText('1');

  // While the job runs, the project's actions are off and its row says so.
  await expect(page.locator('.toolbar').getByRole('button', { name: 'Restart' })).toBeDisabled();
  await expect(page.locator('#project-list li', { hasText: 'd11' })).toContainText('stop d11…');
  await expect(page.locator('#project-list li', { hasText: 'd11' }).locator('.dot')).toHaveClass(/busy/);

  const job = backend.jobs[0];
  backend.jobLines(job.id, ['$ ddev stop d11', 'Stopping d11...']);
  await expect(page.locator('#job-output')).toContainText('Stopping d11...');
  backend.endJob(job.id);
  await expect(page.locator('#job-output')).toContainText('✓ Done');
  await expect(page.locator('#activity-count')).toBeEmpty();
  await expect(page.locator('.toolbar').getByRole('button', { name: 'Restart' })).toBeEnabled();
});

test('reports a failed job', async ({ page, backend }) => {
  await page.locator('.toolbar').getByRole('button', { name: 'Restart' }).click();
  await nthPost(backend, 1);
  backend.endJob(backend.jobs[0].id, 'exit status 1');
  await expect(page.locator('#job-output')).toContainText('✗ Failed: exit status 1');
  await expect(page.locator('#toasts .toast.error')).toContainText('restart d11 failed: exit status 1');
  await expect(page.locator('#job-list li .dot')).toHaveClass(/bad/);
});

test('shows a refused action', async ({ page, backend }) => {
  backend.postReply = () => ({ status: 409, body: 'already running: Starting all projects' });
  await page.locator('.toolbar').getByRole('button', { name: 'Restart' }).click();
  await expect(page.locator('#toasts .toast.error')).toHaveText('already running: Starting all projects');
  expect(backend.jobs).toHaveLength(0);
});

test('a global job blocks every project', async ({ page, backend }) => {
  backend.addJob('Powering off DDEV', '');
  await expect(page.locator('.toolbar').getByRole('button', { name: 'Stop', exact: true })).toBeDisabled();
  await page.locator('#project-list li', { hasText: 'wp' }).click();
  await expect(page.locator('.toolbar').getByRole('button', { name: 'Start', exact: true })).toBeDisabled();
});

test('shows earlier jobs in the activity list', async ({ page, backend }) => {
  const first = backend.addJob('First', 'd11');
  backend.jobLines(first.id, ['first output']);
  backend.endJob(first.id);
  const second = backend.addJob('Second', 'wp');
  backend.jobLines(second.id, ['second output']);
  backend.endJob(second.id);
  await page.locator('#activity-toggle').click();
  const jobs = page.locator('#job-list li');
  await expect(jobs).toHaveCount(2);
  await expect(jobs.nth(0)).toContainText('Second');
  await jobs.nth(1).click();
  await expect(page.locator('#job-output')).toContainText('first output');
  await expect(page.locator('#job-output')).not.toContainText('second output');
  await page.locator('#activity-toggle').click();
  await expect(page.locator('#activity')).toHaveClass(/collapsed/);
});

test('delete asks first, and can skip the snapshot', async ({ page, backend }) => {
  const toolbar = page.locator('.toolbar');
  const dialog = page.locator('#dialog');

  await toolbar.getByRole('button', { name: 'Delete' }).click();
  await expect(dialog.locator('h2')).toHaveText('Delete d11?');
  await dialog.getByRole('button', { name: 'Cancel' }).click();
  await expect(dialog).toBeHidden();

  await toolbar.getByRole('button', { name: 'Delete' }).click();
  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
  expect(backend.posts).toHaveLength(0);

  await toolbar.getByRole('button', { name: 'Delete' }).click();
  await dialog.getByText('Skip the database snapshot').click();
  await dialog.getByRole('button', { name: 'Delete' }).click();
  const post = await nthPost(backend, 1);
  expect(post).toEqual({ path: '/api/projects/d11/actions/delete', form: { omit_snapshot: 'true' } });

  backend.endJob(backend.jobs[0].id);
  await toolbar.getByRole('button', { name: 'Delete' }).click();
  await dialog.getByRole('button', { name: 'Delete' }).click();
  expect((await nthPost(backend, 2)).form).toEqual({ omit_snapshot: 'false' });
});

test('global actions', async ({ page, backend }) => {
  const dialog = page.locator('#dialog');
  await page.getByRole('button', { name: 'Start all' }).click();
  expect((await nthPost(backend, 1)).path).toBe('/api/actions/start-all');
  backend.endJob(backend.jobs[0].id);

  await page.getByRole('button', { name: 'Stop all' }).click();
  await expect(dialog.locator('h2')).toHaveText('Stop all projects?');
  await dialog.getByRole('button', { name: 'Cancel' }).click();
  await page.getByRole('button', { name: 'Stop all' }).click();
  await dialog.getByRole('button', { name: 'Stop all' }).click();
  expect((await nthPost(backend, 2)).path).toBe('/api/actions/stop-all');
  backend.endJob(backend.jobs[1].id);

  await page.getByRole('button', { name: 'Power off' }).click();
  await dialog.getByRole('button', { name: 'Power off' }).click();
  expect((await nthPost(backend, 3)).path).toBe('/api/actions/poweroff');
});

test('opens the folder and editors on the host', async ({ page, backend }) => {
  const toolbar = page.locator('.toolbar');
  await toolbar.getByRole('button', { name: 'Folder' }).click();
  await toolbar.getByRole('button', { name: 'VS Code' }).click();
  await toolbar.getByRole('button', { name: 'PhpStorm' }).click();
  await expect.poll(() => backend.posts.map((p) => p.path)).toEqual([
    '/api/projects/d11/open/folder',
    '/api/projects/d11/open/vscode',
    '/api/projects/d11/open/phpstorm',
  ]);
  expect(backend.jobs).toHaveLength(0);

  backend.postReply = () => ({ status: 500, body: 'no opener found (install xdg-utils)' });
  await toolbar.getByRole('button', { name: 'Folder' }).click();
  await expect(page.locator('#toasts .toast.error')).toHaveText('no opener found (install xdg-utils)');
});

test('copies commands', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.locator('.detail-head .copy-btn').click();
  await expect(page.locator('#toasts .toast')).toHaveText('Copied: /home/me/projects/d11');
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('/home/me/projects/d11');
});
