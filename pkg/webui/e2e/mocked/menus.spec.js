'use strict';

const { test, expect, nthPost } = require('./fixtures');

test('a right-click on a project opens its menu', async ({ page, backend }) => {
  await page.locator('#project-list li', { hasText: 'wp' }).click({ button: 'right' });
  const menu = page.getByRole('menu');
  await expect(menu.getByRole('menuitem', { name: 'Start', exact: true })).toBeEnabled();
  await expect(menu.getByRole('menuitem', { name: 'Restart' })).toBeDisabled();
  await expect(menu.getByRole('menuitem', { name: 'Open site ↗' })).toBeDisabled();
  await page.keyboard.press('Escape');
  await expect(menu).toHaveCount(0);

  await page.locator('#project-list li', { hasText: 'd11' }).click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Stop' }).click();
  expect((await nthPost(backend, 1)).path).toBe('/api/projects/d11/actions/stop');
  await expect(page.getByRole('menu')).toHaveCount(0);

  await page.locator('#project-list li', { hasText: 'wp' }).click({ button: 'right' });
  await page.getByRole('menuitem', { name: 'Open folder' }).click();
  expect((await nthPost(backend, 2)).path).toBe('/api/projects/wp/open/folder');
});

test('a dropdown opens, closes on a second click, and picks with the keyboard', async ({ page, backend }) => {
  await page.getByRole('button', { name: 'Logs' }).click();
  const service = page.getByLabel('Service');
  await expect(service).toHaveText('web');
  await service.click();
  await expect(page.locator('.menu[role=listbox]')).toBeVisible();
  await service.click();
  await expect(page.locator('.menu[role=listbox]')).toHaveCount(0);

  await service.focus();
  await page.keyboard.press('ArrowDown');
  await expect(page.getByRole('option', { name: 'web' })).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('Enter');
  await expect(service).toHaveText('db');
  await expect(service).toBeFocused();
  await expect.poll(() => backend.logRequests.at(-1)?.service).toBe('db');
});

test('opens on the start page, with a tip and notices', async ({ page }) => {
  await page.goto(page.url().split('#')[0]);
  await expect(page.locator('#detail h1')).toHaveText(/^(Good (morning|afternoon|evening)|Working late\?)$/);
  await expect(page.locator('#project-list li.selected')).toHaveCount(0);
  const tip = page.locator('.tip-text');
  const first = await tip.textContent();
  await expect(page.locator('.tip-nav')).toContainText('of 2');
  await page.getByRole('button', { name: 'Next tip' }).click();
  await expect(tip).not.toHaveText(first);
  await page.getByRole('button', { name: 'Previous tip' }).click();
  await expect(tip).toHaveText(first);
  await expect(page.locator('.tip-text code').or(page.locator('.tip-text em')).first()).toBeVisible();

  await expect(page.locator('#notices')).toContainText('Upgrade Docker soon.');
  await page.locator('#notices').getByRole('button', { name: 'Dismiss' }).click();
  await expect(page.locator('#notices')).toBeEmpty();

  await page.locator('#project-list li', { hasText: 'd11' }).click();
  await expect(page.locator('#detail h1')).toContainText('d11');
  await page.getByRole('button', { name: 'Start page', exact: true }).click();
  await expect(page.locator('#detail h1')).toHaveText(/^(Good (morning|afternoon|evening)|Working late\?)$/);
  await page.locator('#project-list li', { hasText: 'd11' }).click();
  await page.locator('#brand').click();
  await expect(page.locator('#detail h1')).toHaveText(/^(Good (morning|afternoon|evening)|Working late\?)$/);
});

