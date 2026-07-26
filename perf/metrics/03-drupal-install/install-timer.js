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
// Prints one JSON metric line: {"metric":"drupal_install_ms","value_ms":N}

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

  const browser = await puppeteer.launch({ headless: true });
  try {
    const page = await browser.newPage();
    page.setDefaultTimeout(0);

    // Give any post-reset background work (php-fpm restart, Mutagen settle) a moment.
    await sleep(2000);

    const installUrl = new URL('core/install.php', siteUrl).toString();

    const start = Date.now();

    await page.goto(installUrl);
    await page.waitForSelector('#edit-langcode');
    await page.click('#edit-submit');

    await page.waitForSelector('#edit-profile-demo-umami');
    await page.click('#edit-profile-demo-umami');
    await page.click('#edit-submit');

    // Requirements/verify step: "Save and continue" kicks off the install batch.
    await page.waitForSelector('#edit-save');
    await page.click('#edit-save');

    // Final "configure site" form, shown once the install batch finishes.
    await page.waitForSelector('#edit-site-name');
    await page.type('#edit-site-mail', 'admin@example.com');
    await page.type('#edit-account-name', 'admin');
    await page.type('#edit-account-pass-pass1', 'admin');
    await page.type('#edit-account-pass-pass2', 'admin');
    await page.click('#edit-submit');

    await page.waitForSelector('#block-umami-account-menu');

    const durationMs = Date.now() - start;
    console.log(JSON.stringify({ metric: 'drupal_install_ms', value_ms: durationMs }));
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
