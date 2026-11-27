'use strict';

const { test, expect, nthPost } = require('./fixtures');

test('a right-click on a project opens its menu', async ({ page, backend }) => {
  await page.locator('#project-list li', { hasText: 'wp' }).click({ button: 'right' });
  const menu = page.getByRole('menu');
  await expect(menu.getByRole('menuitem', { name: 'Start', exact: true })).toBeEnabled();
  await expect(menu.getByRole('menuitem', { name: 'Restart' })).toBeDisabled();
  await expect(menu.getByRole('menuitem', { name: 'Open site' })).toBeDisabled();
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

test('loads the blog posts once the start page shows, and each hour after', async ({ page, backend }) => {
  // Opened on a project, as `ddev gui` in its directory does.
  await page.clock.install();
  backend.newsRequests = 0;
  await page.goto(backend.url + '?again#project=d11');
  await expect(page.locator('#detail h1')).toContainText('d11');
  expect(backend.newsRequests).toBe(0);
  await page.locator('#nav-home').click();
  await expect(page.locator('.news')).toContainText('DDEV v1.25.6 is out');
  expect(backend.newsRequests).toBe(1);
  backend.news = [{ title: 'Newer post', link: 'https://ddev.com/blog/newer', date: '2026-11-27' }];
  await page.clock.fastForward('01:00:01');
  await expect(page.locator('.news')).toContainText('Newer post');
});

test('a project row in the sidebar opens, stops or starts it on hover', async ({ page, backend }) => {
  const list = page.locator('#project-list');
  const d11 = list.locator('li', { hasText: 'd11' });
  const wp = list.locator('li', { hasText: 'wp' });
  await page.locator('#nav-home').click();
  await expect(d11.getByRole('button', { name: 'Stop d11' })).toBeHidden();
  await d11.hover();
  await expect(d11.locator('.type')).toBeHidden();
  await expect(d11.getByRole('link', { name: 'Open site d11' })).toHaveAttribute('href', 'https://d11.ddev.site');
  await d11.getByRole('button', { name: 'Stop d11' }).click();
  expect((await nthPost(backend, 1)).path).toBe('/api/projects/d11/actions/stop');
  // The button acts, and leaves the page as it was.
  await expect(page.locator('#nav-home')).toHaveClass(/selected/);
  await wp.hover();
  await expect(wp.getByRole('link', { name: 'Open site wp' })).toHaveCount(0);
  await wp.getByRole('button', { name: 'Start wp' }).click();
  expect((await nthPost(backend, 2)).path).toBe('/api/projects/wp/actions/start');
});

test('the start page names the running projects, each a link to it', async ({ page, backend }) => {
  await page.goto(page.url().split('#')[0]);
  const summary = page.locator('#detail .summary');
  await expect(summary).toHaveText('2 projects, 1 running: d11.');
  await summary.getByRole('button', { name: 'd11' }).click();
  await expect(page.locator('#detail h1')).toContainText('d11');

  backend.setProjects(Array.from({ length: 20 }, (_, i) => ({
    name: `p${String(i + 1).padStart(2, '0')}`, status: i % 2 ? 'stopped' : 'running', type: 'php', url: `https://p${i + 1}.ddev.site`, approot: `/home/me/p${i + 1}`,
  })));
  await page.goto(page.url().split('#')[0]);
  await expect(summary).toHaveText('20 projects, 10 running: p01, p03, p05, p07, p09 and 5 more.');
});

test('the start page waits for the projects before it offers to add one', async ({ page, backend }) => {
  backend.setProjects([]);
  backend.holdStatus = true;
  await page.goto(page.url().split('#')[0] + '?held');
  await expect(page.locator('#detail h1')).toBeVisible();
  const add = page.locator('#detail').getByRole('button', { name: 'Add project' });
  await expect(add).toHaveCount(0);
  await expect(page.locator('#detail')).not.toContainText('Add a project to get started.');
  backend.pushStatus();
  await expect(add).toBeVisible();
});

test('the start page keeps its layout while the projects and posts load', async ({ page, backend }) => {
  backend.holdStatus = true;
  let release;
  backend.newsGate = new Promise((r) => { release = r; });
  await page.setViewportSize({ width: 1200, height: 900 });
  await page.goto(page.url().split('#')[0] + '?held');
  const tip = page.locator('.card.tip');
  const links = page.locator('.home-links');
  await expect(page.locator('.tip-text')).toBeVisible();
  const before = [(await tip.boundingBox()).y, (await links.boundingBox()).y];
  backend.pushStatus();
  await expect(page.locator('#detail .summary')).toContainText('2 projects');
  expect((await tip.boundingBox()).y).toBe(before[0]);
  release();
  await expect(page.locator('.news a')).toHaveCount(1);
  // The mock has one post; five rows stand in for the five a feed has.
  backend.news = Array.from({ length: 5 }, (_, i) => ({ title: `Post ${i}`, link: `https://ddev.com/blog/p${i}`, date: '2026-11-20', category: 'Guides' }));
  backend.newsGate = new Promise((r) => { release = r; });
  backend.holdStatus = false;
  await page.goto(page.url().split('?')[0] + '?again');
  await expect(page.locator('.tip-text')).toBeVisible();
  const waiting = (await links.boundingBox()).y;
  release();
  await expect(page.locator('.news a')).toHaveCount(5);
  expect((await links.boundingBox()).y).toBe(waiting);
});

test('shows how far the sponsors are from the goal, or nothing without data', async ({ page, backend }) => {
  await page.goto(page.url().split('#')[0]);
  const card = page.locator('.sponsor-line');
  await expect(card).toHaveText("Thanks for your ongoing support! We're up to 86.4% of our $12,000/month goal ($10,368). Join all the wonderful sponsors at https://ddev.com/sponsor");
  await expect(card.getByRole('link', { name: 'https://ddev.com/sponsor' })).toHaveAttribute('href', 'https://ddev.com/sponsor');
  // Without the message, the page says it in its own words.
  delete backend.messages.sponsorship.message;
  await page.reload();
  await expect(card).toHaveText('DDEV has reached 86.4% of its $12,000 monthly sponsorship goal. Become a sponsor');
  await expect(card.getByRole('link', { name: 'Become a sponsor' })).toHaveAttribute('href', 'https://ddev.com/sponsor');
  delete backend.messages.sponsorship;
  await page.reload();
  await expect(page.locator('.tip-text')).toBeVisible();
  await expect(card).toHaveCount(0);
});

test('brings a dismissed notice back, and new ones while the app is open', async ({ page, backend }) => {
  await page.clock.install();
  await page.reload();
  const notices = page.locator('#notices');
  await expect(notices).toContainText('Upgrade Docker soon.');
  await notices.getByRole('button', { name: 'Dismiss' }).click();
  await expect(notices).toBeEmpty();

  backend.messages.infos = [{ message: 'DDEV v9 is out.' }];
  await page.clock.fastForward('01:00:01');
  await expect(notices).toContainText('DDEV v9 is out.');
  await expect(notices).not.toContainText('Upgrade Docker soon.');
  await page.clock.fastForward('20:00:00');
  await expect(notices).toContainText('Upgrade Docker soon.');
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

  await expect(page.locator('.news')).toContainText('DDEV v1.25.6 is out');
  await expect(page.locator('.news').getByRole('link')).toHaveAttribute('href', 'https://ddev.com/blog/release-v1-25-6');
  await expect(page.locator('.news').getByRole('link')).toHaveAttribute('title', 'PHP 8.6.0 and security fixes.');
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

test('shows the colors of command output', async ({ page, backend }) => {
  const job = backend.addJob('Start d11', 'd11');
  await expect(page.locator('#activity-title .job-title')).toHaveText('Start d11');
  // A color may go on to the next line, as ddev's own do.
  backend.jobLines(job.id, ['\x1b[32mConfiguring d11.', 'For details use ddev describe.\x1b[0m', 'plain \x1b[1;33mwarning\x1b[0m']);
  backend.jobPartial(job.id, '\x1b[31mWaiting');
  const out = page.locator('#job-output');
  await expect(out.locator('.ansi-fg-2')).toHaveText(['Configuring d11.\nFor details use ddev describe.']);
  await expect(out.locator('.ansi-fg-3.ansi-bold')).toHaveText('warning');
  await expect(out.locator('.ansi-fg-1')).toHaveText('Waiting');
  await expect(out).not.toContainText('\x1b');
  backend.endJob(job.id, 'exit status 1');
  await expect(out.locator('.ansi-fg-1')).toHaveText('✗ Failed: exit status 1');
});

test('shows the shared services and follows their logs', async ({ page, backend }) => {
  await expect(page.locator('#status-text')).toHaveText('All services running');
  await expect(page.locator('#status')).toHaveAttribute('title', /SSH agent: healthy/);
  await page.locator('#status').click();
  await expect(page.locator('#detail h1')).toHaveText('Shared services');
  const cards = page.locator('.shared-cards .card');
  await expect(cards.first()).toContainText('healthy');
  await expect(cards.first().locator('.dot')).toHaveClass(/running/);
  await expect(cards.first().getByRole('link', { name: 'Open the Traefik dashboard' })).toHaveAttribute('href', 'http://127.0.0.1:10999');
  await expect(cards.first()).toContainText('docker exec -it ddev-router bash');
  await expect(cards.nth(1)).toContainText('stopped');
  await expect(cards.nth(1)).not.toContainText('docker exec');
  // Only a running container has logs to follow.
  await expect(page.getByLabel('Logs of')).toHaveText('Router');
  await expect.poll(() => backend.logRequests.at(-1)?.shared).toBe('ddev-router');
  backend.logLines(['router started']);
  await expect(page.locator('.logs-output')).toContainText('router started');

  await page.getByRole('button', { name: 'Back to the start page' }).click();
  await expect(page.locator('#nav-home')).toHaveClass(/selected/);
});

test('opens the shared services from a project\'s logs', async ({ page }) => {
  await page.getByRole('button', { name: 'Logs' }).click();
  await expect(page.locator('.logs-output')).toBeVisible();
  await page.locator('#nav-shared').click();
  await expect(page.locator('#detail h1')).toHaveText('Shared services');
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

test('acts on several projects at once', async ({ page, backend }) => {
  await page.locator('#project-list li', { hasText: 'wp' }).click({ modifiers: ['ControlOrMeta'] });
  await expect(page.locator('#detail h1')).toHaveText('2 projects selected');
  await expect(page.locator('#project-list li.selected')).toHaveCount(2);
  await expect(page.getByRole('button', { name: 'Stop 1' })).toBeEnabled();
  await page.getByRole('button', { name: 'Start 1', exact: true }).click();
  expect(await nthPost(backend, 1)).toEqual({ path: '/api/actions/start', form: { project: 'wp' } });
  // Esc goes back to the project shown before.
  await page.keyboard.press('Escape');
  await expect(page.locator('#detail h1')).toContainText('d11');
  await expect(page.locator('#project-list li.selected')).toHaveCount(1);

  // Ctrl+click on the project shown selects it.
  await page.locator('#project-list li', { hasText: 'd11' }).click({ modifiers: ['ControlOrMeta'] });
  await expect(page.locator('#detail h1')).toHaveText('1 project selected');
  await page.keyboard.press('Escape');

  // A checkbox starts a selection of its own, without the project shown.
  const wp = page.locator('#project-list li', { hasText: 'wp' });
  await wp.hover();
  await wp.locator('.lead').click();
  await expect(page.locator('#detail h1')).toHaveText('1 project selected');
  await expect(page.locator('#project-list li.selected')).toHaveCount(1);
  await expect(wp.getByRole('checkbox')).toBeChecked();
  await page.keyboard.press('Escape');

  // So does the checkbox of the project shown.
  const d11 = page.locator('#project-list li', { hasText: 'd11' });
  await d11.click();
  await d11.locator('.lead').click();
  await expect(page.locator('#detail h1')).toHaveText('1 project selected');
  await expect(d11.getByRole('checkbox')).toBeChecked();
});

test('About shows the versions and the license', async ({ page, backend }) => {
  await page.locator('#btn-more').click();
  await page.getByRole('menuitem', { name: 'About DDEV' }).click();
  const dlg = page.locator('#dialog');
  await expect(dlg.locator('h2')).toHaveText('About DDEV');
  await expect(dlg).toContainText('v1.25.6');
  await expect(dlg).toContainText('/usr/local/bin/ddev');
  await expect(dlg.getByRole('link', { name: 'Apache License 2.0' })).toBeVisible();
  await expect(dlg.getByRole('link', { name: 'ddev.com/sponsor' })).toHaveAttribute('href', 'https://ddev.com/sponsor');
  await expect(dlg).toContainText('28.0.1');
  await expect(dlg).not.toContainText('null');
  // A browser has no GUI app to name.
  await expect(dlg).not.toContainText('GUI app');
  await expect(dlg.getByRole('button', { name: 'Cancel' })).toBeHidden();
  await dlg.getByRole('button', { name: 'Close' }).click();
  await expect(dlg).toBeHidden();

  // The GUI app's tray asks through the status stream; a link works too.
  backend.desktop = true;
  backend.pushStatus();
  backend.show('about');
  await expect(dlg).toContainText('GUI app');
  await expect(dlg).toContainText('v1.0.0');
  await dlg.getByRole('button', { name: 'Close' }).click();
  await page.evaluate(() => { location.hash = 'about'; });
  await expect(dlg).toBeVisible();
  await dlg.getByRole('button', { name: 'Close' }).click();

  backend.show('project=wp');
  await expect(page.locator('#detail h1')).toContainText('wp');
});

test('the status names a stopped or failing service', async ({ page, backend }) => {
  backend.router = 'stopped';
  backend.pushStatus();
  await expect(page.locator('#status-text')).toHaveText('Router stopped');
  backend.sshAgent = 'stopped';
  backend.pushStatus();
  await expect(page.locator('#status-text')).toHaveText('Services stopped');
  backend.router = 'unhealthy';
  backend.pushStatus();
  await expect(page.locator('#status-text')).toHaveText('Router unhealthy');
  await expect(page.locator('#status')).toHaveClass(/bad/);
});
