// @ts-check
const { defineConfig, devices } = require('@playwright/test');

// "mocked" tests run the dashboard's own page against a fake API and need
// no Docker; "mocked-webkit" runs them in WebKit, the engine of the desktop
// app's window. "real" tests drive `ddev serve` with real projects.
module.exports = defineConfig({
  forbidOnly: !!process.env.CI,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    ...devices['Desktop Chrome'],
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'mocked',
      testDir: './mocked',
      fullyParallel: true,
      retries: process.env.CI ? 1 : 0,
    },
    {
      name: 'mocked-webkit',
      testDir: './mocked',
      fullyParallel: true,
      retries: process.env.CI ? 1 : 0,
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'real',
      testDir: './real',
      timeout: 20 * 60 * 1000,
      expect: { timeout: 30 * 1000 },
      use: { ignoreHTTPSErrors: true },
    },
  ],
});
