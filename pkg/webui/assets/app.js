// DDEV dashboard for `ddev serve`. Plain DOM code, no build step: the page
// runs under a strict Content-Security-Policy, so nothing is inlined and all
// text goes in through textContent.
'use strict';

const state = {
  projects: [],
  jobs: [],
  router: '',
  selected: localStorage.getItem('ddev.selected') || '',
  filter: '',
  detail: null,
  detailFor: '',
  detailStatus: '',
  tab: localStorage.getItem('ddev.tab') || 'overview',
  jobId: 0,
  jobSource: null,
  logSource: null,
  desktop: false,
  // xdebugWanted holds a project's requested Xdebug state until its detail
  // is reloaded, so the switch does not flip back while the job runs.
  xdebugWanted: {},
  logService: 'web',
  registry: null,
  types: null,
  seenJobs: new Set(),
};

// h builds an element. Attributes starting with "on" are listeners.
function h(tag, attrs, ...children) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v === undefined || v === null || v === false) continue;
    if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'class') el.className = v;
    else if (v === true) el.setAttribute(k, '');
    else el.setAttribute(k, v);
  }
  for (const c of children.flat()) {
    if (c === undefined || c === null || c === false) continue;
    el.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return el;
}

const $ = (id) => document.getElementById(id);

function toast(text, isError) {
  const t = h('div', { class: 'toast' + (isError ? ' error' : '') }, text);
  $('toasts').append(t);
  setTimeout(() => t.remove(), isError ? 9000 : 4000);
}

async function post(url, params) {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'X-DDEV-Dashboard': '1' },
    body: new URLSearchParams(params || {}),
  });
  if (!res.ok) {
    throw new Error((await res.text()).trim() || res.statusText);
  }
  return res.status === 204 ? null : res.json();
}

async function runJob(url, params) {
  try {
    const job = await post(url, params);
    showJob(job.id, true);
  } catch (e) {
    toast(e.message, true);
  }
}

function copy(text) {
  navigator.clipboard.writeText(text).then(
    () => toast('Copied: ' + text),
    () => toast('Copy failed', true),
  );
}

function copyable(text, display) {
  return h('span', { class: 'copyable' }, display || text,
    h('button', { class: 'copy-btn', title: 'Copy', type: 'button', onclick: () => copy(text) }, '⧉'));
}

function link(url, text) {
  return h('a', { href: url, target: '_blank', rel: 'noopener noreferrer' }, text || url);
}

// Status classes: running, paused/starting (amber), stopped (grey), the
// rest red.
function statusClass(status) {
  if (!status) return '';
  if (status === 'running') return 'running';
  if (status === 'stopped') return '';
  if (status === 'paused' || status.startsWith('starting')) return 'paused';
  return 'bad';
}

function runningJobFor(name) {
  return state.jobs.find((j) => !j.done && (j.project === name || j.project === ''));
}

// ---------- status stream ----------

function connectEvents() {
  const es = new EventSource('/api/events');
  es.addEventListener('open', () => setConn('connected', 'ok'));
  es.addEventListener('error', () => setConn('disconnected, retrying…', 'bad'));
  es.addEventListener('status', (ev) => {
    const st = JSON.parse(ev.data);
    state.projects = st.projects || [];
    state.jobs = st.jobs || [];
    state.router = st.router || '';
    state.desktop = !!st.desktop;
    const banner = $('error-banner');
    banner.textContent = st.error || '';
    banner.classList.toggle('hidden', !st.error);
    renderRouter();
    renderProjects();
    renderJobs();
    const p = selectedProject();
    if (p && (p.name !== state.detailFor || p.status !== state.detailStatus)) {
      loadDetail();
    } else if (p) {
      renderDetail();
    }
    if (!p && state.selected && state.projects.length && !st.error) {
      selectProject(state.projects[0].name);
    }
  });
}

function setConn(text, cls) {
  const el = $('conn-status');
  el.textContent = text;
  el.className = 'pill ' + cls;
}

function renderRouter() {
  const el = $('router-status');
  el.textContent = 'router: ' + (state.router || 'unknown');
  el.className = 'pill ' + (state.router === 'healthy' ? 'ok' : state.router === 'disabled' || !state.router ? '' : 'warn');
}

