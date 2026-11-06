'use strict';

// The desktop app shows this page in a webview that opens no new windows,
// so links that would open a tab go to the server instead.
const { test, expect, nthPost } = require('./fixtures');

test('sends every state change with the dashboard header', async ({ page, backend }) => {
  await page.locator('.toolbar').getByRole('button', { name: 'Restart', exact: true }).click();
  await nthPost(backend, 1);
  expect(backend.postHeaders[0]['x-ddev-dashboard']).toBe('1');
});

test('opens links through the server in the desktop app', async ({ page, backend }) => {
  backend.desktop = true;
  await page.reload();
  const opened = page.waitForEvent('popup', { timeout: 1000 }).then(() => true, () => false);
  await page.locator('.toolbar').getByRole('link', { name: 'Open site ↗' }).click();
  const post = await nthPost(backend, 1);
  expect(post).toMatchObject({ path: '/api/open-url', form: { url: 'https://d11.ddev.site/' } });
  expect(await opened).toBe(false);
});

test('lets the browser open links outside the desktop app', async ({ page, backend }) => {
  const popup = page.waitForEvent('popup');
  await page.locator('.toolbar').getByRole('link', { name: 'Open site ↗' }).click();
  await popup;
  expect(backend.posts).toHaveLength(0);
});
