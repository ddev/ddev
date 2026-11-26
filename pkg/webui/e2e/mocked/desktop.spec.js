'use strict';

// The GUI app shows this page in a webview that opens no new windows,
// so links that would open a tab go to the server instead.
const { test, expect, nthPost, reload } = require('./fixtures');

test('sends every state change with the dashboard header', async ({ page, backend }) => {
  await page.locator('.toolbar').getByRole('button', { name: 'Restart', exact: true }).click();
  await nthPost(backend, 1);
  expect(backend.postHeaders[0]['x-ddev-dashboard']).toBe('1');
});

test('opens links through the server in the GUI app', async ({ page, backend }) => {
  backend.desktop = true;
  await reload(page);
  const opened = page.waitForEvent('popup', { timeout: 1000 }).then(() => true, () => false);
  await page.locator('.toolbar').getByRole('link', { name: 'Open site' }).click();
  const post = await nthPost(backend, 1);
  expect(post).toMatchObject({ path: '/api/open-url', form: { url: 'https://d11.ddev.site/' } });
  expect(await opened).toBe(false);
});

test('Open site on a project row opens the site from the GUI app', async ({ page, backend }) => {
  backend.desktop = true;
  await reload(page);
  await page.locator('#nav-home').click();
  const row = page.locator('#project-list li', { hasText: 'd11' });
  await row.hover();
  await row.getByRole('link', { name: 'Open site d11' }).click();
  expect(await nthPost(backend, 1)).toMatchObject({ path: '/api/open-url', form: { url: 'https://d11.ddev.site/' } });
  await expect(page.locator('#nav-home')).toHaveClass(/selected/);
});

test('the ⋯ menu links to the docs and to sponsoring DDEV', async ({ page, backend }) => {
  backend.desktop = true;
  await reload(page);
  await page.locator('#btn-more').click();
  const menu = page.locator('.menu');
  await expect(menu.getByRole('menuitem')).toHaveText(['Refresh', 'DDEV docs', 'Sponsor DDEV', 'About DDEV']);
  await menu.getByRole('menuitem', { name: 'Sponsor DDEV' }).click();
  expect(await nthPost(backend, 1)).toMatchObject({ path: '/api/open-url', form: { url: 'https://ddev.com/sponsor' } });
});

test('right-click on a link or selected text opens the page menu in the GUI app', async ({ page, backend }) => {
  backend.desktop = true;
  await reload(page);
  await page.locator('.toolbar').getByRole('link', { name: 'Open site' }).click({ button: 'right' });
  const menu = page.locator('.menu');
  await expect(menu.getByRole('menuitem')).toHaveText(['Open link', 'Copy link address']);
  await menu.getByRole('menuitem', { name: 'Open link' }).click();
  expect(await nthPost(backend, 1)).toMatchObject({ path: '/api/open-url', form: { url: 'https://d11.ddev.site/' } });

  await page.locator('.detail-head h1').selectText();
  await page.locator('.detail-head h1').click({ button: 'right', position: { x: 3, y: 5 } });
  await expect(menu.getByRole('menuitem')).toHaveText(['Copy']);
  await page.keyboard.press('Escape');
  // Elsewhere there is no menu at all.
  await page.locator('#detail').click({ button: 'right', position: { x: 5, y: 300 } });
  await expect(menu).toHaveCount(0);
});

test('lets the browser open links outside the GUI app', async ({ page, backend }) => {
  const popup = page.waitForEvent('popup');
  await page.locator('.toolbar').getByRole('link', { name: 'Open site' }).click();
  await popup;
  expect(backend.posts).toHaveLength(0);
});