// ---------- project list ----------

function selectedProject() {
  return state.projects.find((p) => p.name === state.selected);
}

function selectProject(name) {
  if (state.selected === name) return;
  state.selected = name;
  localStorage.setItem('ddev.selected', name);
  state.detail = null;
  state.detailFor = '';
  stopLogs();
  renderProjects();
  renderDetail();
  loadDetail();
}

function renderProjects() {
  const list = $('project-list');
  const f = state.filter.toLowerCase();
  const shown = state.projects.filter((p) =>
    !f || p.name.toLowerCase().includes(f) || p.type.toLowerCase().includes(f) || p.status.toLowerCase().includes(f));
  list.replaceChildren(...shown.map((p) => {
    const busy = runningJobFor(p.name);
    return h('li', {
      class: p.name === state.selected ? 'selected' : '',
      onclick: () => selectProject(p.name),
      title: p.approot,
    },
    h('span', { class: 'dot ' + (busy ? 'busy' : statusClass(p.status)) }),
    h('span', { class: 'name' }, p.name),
    h('span', { class: 'type' }, p.type),
    h('span', { class: 'sub' }, busy ? busy.title + '…' : (p.url || p.status)));
  }));
  $('project-empty').classList.toggle('hidden', state.projects.length > 0);
  if (!state.selected && state.projects.length) selectProject(state.projects[0].name);
}

// ---------- project detail ----------

let detailSeq = 0;
async function loadDetail() {
  const p = selectedProject();
  if (!p) return;
  const seq = ++detailSeq;
  state.detailFor = p.name;
  state.detailStatus = p.status;
  try {
    const res = await fetch('/api/projects/' + encodeURIComponent(p.name));
    if (!res.ok) throw new Error((await res.text()).trim());
    const d = await res.json();
    if (seq !== detailSeq) return;
    state.detail = d;
    delete state.xdebugWanted[p.name];
  } catch (e) {
    if (seq !== detailSeq) return;
    state.detail = { error: e.message };
  }
  renderDetail();
}

// detailKey is what the panel was last built from. Rebuilding it on every
// status event would replace a control the user is clicking or typing in.
let detailKey = '';

function renderDetail() {
  const root = $('detail');
  const p = selectedProject();
  const key = JSON.stringify([p, state.tab, state.detail, state.xdebugWanted, state.jobs.filter((j) => !j.done).map((j) => [j.id, j.project])]);
  if (key === detailKey && root.childElementCount) return;
  detailKey = key;
  if (!p) {
    root.replaceChildren(h('div', { class: 'placeholder muted' }, state.projects.length ? 'Select a project.' : ''));
    return;
  }
  // The logs tab keeps its stream, so it is not rebuilt on every status.
  if (state.tab === 'logs' && root.querySelector('.logs-output') && root.dataset.project === p.name) {
    root.querySelector('.detail-head').replaceWith(detailHead(p));
    root.querySelector('.toolbar').replaceWith(toolbar(p));
    return;
  }
  root.dataset.project = p.name;
  const tabs = [['overview', 'Overview'], ['services', 'Services'], ['database', 'Database'],
    ['addons', 'Add-ons'], ['snapshots', 'Snapshots'], ['logs', 'Logs']];
  const tabBar = h('nav', { class: 'tabs' }, tabs.map(([id, label]) =>
    h('button', { class: state.tab === id ? 'active' : '', type: 'button', onclick: () => setTab(id) }, label)));
  let body;
  const d = state.detail;
  if (state.tab === 'logs') {
    body = logsTab(p);
  } else if (!d) {
    body = h('p', { class: 'muted' }, 'Loading…');
  } else if (d.error) {
    body = h('p', { class: 'muted' }, d.error);
  } else {
    body = ({ overview: overviewTab, services: servicesTab, database: databaseTab,
      addons: addonsTab, snapshots: snapshotsTab })[state.tab](p, d);
  }
  root.replaceChildren(detailHead(p), toolbar(p), tabBar, body);
}

