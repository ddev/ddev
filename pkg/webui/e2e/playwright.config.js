// @ts-check
const { defineConfig, devices } = require('@playwright/test');

// "mocked" tests run the dashboard's own page against a fake API and need
// no Docker. "real" tests drive `ddev serve` with real projects.
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
      name: 'real',
      testDir: './real',
      timeout: 20 * 60 * 1000,
      expect: { timeout: 30 * 1000 },
      use: { ignoreHTTPSErrors: true },
    },
  ],
});
