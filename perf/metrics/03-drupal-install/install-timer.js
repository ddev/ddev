#!/usr/bin/env node
// Times a Drupal demo_umami web install driven through a real browser, so the
// timing reflects the full DDEV router -> webserver -> PHP-FPM -> filesystem
// path a developer actually experiences (see perf/README.md for why this is
// kept alongside, not replaced by, the CLI-only drush-install diagnostic).
//
// Assumes the target project's database/files have already been reset (see
// perf/lib/reset-drupal.sh) and that DDEV_PERF_SITE_URL is set, e.g.
// https://d11.ddev.site/
//
// Prints one JSON metric line: {"metric":"drupal_install_s","value_s":N}

const puppeteer = require('puppeteer');

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

(async () => {
  const siteUrl = process.env.DDEV_PERF_SITE_URL;
  if (!siteUrl) {
    console.error('DDEV_PERF_SITE_URL must be set');
    process.exit(1);
  }

  // --no-sandbox: Chrome's user-namespace sandbox needs a kernel/AppArmor config
  // that's commonly locked down on Ubuntu 23.10+ and in containers.
  // --ignore-certificate-errors: the bundled Puppeteer Chrome doesn't share the
  // system/NSS trust store mkcert -install populates, so it won't trust DDEV's
  // local cert on every environment. Both are fine since this only ever
  // navigates to the project's own core/install.php on localhost.
  // protocolTimeout: Puppeteer's own CDP round-trip timeout defaults to 180s,
  // independent of the page.setDefaultTimeout(0) below -- a slow cold-start
  // install (exactly what this metric measures) can exceed that on a loaded
  // CI runner, so give it plenty of headroom instead of a silent kill.
  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--ignore-certificate-errors'],
    protocolTimeout: 600000,
  });
  try {
    const page = await browser.newPage();
    page.setDefaultTimeout(0);

    // waitForSelector/goto have no per-call timeout (see setDefaultTimeout(0)
    // above), so the only backstop against a genuine hang is protocolTimeout,
    // 10 minutes. Stage markers on stderr mean a hung run says where it's
    // stuck immediately instead of going silent for the full 10 minutes.
    const stage = (s) => console.error(`[install-timer] ${s}`);
    page.on('console', (msg) => console.error(`[install-timer] page console: ${msg.text()}`));
    page.on('pageerror', (err) => console.error(`[install-timer] page error: ${err}`));
    browser.on('disconnected', () => console.error('[install-timer] browser disconnected'));

    // Give any post-reset background work (php-fpm restart, Mutagen settle) a moment.
    await sleep(2000);

    const installUrl = new URL('core/install.php', siteUrl).toString();

    const start = Date.now();

    stage(`navigating to ${installUrl}`);
    await page.goto(installUrl);
    stage('waiting for langcode selector');
    await page.waitForSelector('#edit-langcode');
    await page.click('#edit-submit');

    stage('waiting for profile-demo-umami selector');
    await page.waitForSelector('#edit-profile-demo-umami');
    await page.click('#edit-profile-demo-umami');
    await page.click('#edit-submit');

    // Requirements/verify step only appears if Drupal has a warning to acknowledge
    // (e.g. a missing recommended PHP extension); on a clean environment it's
    // skipped straight through to the configure-site form, so don't wait on
    // #edit-save unconditionally.
    stage('waiting for requirements or configure-site form');
    await Promise.race([
      page.waitForSelector('#edit-save'),
      page.waitForSelector('#edit-site-name'),
    ]);
    const requirementsPage = await page.$('#edit-save');
    if (requirementsPage) {
      stage('acknowledging requirements warning');
      await requirementsPage.click();
    }

    // Final "configure site" form, shown once the install batch finishes.
    stage('waiting for configure-site form');
    await page.waitForSelector('#edit-site-name');
    await page.type('#edit-site-mail', 'admin@example.com');
    await page.type('#edit-account-name', 'admin');
    await page.type('#edit-account-pass-pass1', 'admin');
    await page.type('#edit-account-pass-pass2', 'admin');
    await page.click('#edit-submit');

    stage('waiting for post-install account menu');
    await page.waitForSelector('#block-umami-account-menu');
    stage('done');

    const durationS = Math.round((Date.now() - start) / 100) / 10;
    console.log(JSON.stringify({ metric: 'drupal_install_s', value_s: durationS }));
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