function setTab(id) {
  if (state.tab === 'logs' && id !== 'logs') stopLogs();
  state.tab = id;
  localStorage.setItem('ddev.tab', id);
  renderDetail();
}

function detailHead(p) {
  const busy = runningJobFor(p.name);
  return h('div', { class: 'detail-head' },
    h('div', {},
      h('h1', {}, p.name,
        h('span', { class: 'pill ' + (p.status === 'running' ? 'ok' : statusClass(p.status) === 'bad' ? 'bad' : '') }, busy ? busy.title + '…' : p.status),
        h('span', { class: 'badge' }, p.type)),
      h('div', { class: 'path' }, copyable(p.approot))));
}

function toolbar(p) {
  const d = state.detail && !state.detail.error ? state.detail.describe : null;
  const busy = !!runningJobFor(p.name);
  const running = p.status === 'running';
  const act = (a, params) => () => runJob('/api/projects/' + encodeURIComponent(p.name) + '/actions/' + a, params);
  const open = (t) => async () => {
    try { await post('/api/projects/' + encodeURIComponent(p.name) + '/open/' + t); } catch (e) { toast(e.message, true); }
  };
  const mailpit = d && (d.mailpit_https_url || d.mailpit_url);
  const xhgui = d && d.xhgui_status === 'enabled' && (d.xhgui_https_url || d.xhgui_url);
  return h('div', { class: 'toolbar' },
    running
      ? h('button', { class: 'btn', disabled: busy, onclick: act('stop') }, 'Stop')
      : h('button', { class: 'btn primary', disabled: busy, onclick: act('start') }, 'Start'),
    h('button', { class: 'btn', disabled: busy, onclick: act('restart') }, 'Restart'),
    h('span', { class: 'sep' }),
    h('a', { class: 'btn', href: p.url || '#', target: '_blank', rel: 'noopener noreferrer', hidden: !p.url }, 'Open site ↗'),
    mailpit && running ? h('a', { class: 'btn', href: mailpit, target: '_blank', rel: 'noopener noreferrer' }, 'Mailpit ↗') : null,
    xhgui && running ? h('a', { class: 'btn', href: xhgui, target: '_blank', rel: 'noopener noreferrer' }, 'XHGui ↗') : null,
    h('span', { class: 'sep' }),
    h('button', { class: 'btn', onclick: open('folder') }, 'Folder'),
    h('button', { class: 'btn', onclick: open('vscode') }, 'VS Code'),
    h('button', { class: 'btn', onclick: open('phpstorm') }, 'PhpStorm'),
    h('span', { class: 'sep' }),
    h('button', { class: 'btn danger', disabled: busy, onclick: () => confirmDelete(p) }, 'Delete'));
}

function kv(rows) {
  return h('table', { class: 'kv' }, rows.filter((r) => r && r[1] !== undefined && r[1] !== '' && r[1] !== null)
    .map(([k, v]) => h('tr', {}, h('th', {}, k), h('td', {}, v))));
}

