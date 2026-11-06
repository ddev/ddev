'use strict';

const base = require('@playwright/test');
const { MockBackend } = require('./mock-backend');

const running = {
  name: 'd11', status: 'running', type: 'drupal11', url: 'https://d11.ddev.site', approot: '/home/me/projects/d11',
};
const stopped = {
  name: 'wp', status: 'stopped', type: 'wordpress', approot: '/home/me/projects/my wp',
};

function describe(p, extra) {
  const isRunning = p.status === 'running';
  return {
    name: p.name,
    status: p.status,
    type: p.type,
    approot: p.approot,
    php_version: '8.4',
    webserver_type: 'nginx-fpm',
    nodejs_version: '24',
    docroot: 'web',
    database_type: 'mariadb',
    database_version: '11.8',
    performance_mode: 'mutagen',
    mutagen_status: 'ok',
    xhprof_mode: 'xhgui',
    xhgui_status: 'disabled',
    router: 'traefik',
    primary_url: `https://${p.name}.ddev.site`,
    urls: [`https://${p.name}.ddev.site`, 'https://127.0.0.1:32770'],
    mailpit_https_url: `https://${p.name}.ddev.site:8026`,
    dbinfo: {
      dbname: 'db', username: 'db', password: 'db', host: 'db', dbPort: '3306',
      published_port: isRunning ? 32771 : 0, database_type: 'mariadb', database_version: '11.8',
    },
    services: {
      web: {
        status: p.status, https_url: `https://${p.name}.ddev.site`, host_https_url: 'https://127.0.0.1:32770',
        host_ports_mapping: isRunning ? [{ exposed_port: '80', host_port: '32769' }] : [], exposed_ports: '80,443', image: 'ddev/ddev-webserver:v1',
      },
      db: { status: p.status, exposed_ports: '3306', image: 'ddev/ddev-dbserver-mariadb-11.8:v1' },
      redis: { status: p.status, exposed_ports: '6379', image: 'redis:8', 'describe-info': 'Pass: <none>' },
    },
    ...extra,
  };
}

function detail(p, extra = {}) {
  return {
    describe: describe(p, extra.describe),
    addons: extra.addons ?? [{ name: 'redis', version: 'v3.0.0', repository: 'ddev/ddev-redis' }],
    snapshots: extra.snapshots ?? [
      { name: `${p.name}_new`, label: `${p.name}_new`, path: `${p.name}_new`, created: '2026-11-06 14:31', size: 2621440 },
      { name: `${p.name}_old`, label: `${p.name}_old`, path: `${p.name}_old`, created: '2026-11-01 09:00', size: 1048576 },
    ],
    xdebug: p.status === 'running' ? (extra.xdebug ?? 'off') : undefined,
  };
}

const test = base.test.extend({
  backend: async ({}, use) => {
    const b = new MockBackend();
    b.projects = [running, stopped];
    b.details = { d11: detail(running), wp: detail(stopped) };
    await b.start();
    await use(b);
    await b.stop();
  },
  // The page opens the dashboard, and the test fails on any script error
  // or Content-Security-Policy violation.
  page: async ({ page, backend }, use) => {
    const problems = [];
    page.on('pageerror', (e) => problems.push('page error: ' + e.message));
    page.on('console', (m) => {
      if (m.type() === 'error' && !m.text().startsWith('Failed to load resource')) problems.push('console: ' + m.text());
    });
    await page.addInitScript(() => localStorage.clear());
    await page.goto(backend.url);
    await use(page);
    base.expect(problems).toEqual([]);
  },
});

// nthPost waits for the dashboard's nth POST and returns it.
async function nthPost(backend, n) {
  await base.expect.poll(() => backend.posts.length).toBeGreaterThanOrEqual(n);
  return backend.posts[n - 1];
}

module.exports = { test, expect: base.expect, running, stopped, detail, nthPost };