test('the activity header opens the panel, and finished jobs clear', async ({ page, backend }) => {
  const activity = page.locator('#activity');
  await expect(page.locator('#job-list')).toContainText('show their output here');
  await page.locator('#activity-head').click({ position: { x: 500, y: 12 } });
  await expect(activity).not.toHaveClass(/collapsed/);
  await page.locator('#activity-head').click({ position: { x: 500, y: 12 } });
  await expect(activity).toHaveClass(/collapsed/);

  // A job started elsewhere, such as from the tray, takes the panel.
  const job = backend.addJob('Start d11', 'd11');
  await expect(page.locator('#activity-title .job-title')).toHaveText('Start d11');
  backend.endJob(job.id);
  await page.locator('#activity-head').click({ position: { x: 500, y: 12 } });
  await page.locator('#job-list li', { hasText: 'Start d11' }).click();
  await expect(page.locator('#activity-title .job-title')).toHaveText('Start d11');
  // A running action has no remove button; a finished one does.
  const running = backend.addJob('Stop wp', 'wp');
  await expect(page.locator('#activity-title .job-title')).toHaveText('Stop wp');
  await expect(page.getByRole('button', { name: 'Remove Stop wp from the list' })).toBeHidden();
  backend.jobLines(running.id, ['Stopping wp']);
  backend.jobPartial(running.id, 'Waiting for containers..');
  await expect(page.locator('#job-output')).toHaveText(/Stopping wp\nWaiting for containers\.\.$/);
  backend.jobPartial(running.id, 'Waiting for containers....');
  backend.jobLines(running.id, ['Waiting for containers.... done', 'Stopped wp']);
  await expect(page.locator('#job-output')).toHaveText(/Waiting for containers\.\.\.\. done\nStopped wp\n$/);
  backend.endJob(running.id);
  await page.getByRole('button', { name: 'Remove Stop wp from the list' }).click();
  await expect(page.locator('#job-list li', { hasText: 'Stop wp' })).toHaveCount(0);
  await expect(page.locator('#activity-title')).toBeEmpty();

  const clear = page.getByRole('button', { name: 'Clear history' });
  await expect(clear).toBeEnabled();
  await clear.click();
  await expect(page.locator('#job-list')).toContainText('show their output here');
  await expect(page.locator('#activity-title')).toBeEmpty();
  await expect(activity).not.toHaveClass(/collapsed/, 'the Clear button does not toggle the panel');
  await expect(clear).toBeDisabled();
});

test('shows the shared services and follows their logs', async ({ page, backend }) => {
  await expect(page.locator('#ssh-agent-status')).toHaveText('SSH agent: healthy');
  await page.locator('#router-status').click();
  await expect(page.locator('#detail h1')).toHaveText('Shared services');
  const cards = page.locator('.shared-cards .card');
  await expect(cards.first()).toContainText('healthy');
  await expect(cards.first().locator('.dot')).toHaveClass(/running/);
  await expect(cards.first().getByRole('link', { name: 'Open the Traefik dashboard ↗' })).toHaveAttribute('href', 'http://127.0.0.1:10999');
  await expect(cards.nth(1)).toContainText('stopped');
  // Only a running container has logs to follow.
  await expect(page.getByLabel('Logs of')).toHaveText('Router');
  await expect.poll(() => backend.logRequests.at(-1)?.shared).toBe('ddev-router');
  backend.logLines(['router started']);
  await expect(page.locator('.logs-output')).toContainText('router started');

  await page.getByRole('button', { name: 'Back to the start page' }).click();
  await expect(page.locator('#nav-home')).toHaveClass(/selected/);
});

test('a project type links to its quickstart', async ({ page }) => {
  await expect(page.locator('.type-link')).toHaveAttribute('href', 'https://docs.ddev.com/en/stable/users/quickstart/#drupal');
});

test('moves through the sidebar with the keyboard', async ({ page }) => {
  const h1 = page.locator('#detail h1');
  await page.locator('#project-list li', { hasText: 'd11' }).click();
  await page.keyboard.press('ArrowDown');
  await expect(h1).toContainText('wp');
  await expect(page.locator('#project-list li', { hasText: 'wp' })).toBeFocused();
  await page.keyboard.press('Home');
  await expect(page.locator('#nav-home')).toBeFocused();
  await page.keyboard.press('ArrowDown');
  await expect(h1).toHaveText('Shared services');
  await page.keyboard.press('End');
  await expect(h1).toContainText('wp');

  await page.keyboard.press('Shift+F10');
  await expect(page.getByRole('menuitem', { name: 'Start', exact: true })).toBeFocused();
  await page.keyboard.press('Escape');
});