function overviewTab(p, d) {
  const desc = d.describe;
  const running = p.status === 'running';
  const urls = desc.urls || [];
  const shell = `cd ${shellQuote(p.approot)} && ddev ssh`;
  const xdebugOn = d.xdebug === 'on';
  const wanted = state.xdebugWanted[p.name];
  return h('div', { class: 'cards' },
    h('div', { class: 'card' }, h('h3', {}, 'URLs'),
      running && urls.length
        ? h('ul', { class: 'url-list' }, urls.map((u) => h('li', {}, link(u), ' ', h('button', { class: 'copy-btn', type: 'button', onclick: () => copy(u) }, '⧉'))))
        : h('p', { class: 'muted' }, 'Start the project to get its URLs.')),
    h('div', { class: 'card' }, h('h3', {}, 'Configuration'), kv([
      ['PHP', desc.php_version],
      ['Web server', desc.webserver_type],
      ['Node.js', desc.nodejs_version],
      ['Docroot', desc.docroot || '(project root)'],
      ['Database', desc.database_type ? `${desc.database_type} ${desc.database_version}` : 'none'],
      ['Performance', desc.performance_mode],
      ['Mutagen', running ? desc.mutagen_status : undefined],
      ['XHProf mode', desc.xhprof_mode],
      ['Router', desc.router_disabled ? 'disabled' : desc.router],
    ])),
    h('div', { class: 'card' }, h('h3', {}, 'Debugging'),
      h('label', { class: 'switch' },
        h('input', {
          type: 'checkbox',
          checked: wanted ?? xdebugOn,
          disabled: !running || !!runningJobFor(p.name),
          onchange: (e) => {
            state.xdebugWanted[p.name] = e.target.checked;
            runJob('/api/projects/' + encodeURIComponent(p.name) + '/actions/xdebug-' + (e.target.checked ? 'on' : 'off'));
          },
        }),
        'Xdebug ', h('span', { class: 'muted' }, !running ? '(project stopped)'
          : wanted !== undefined ? (wanted ? 'enabling…' : 'disabling…') : (xdebugOn ? 'enabled' : 'disabled')))),
    h('div', { class: 'card' }, h('h3', {}, 'Terminal'),
      h('p', { class: 'muted' }, 'A browser cannot host a shell. Paste this into a terminal:'),
      copyable(shell)));
}

function shellQuote(s) {
  return /^[\w@%+=:,./~-]+$/.test(s) ? s : "'" + s.replaceAll("'", "'\\''") + "'";
}

function servicesTab(p, d) {
  const services = Object.entries(d.describe.services || {});
  services.sort(([a], [b]) => (a === 'web' ? -1 : b === 'web' ? 1 : a === 'db' ? -1 : b === 'db' ? 1 : a.localeCompare(b)));
  if (!services.length) return h('p', { class: 'muted' }, 'No services.');
  return h('table', { class: 'grid' },
    h('tr', {}, h('th', {}, 'Service'), h('th', {}, 'Status'), h('th', {}, 'URL'), h('th', {}, 'Ports'), h('th', {}, 'Image')),
    services.map(([name, s]) => {
      const url = s.https_url || s.http_url || s.host_https_url || s.host_http_url;
      const ports = (s.host_ports_mapping || []).map((m) => `${m.exposed_port}→${m.host_port}`).join(', ') || s.exposed_ports || '';
      return h('tr', {},
        h('td', {}, h('strong', {}, name), s['describe-info'] ? h('div', { class: 'muted' }, s['describe-info']) : null),
        h('td', {}, h('span', { class: 'dot ' + statusClass(s.status) }), ' ', s.status),
        h('td', {}, url && s.status === 'running' ? link(url) : ''),
        h('td', {}, ports),
        h('td', { class: 'muted' }, s.image || ''));
    }));
}

function databaseTab(p, d) {
  const db = d.describe.dbinfo;
  if (!db) return h('p', { class: 'muted' }, 'This project has no database container.');
  const host = (d.describe.services?.web?.host_https_url || d.describe.services?.web?.host_http_url || '').replace(/^https?:\/\/([^:/]+).*$/, '$1') || '127.0.0.1';
  const cli = (db.database_type || d.describe.database_type) === 'postgres' ? 'ddev psql' : 'ddev mysql';
  return h('div', { class: 'cards' },
    h('div', { class: 'card' }, h('h3', {}, 'Connection'), kv([
      ['Type', `${db.database_type || d.describe.database_type} ${db.database_version || d.describe.database_version}`],
      ['Database', db.dbname && copyable(db.dbname)],
      ['User', db.username && copyable(db.username)],
      ['Password', db.password && copyable(db.password)],
      ['From this project', copyable(`db:${db.dbPort}`)],
      ['From other projects', copyable(`ddev-${p.name}-db:${db.dbPort}`)],
      ['From the host', db.published_port ? copyable(`${host}:${db.published_port}`) : (p.status === 'running' ? 'not published' : 'start the project')],
    ])),
    h('div', { class: 'card' }, h('h3', {}, 'Tools'),
      h('p', { class: 'muted' }, 'Open a database client in a terminal:'),
      copyable(`cd ${shellQuote(p.approot)} && ${cli}`),
      h('p', { class: 'muted' }, 'Export the database:'),
      copyable(`cd ${shellQuote(p.approot)} && ddev export-db --file=db.sql.gz`)));
}

