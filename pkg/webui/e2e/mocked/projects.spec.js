'use strict';

const { test, expect, running, stopped, detail } = require('./fixtures');

test('lists projects and selects the first', async ({ page }) => {
  const items = page.locator('#project-list li');
  await expect(items).toHaveCount(2);
  await expect(items.nth(0)).toContainText('d11');
  await expect(items.nth(0)).toContainText('https://d11.ddev.site');
  await expect(items.nth(0).locator('.dot')).toHaveClass(/running/);
  await expect(items.nth(1)).toContainText('stopped');
  await expect(items.nth(0)).toHaveClass(/selected/);
  await expect(page.locator('.detail-head h1')).toContainText('d11');
  await expect(page.locator('#router-status')).toHaveText('router: healthy');
  await expect(page.locator('#conn-status')).toHaveText('connected');
});

test('filters by name, type and status', async ({ page }) => {
  const items = page.locator('#project-list li');
  await page.keyboard.press('/');
  await expect(page.locator('#filter')).toBeFocused();
  await page.keyboard.type('word');
  await expect(items).toHaveCount(1);
  await expect(items).toContainText('wp');
  await page.locator('#filter').fill('running');
  await expect(items).toHaveCount(1);
  await expect(items).toContainText('d11');
  await page.locator('#filter').fill('nothing-matches');
  await expect(items).toHaveCount(0);
});

test('shows a running project', async ({ page }) => {
  const toolbar = page.locator('.toolbar');
  await expect(toolbar.getByRole('button', { name: 'Stop', exact: true })).toBeEnabled();
  await expect(toolbar.getByRole('link', { name: 'Open site ↗' })).toHaveAttribute('href', 'https://d11.ddev.site');
  await expect(toolbar.getByRole('link', { name: 'Mailpit ↗' })).toHaveAttribute('href', 'https://d11.ddev.site:8026');
  await expect(toolbar.getByRole('link', { name: 'XHGui ↗' })).toHaveCount(0);
  for (const link of await toolbar.getByRole('link').all()) {
    await expect(link).toHaveAttribute('target', '_blank');
    await expect(link).toHaveAttribute('rel', 'noopener noreferrer');
  }
  const detail = page.locator('#detail');
  await expect(detail.getByRole('link', { name: 'https://127.0.0.1:32770' })).toBeVisible();
  await expect(detail.locator('table.kv')).toContainText('mariadb 11.8');
  await expect(detail).toContainText('Mutagen');
  await expect(detail.getByText('cd /home/me/projects/d11 && ddev ssh')).toBeVisible();
});

test('shows a stopped project', async ({ page }) => {
  await page.locator('#project-list li', { hasText: 'wp' }).click();
  await expect(page.locator('.detail-head h1')).toContainText('wp');
  const toolbar = page.locator('.toolbar');
  await expect(toolbar.getByRole('button', { name: 'Start', exact: true })).toBeEnabled();
  await expect(toolbar.getByRole('link', { name: 'Open site ↗' })).toBeHidden();
  await expect(toolbar.getByRole('link', { name: 'Mailpit ↗' })).toHaveCount(0);
  await expect(page.locator('#detail')).toContainText('Start the project to get its URLs.');
  await expect(page.locator('#detail')).not.toContainText('Mutagen');
  await expect(page.getByRole('checkbox')).toBeDisabled();
  // A path with a space is quoted for the shell.
  await expect(page.getByText("cd '/home/me/projects/my wp' && ddev ssh")).toBeVisible();
});

test('shows services and database details', async ({ page }) => {
  await page.getByRole('button', { name: 'Services' }).click();
  const rows = page.locator('table.grid tr');
  await expect(rows).toHaveCount(4);
  await expect(rows.nth(1)).toContainText('web');
  await expect(rows.nth(1)).toContainText('80→32769');
  await expect(rows.nth(2)).toContainText('db');
  await expect(rows.nth(3)).toContainText('Pass: <none>');

  await page.getByRole('button', { name: 'Database' }).click();
  const db = page.locator('#detail .cards');
  await expect(db).toContainText('db:3306');
  await expect(db).toContainText('ddev-d11-db:3306');
  await expect(db).toContainText('127.0.0.1:32771');
  await expect(db).toContainText('ddev mysql');
});

test('remembers the tab across projects', async ({ page }) => {
  await page.getByRole('button', { name: 'Snapshots' }).click();
  await page.locator('#project-list li', { hasText: 'wp' }).click();
  await expect(page.getByRole('button', { name: 'Snapshots' })).toHaveClass(/active/);
  await expect(page.locator('table.grid')).toContainText('wp_new');
});

test('updates when the status changes', async ({ page, backend }) => {
  backend.setProjects([running, { ...stopped, status: 'running', url: 'https://wp.ddev.site' }]);
  await expect(page.locator('#project-list li', { hasText: 'wp' })).toContainText('https://wp.ddev.site');
  backend.setProjects([running]);
  await expect(page.locator('#project-list li')).toHaveCount(1);
});

test('shows an error banner while Docker is unreachable', async ({ page, backend }) => {
  const banner = page.locator('#error-banner');
  await expect(banner).toBeHidden();
  backend.error = 'Docker is not reachable: connection refused';
  backend.pushStatus();
  await expect(banner).toHaveText('Docker is not reachable: connection refused');
  backend.error = '';
  backend.pushStatus();
  await expect(banner).toBeHidden();
});

test('shows the empty state', async ({ page, backend }) => {
  backend.setProjects([]);
  await expect(page.locator('#project-empty')).toBeVisible();
});

test('shows a detail error', async ({ page, backend }) => {
  delete backend.details.wp;
  await page.locator('#project-list li', { hasText: 'wp' }).click();
  await expect(page.locator('#detail')).toContainText('no project named wp');
});

test('renders names and output as text, not HTML', async ({ page, backend }) => {
  const name = '<img src=x onerror="window.pwned=1">';
  const p = { ...stopped, name, approot: '/tmp/x' };
  backend.details[name] = detail(p);
  backend.setProjects([p]);
  await expect(page.locator('#project-list li')).toContainText(name);
  await expect(page.locator('.detail-head h1')).toContainText(name);
  const job = backend.addJob(name, name);
  backend.jobLines(job.id, ['<script>window.pwned=1</script>']);
  await page.locator('#activity-toggle').click();
  await page.locator('#job-list li').first().click();
  await expect(page.locator('#job-output')).toContainText('<script>window.pwned=1</script>');
  await expect(page.locator('#detail img, #job-output script, #project-list img')).toHaveCount(0);
  expect(await page.evaluate(() => window.pwned)).toBeUndefined();
});

test('fits a phone screen', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 });
  await expect(page.locator('.detail-head h1')).toContainText('d11');
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(overflow).toBeLessThanOrEqual(0);
});
