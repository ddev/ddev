// A stand-in for the `ddev serve` API. It serves the real page files from
// ../../assets and keeps event streams open, so a test decides when status,
// job output and log lines arrive.
'use strict';

const fs = require('node:fs');
const http = require('node:http');
const path = require('node:path');

const ASSETS = path.join(__dirname, '..', '..', 'assets');
// Same policy as secure() in pkg/webui/server.go, so a page change that
// breaks it fails here.
const CSP = "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'";
const TYPES = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.svg': 'image/svg+xml' };

class MockBackend {
  constructor() {
    this.projects = [];
    this.details = {};
    this.router = 'healthy';
    this.error = '';
    this.jobs = [];
    this.posts = [];
    this.registry = [
      { repo: 'ddev/ddev-redis', title: 'ddev-redis', description: 'Redis', type: 'official' },
      { repo: 'ddev/ddev-solr', title: 'ddev-solr', description: 'Solr', type: 'official' },
    ];
    this.projectTypes = ['drupal11', 'laravel', 'php', 'wordpress'];
    // postReply lets a test answer a POST itself, returning {status, body}.
    this.postReply = null;
    this.statusClients = new Set();
    this.jobClients = new Map();
    this.logClients = new Set();
    this.logRequests = [];
    this.nextJob = 1;
  }

  async start() {
    this.server = http.createServer((req, res) => {
      this.handle(req, res).catch((e) => {
        console.error('mock backend:', e);
        if (!res.headersSent) text(res, 500, String(e));
      });
    });
    await new Promise((resolve) => this.server.listen(0, '127.0.0.1', resolve));
    this.url = `http://127.0.0.1:${this.server.address().port}/`;
  }

  async stop() {
    for (const res of [...this.statusClients, ...this.logClients, ...[...this.jobClients.values()].flat()]) res.end();
    this.server.closeAllConnections();
    await new Promise((resolve) => this.server.close(resolve));
  }

  // ---- test controls ----

  setProjects(projects) {
    this.projects = projects;
    this.pushStatus();
  }

  pushStatus() {
    const status = { projects: this.projects, router: this.router, jobs: this.jobs.map(summary), error: this.error || undefined };
    for (const res of this.statusClients) send(res, 'status', status);
  }

  addJob(title, project) {
    const job = { id: this.nextJob++, title, project, started: new Date().toISOString(), done: false, err: '', lines: [] };
    this.jobs.push(job);
    this.pushStatus();
    return job;
  }

  jobLines(id, lines) {
    const job = this.job(id);
    job.lines.push(...lines);
    for (const res of this.jobClients.get(id) || []) send(res, 'lines', lines);
  }

  endJob(id, err = '') {
    const job = this.job(id);
    job.done = true;
    job.err = err;
    for (const res of this.jobClients.get(id) || []) {
      send(res, 'end', { err });
      res.end();
    }
    this.jobClients.delete(id);
    this.pushStatus();
  }

  logLines(lines) {
    for (const res of this.logClients) send(res, 'lines', lines);
  }

  job(id) {
    const job = this.jobs.find((j) => j.id === id);
    if (!job) throw new Error(`no job ${id}`);
    return job;
  }

  // ---- HTTP ----

  async handle(req, res) {
    const url = new URL(req.url, 'http://x');
    res.setHeader('Content-Security-Policy', CSP);
    res.setHeader('Cache-Control', 'no-store');
    if (req.method === 'POST') return this.handlePost(req, res, url);
    const p = url.pathname;
    let m;
    if (p === '/api/events') {
      sse(res);
      this.statusClients.add(res);
      req.on('close', () => this.statusClients.delete(res));
      this.pushStatus();
    } else if ((m = p.match(/^\/api\/jobs\/(\d+)$/))) {
      const job = this.jobs.find((j) => j.id === Number(m[1]));
      if (!job) return text(res, 404, 'not found');
      sse(res);
      if (job.lines.length) send(res, 'lines', job.lines);
      if (job.done) {
        send(res, 'end', { err: job.err });
        return res.end();
      }
      const list = this.jobClients.get(job.id) || [];
      list.push(res);
      this.jobClients.set(job.id, list);
    } else if ((m = p.match(/^\/api\/projects\/([^/]+)\/logs$/))) {
      this.logRequests.push({ project: decodeURIComponent(m[1]), service: url.searchParams.get('service') });
      sse(res);
      this.logClients.add(res);
      req.on('close', () => this.logClients.delete(res));
    } else if ((m = p.match(/^\/api\/projects\/([^/]+)$/))) {
      const d = this.details[decodeURIComponent(m[1])];
      if (!d) return text(res, 404, 'no project named ' + m[1]);
      json(res, 200, d);
    } else if (p === '/api/project-types') {
      json(res, 200, this.projectTypes);
    } else if (p === '/api/addons/registry') {
      json(res, 200, this.registry);
    } else {
      this.serveAsset(res, p === '/' ? '/index.html' : p);
    }
  }

  async handlePost(req, res, url) {
    let body = '';
    for await (const chunk of req) body += chunk;
    const post = { path: url.pathname, form: Object.fromEntries(new URLSearchParams(body)) };
    this.posts.push(post);
    const reply = this.postReply && this.postReply(post);
    if (reply) {
      return typeof reply.body === 'string' ? text(res, reply.status, reply.body) : json(res, reply.status, reply.body ?? {});
    }
    if (url.pathname.includes('/open/')) {
      res.writeHead(204);
      return res.end();
    }
    const m = url.pathname.match(/^\/api\/projects\/([^/]+)\/actions\/(.+)$/);
    const job = this.addJob(m ? `${m[2]} ${decodeURIComponent(m[1])}` : url.pathname, m ? decodeURIComponent(m[1]) : '');
    json(res, 202, summary(job));
  }

  serveAsset(res, p) {
    const file = path.join(ASSETS, path.normalize(p));
    if (!file.startsWith(ASSETS) || !fs.existsSync(file)) return text(res, 404, 'not found');
    res.writeHead(200, { 'Content-Type': TYPES[path.extname(file)] || 'application/octet-stream' });
    res.end(fs.readFileSync(file));
  }
}

function summary(j) {
  return { id: j.id, title: j.title, project: j.project, started: j.started, done: j.done, err: j.err || undefined };
}

function sse(res) {
  res.writeHead(200, { 'Content-Type': 'text/event-stream' });
  res.flushHeaders();
}

function send(res, event, data) {
  res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
}

function json(res, status, v) {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(v));
}

function text(res, status, s) {
  res.writeHead(status, { 'Content-Type': 'text/plain' });
  res.end(s);
}

module.exports = { MockBackend };