function addonsTab(p, d) {
  const busy = !!runningJobFor(p.name);
  const act = (a, params) => runJob('/api/projects/' + encodeURIComponent(p.name) + '/actions/' + a, params);
  const input = h('input', { type: 'text', list: 'addon-registry', placeholder: 'ddev/ddev-redis', 'aria-label': 'Add-on repository' });
  const datalist = h('datalist', { id: 'addon-registry' });
  loadRegistry().then((addons) => {
    datalist.replaceChildren(...addons.map((a) => h('option', { value: a.repo }, `${a.title || a.repo}${a.type === 'official' ? ' (official)' : ''}`)));
  });
  return h('div', {},
    h('form', {
      class: 'row',
      onsubmit: (e) => {
        e.preventDefault();
        if (input.value.trim()) act('addon-get', { repo: input.value.trim() });
      },
    }, input, datalist,
    h('button', { class: 'btn primary', disabled: busy }, 'Install'),
    h('button', { class: 'btn', type: 'button', disabled: busy || !d.addons.length, onclick: () => act('addon-update') }, 'Update all'),
    h('a', { href: 'https://addons.ddev.com', target: '_blank', rel: 'noopener noreferrer' }, 'Browse add-ons ↗')),
    d.addons.length
      ? h('table', { class: 'grid' },
        h('tr', {}, h('th', {}, 'Add-on'), h('th', {}, 'Version'), h('th', {}, 'Repository'), h('th', {}, '')),
        d.addons.map((a) => h('tr', {},
          h('td', {}, h('strong', {}, a.name)),
          h('td', {}, a.version),
          h('td', {}, a.repository ? link('https://github.com/' + a.repository, a.repository) : ''),
          h('td', {}, h('button', {
            class: 'btn small danger',
            disabled: busy,
            onclick: () => confirmDialog(`Remove ${a.name}?`, `This runs “ddev add-on remove ${a.name}” in ${p.name}.`, 'Remove')
              .then((ok) => ok && act('addon-remove', { addon: a.name })),
          }, 'Remove')))))
      : h('p', { class: 'muted' }, 'No add-ons installed.'));
}

async function loadRegistry() {
  if (!state.registry) {
    state.registry = fetch('/api/addons/registry').then((r) => (r.ok ? r.json() : [])).catch(() => []);
  }
  return state.registry;
}

function snapshotsTab(p, d) {
  const busy = !!runningJobFor(p.name);
  const act = (a, params) => runJob('/api/projects/' + encodeURIComponent(p.name) + '/actions/' + a, params);
  const restore = (s) => confirmDialog(`Restore ${s ? s.label : 'the latest snapshot'}?`,
    `This replaces the current database of ${p.name}.`, 'Restore')
    .then((ok) => ok && act('restore', { snapshot: s ? s.path : '' }));
  return h('div', {},
    h('div', { class: 'row' },
      h('button', { class: 'btn primary', disabled: busy || !d.describe.dbinfo, onclick: () => act('snapshot') }, 'Create snapshot'),
      h('button', { class: 'btn', disabled: busy || !d.snapshots.length, onclick: () => restore(null) }, 'Restore latest')),
    d.snapshots.length
      ? h('table', { class: 'grid' },
        h('tr', {}, h('th', {}, 'Snapshot'), h('th', {}, 'Created'), h('th', {}, 'Size'), h('th', {}, '')),
        d.snapshots.map((s) => h('tr', {},
          h('td', {}, s.label),
          h('td', {}, s.created),
          h('td', {}, formatSize(s.size)),
          h('td', {}, h('button', { class: 'btn small', disabled: busy, onclick: () => restore(s) }, 'Restore')))))
      : h('p', { class: 'muted' }, 'No snapshots.'));
}

function formatSize(n) {
  if (!n) return '';
  const units = ['B', 'KB', 'MB', 'GB'];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
  return `${n.toFixed(i ? 1 : 0)} ${units[i]}`;
}

// ---------- logs ----------

function logsTab(p) {
  const services = Object.keys(state.detail?.describe?.services || { web: {}, db: {} });
  if (!services.includes(state.logService)) state.logService = 'web';
  const out = h('pre', { class: 'output logs-output' });
  const select = h('select', {
    onchange: (e) => { state.logService = e.target.value; startLogs(p, out); },
    'aria-label': 'Service',
  }, services.map((s) => h('option', { value: s, selected: s === state.logService }, s)));
  const view = h('div', {},
    h('div', { class: 'row' }, 'Service', select,
      h('button', { class: 'btn', onclick: () => startLogs(p, out) }, 'Reconnect'),
      h('button', { class: 'btn', onclick: () => { out.textContent = ''; } }, 'Clear')),
    out);
  startLogs(p, out);
  return view;
}

function startLogs(p, out) {
  stopLogs();
  out.textContent = '';
  if (p.status !== 'running') {
    out.textContent = 'The project is not running.';
    return;
  }
  const es = new EventSource(`/api/projects/${encodeURIComponent(p.name)}/logs?service=${encodeURIComponent(state.logService)}`);
  state.logSource = es;
  es.addEventListener('lines', (ev) => appendOutput(out, JSON.parse(ev.data)));
  es.addEventListener('end', (ev) => {
    const { err } = JSON.parse(ev.data);
    appendOutput(out, [err ? `[log stream failed: ${err}]` : '[log stream ended]']);
    es.close();
  });
  es.addEventListener('error', () => es.close());
}

function stopLogs() {
  if (state.logSource) state.logSource.close();
  state.logSource = null;
}

// appendOutput adds lines, following the end only if the view was there.
function appendOutput(el, lines) {
  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30;
  el.append(document.createTextNode(lines.join('\n') + '\n'));
  // Long streams are trimmed so the page stays responsive.
  if (el.textContent.length > 400000) el.textContent = el.textContent.slice(-200000);
  if (atBottom) el.scrollTop = el.scrollHeight;
}

// ---------- activity ----------

function renderJobs() {
  const running = state.jobs.filter((j) => !j.done).length;
  $('activity-count').textContent = running ? String(running) : '';
  const jobs = [...state.jobs].reverse();
  $('job-list').replaceChildren(...jobs.map((j) => h('li', {
    class: j.id === state.jobId ? 'selected' : '',
    onclick: () => showJob(j.id, false),
    title: j.err || '',
  },
  h('span', { class: 'dot ' + (!j.done ? 'busy' : j.err ? 'bad' : 'running') }),
  h('span', {}, j.title),
  h('span', { class: 'when' }, new Date(j.started).toLocaleTimeString()))));
  const current = state.jobs.find((j) => j.id === state.jobId);
  if (current) $('activity-title').textContent = current.title;
  for (const j of state.jobs) {
    if (j.done && !state.seenJobs.has(j.id)) {
      state.seenJobs.add(j.id);
      if (j.err) toast(`${j.title} failed: ${j.err}`, true);
      if (j.project === '' || j.project === state.selected) loadDetail();
    }
    if (!j.done) state.seenJobs.delete(j.id);
  }
}

function showJob(id, open) {
  state.jobId = id;
  if (open) $('activity').classList.remove('collapsed');
  if (state.jobSource) state.jobSource.close();
  const out = $('job-output');
  out.textContent = '';
  const job = state.jobs.find((j) => j.id === id);
  $('activity-title').textContent = job ? job.title : '';
  const es = new EventSource('/api/jobs/' + id);
  state.jobSource = es;
  es.addEventListener('lines', (ev) => appendOutput(out, JSON.parse(ev.data)));
  es.addEventListener('end', (ev) => {
    const { err } = JSON.parse(ev.data);
    appendOutput(out, [err ? `\n✗ Failed: ${err}` : '\n✓ Done']);
    es.close();
  });
  es.addEventListener('error', () => es.close());
  renderJobs();
}

// ---------- dialogs ----------

function openDialog(title, body, okLabel, danger) {
  const dlg = $('dialog');
  $('dialog-title').textContent = title;
  $('dialog-body').replaceChildren(...[].concat(body));
  const ok = $('dialog-ok');
  ok.textContent = okLabel || 'OK';
  ok.className = 'btn ' + (danger ? 'danger' : 'primary');
  dlg.showModal();
  // Resolved from the form's own events: the dialog's close event comes
  // late or not at all in some embedded browsers.
  return new Promise((resolve) => {
    const finish = (result) => {
      $('dialog-form').onsubmit = null;
      $('dialog-cancel').onclick = null;
      dlg.oncancel = null;
      if (dlg.open) dlg.close();
      resolve(result);
    };
    $('dialog-form').onsubmit = (e) => { e.preventDefault(); finish(true); };
    $('dialog-cancel').onclick = () => finish(false);
    dlg.oncancel = (e) => { e.preventDefault(); finish(false); };
  });
}

function confirmDialog(title, text, okLabel) {
  return openDialog(title, h('p', {}, text), okLabel, true);
}

async function confirmDelete(p) {
  const omit = h('input', { type: 'checkbox', id: 'omit-snapshot' });
  const ok = await openDialog(`Delete ${p.name}?`, [
    h('p', {}, 'This removes the project’s containers and database and unregisters it. The project files are not touched.'),
    h('label', { class: 'switch' }, omit, 'Skip the database snapshot'),
  ], 'Delete', true);
  if (!ok) return;
  await runJob('/api/projects/' + encodeURIComponent(p.name) + '/actions/delete', { omit_snapshot: omit.checked ? 'true' : 'false' });
}

async function addProject() {
  if (!state.types) {
    state.types = await fetch('/api/project-types').then((r) => r.json()).catch(() => []);
  }
  const dir = h('input', { type: 'text', id: 'f-dir', placeholder: '/home/me/projects/my-site', required: true });
  const type = h('select', { id: 'f-type' }, h('option', { value: '' }, 'detect automatically'),
    state.types.map((t) => h('option', { value: t }, t)));
  const name = h('input', { type: 'text', id: 'f-name', placeholder: 'defaults to the directory name' });
  const docroot = h('input', { type: 'text', id: 'f-docroot', placeholder: 'e.g. web or public' });
  const field = (label, input, hint) => h('div', { class: 'field' }, h('label', { for: input.id }, label), input, hint ? h('span', { class: 'hint' }, hint) : null);
  const ok = await openDialog('Add project', [
    field('Project directory', dir, 'An existing directory. This runs “ddev config” there.'),
    field('Project type', type),
    field('Project name', name),
    field('Docroot', docroot),
  ], 'Configure');
  if (!ok) return;
  await runJob('/api/projects', { dir: dir.value, type: type.value, name: name.value, docroot: docroot.value });
}

// ---------- startup ----------

function init() {
  $('filter').addEventListener('input', (e) => { state.filter = e.target.value; renderProjects(); });
  document.addEventListener('keydown', (e) => {
    if (e.key === '/' && document.activeElement.tagName !== 'INPUT' && !$('dialog').open) {
      e.preventDefault();
      $('filter').focus();
    }
  });
  $('activity-toggle').addEventListener('click', () => $('activity').classList.toggle('collapsed'));
  $('btn-add').addEventListener('click', addProject);
  $('btn-start-all').addEventListener('click', () => runJob('/api/actions/start-all'));
  $('btn-stop-all').addEventListener('click', async () => {
    if (await confirmDialog('Stop all projects?', 'Every running DDEV project will be stopped.', 'Stop all')) runJob('/api/actions/stop-all');
  });
  $('btn-poweroff').addEventListener('click', async () => {
    if (await confirmDialog('Power off DDEV?', 'This stops all projects and DDEV’s shared containers.', 'Power off')) runJob('/api/actions/poweroff');
  });
  // A desktop webview opens no new windows, so the server opens the link.
  document.addEventListener('click', (e) => {
    const a = e.target.closest && e.target.closest('a[target="_blank"]');
    if (!state.desktop || !a || !/^https?:/.test(a.href)) return;
    e.preventDefault();
    post('/api/open-url', { url: a.href }).catch((err) => toast(err.message, true));
  });
  connectEvents();
}

init();
