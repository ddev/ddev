// DDEV web UI for `ddev gui`. Plain DOM code, no build step: the page
// runs under a strict Content-Security-Policy, so nothing is inlined and all
// text goes in through textContent.
'use strict';

const state = {
  projects: [],
  jobs: [],
  router: '',
  selected: '',
  filter: '',
  // show is which projects the list shows: all, up, or down.
  show: localStorage.getItem('ddev.show') || 'all',
  detail: null,
  detailFor: '',
  detailStatus: '',
  tab: localStorage.getItem('ddev.tab') || 'overview',
  jobId: 0,
  jobSource: null,
  logSource: null,
  desktop: false,
  error: '',
  // xdebugWanted holds a project's requested Xdebug state until its detail
  // is reloaded, so the switch does not flip back while the job runs.
  xdebugWanted: {},
  logService: 'web',
  registry: null,
  types: null,
  seenJobs: new Set(),
  // shownJobs are the jobs the panel has shown, so a finished one doesn't
  // take it again.
  shownJobs: new Set(),
  // view is home, project, shared, or multi: what the detail area shows.
  view: 'home',
  // messages are DDEV's tips and notices, from /api/messages.
  messages: null,
  // shared holds the router's and SSH agent's status, from /api/shared.
  shared: null,
  sharedLog: 'ddev-router',
  tipIndex: Math.floor(Math.random() * 1000),
  // picked are the projects chosen with Ctrl+click or Shift+click, for an
  // action on several at once; pickFrom is where a Shift+click range starts.
  picked: [],
  pickFrom: '',
  pickReturn: '',
  // news are the latest posts of the DDEV blog, from /api/news.
  news: null,
  // conn is the status stream: connecting, ok, or lost.
  conn: 'connecting',
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

function closeButton(label, onclick) {
  return h('button', { class: 'close-btn i-x', type: 'button', title: label, 'aria-label': label, onclick });
}

function icon(name, cls = '') {
  return h('i', { class: `i i-${name} ${cls}`, 'aria-hidden': 'true' });
}

function toast(text, isError) {
  const t = h('div', { class: 'toast' + (isError ? ' error' : '') }, h('span', {}, text), closeButton('Dismiss', () => t.remove()));
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

// runJob starts a job and shows its output, and returns whether it started.
async function runJob(url, params, openActivity = true) {
  try {
    const job = await post(url, params);
    showJob(job.id, openActivity);
    return true;
  } catch (e) {
    toast(e.message, true);
    return false;
  }
}

function copy(text) {
  navigator.clipboard.writeText(text).then(
    () => toast('Copied: ' + text),
    () => toast('Copy failed', true),
  );
}

function copyButton(text) {
  return h('button', { class: 'copy-btn i-copy', title: 'Copy', 'aria-label': 'Copy', type: 'button', onclick: () => copy(text) });
}

function copyable(text, display, cls = '') {
  return h('span', { class: 'copyable ' + cls }, display || text, copyButton(text));
}

function link(url, text) {
  return h('a', { href: url, target: '_blank', rel: 'noopener noreferrer' }, text || url);
}

const DOCS = 'https://docs.ddev.com/en/stable/';

function docLink(page, label = 'Docs', title) {
  return h('a', { class: 'doc-link', href: DOCS + page, target: '_blank', rel: 'noopener noreferrer', title: title || 'Read more on docs.ddev.com' }, label, icon('ext', 'ext'));
}

// helpLink is a "?" to a docs page about the thing next to it.
function helpLink(page, about) {
  const label = `About ${about} on docs.ddev.com`;
  return h('a', { class: 'help-link', href: DOCS + page, target: '_blank', rel: 'noopener noreferrer', title: label, 'aria-label': label }, '?');
}

// richText shows the remote config's markup: `code`, *emphasis*, and URLs.
function richText(s) {
  const out = [];
  const re = /`([^`]+)`|\*([^*]+)\*|(https?:\/\/[^\s<>"]*[^\s<>".,;:!?)'’])/g;
  let last = 0;
  let m;
  while ((m = re.exec(s))) {
    out.push(s.slice(last, m.index));
    out.push(m[1] ? h('code', {}, m[1]) : m[2] ? h('em', {}, m[2]) : link(m[3]));
    last = re.lastIndex;
  }
  out.push(s.slice(last));
  return out;
}

// Status classes: running, paused/starting (amber), stopped (grey), the
// rest red.
function statusClass(status) {
  if (!status) return '';
  if (status === 'running' || status === 'healthy') return 'running';
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
  es.addEventListener('open', () => { state.conn = 'ok'; renderStatus(); });
  es.addEventListener('error', () => { state.conn = 'lost'; renderStatus(); });
  es.addEventListener('status', (ev) => {
    const st = JSON.parse(ev.data);
    state.projects = st.projects || [];
    state.jobs = st.jobs || [];
    state.router = st.router || '';
    state.sshAgent = st.ssh_agent || '';
    state.desktop = !!st.desktop;
    state.error = st.error || '';
    state.conn = 'ok';
    const banner = $('error-banner');
    banner.textContent = st.error || '';
    banner.classList.toggle('hidden', !st.error);
    renderStatus();
    renderProjects();
    renderJobs();
    if (state.view === 'shared' && st.router + st.ssh_agent !== lastShared) loadShared();
    lastShared = st.router + st.ssh_agent;
    const p = state.view === 'project' ? selectedProject() : null;
    if (p && (p.name !== state.detailFor || p.status !== state.detailStatus)) {
      loadDetail();
    } else {
      renderDetail();
    }
    // A deleted project leaves the start page in its place.
    if (state.view === 'project' && !p && state.selected && state.projects.length && !st.error) goHome();
  });
}

let lastShared = '';

// renderStatus sums up the server connection, Docker, the router and the
// SSH agent in one pill, which names the first problem.
function renderStatus() {
  const services = [['Router', state.router], ['SSH agent', state.sshAgent]];
  const docker = state.error.startsWith('Docker is not reachable') ? 'unreachable' : state.error ? 'error' : 'connected';
  let text = 'Connecting…';
  let cls = '';
  if (state.conn === 'lost') {
    [text, cls] = ['Disconnected, retrying…', 'bad'];
  } else if (state.error) {
    [text, cls] = [docker === 'unreachable' ? 'Docker unreachable' : 'Error', 'bad'];
  } else if (state.conn === 'ok') {
    const active = services.filter(([, v]) => v && v !== 'disabled');
    const problem = active.find(([, v]) => !['healthy', 'stopped'].includes(v));
    // A stopped service is no problem: it starts with the next project.
    const stopped = active.filter(([, v]) => v === 'stopped');
    if (problem) [text, cls] = [`${problem[0]} ${problem[1]}`, statusClass(problem[1]) === 'bad' ? 'bad' : 'warn'];
    else if (!stopped.length) [text, cls] = ['All services running', 'ok'];
    else if (stopped.length === active.length) [text, cls] = ['Services stopped', ''];
    else [text, cls] = [`${stopped[0][0]} stopped`, ''];
  }
  const el = $('status');
  el.className = 'status-button ' + (cls === 'ok' ? '' : cls);
  el.querySelector('.dot').className = 'dot ' + (cls === 'ok' ? 'running' : cls === 'warn' ? 'paused' : cls);
  $('status-text').textContent = text;
  el.title = services.map(([l, v]) => `${l}: ${v || 'unknown'}`).join(' · ') + ` · Docker: ${docker}. Click for the shared services.`;
}

// ---------- project list ----------

function selectedProject() {
  return state.projects.find((p) => p.name === state.selected);
}

function setView(view) {
  if (state.view === 'shared' && view !== 'shared') stopLogs();
  state.view = view;
  if (view !== 'multi') state.picked = [];
  $('nav-home').classList.toggle('selected', view === 'home');
  $('nav-shared').classList.toggle('selected', view === 'shared');
}

function goHome() {
  stopLogs();
  setView('home');
  state.selected = '';
  renderProjects();
  renderDetail();
}

function selectProject(name) {
  const wasProject = state.view === 'project';
  state.pickFrom = name;
  setView('project');
  if (state.selected === name) {
    if (!wasProject) { renderProjects(); renderDetail(); }
    return;
  }
  state.selected = name;
  state.detail = null;
  state.detailFor = '';
  stopLogs();
  renderProjects();
  renderDetail();
  loadDetail();
}

// pick adds or removes a project from the selection, or with range adds the
// listed projects from pickFrom to it. Ctrl+click starts from the project
// shown, as in a file manager; a checkbox (fresh) starts from none. The
// selection stays until it is empty or cleared.
function pick(name, { range = false, fresh = false } = {}) {
  const shown = state.view === 'project' ? state.selected : '';
  if (state.view !== 'multi') state.pickReturn = shown;
  let picked = state.view === 'multi' ? [...state.picked] : shown && !fresh ? [shown] : [];
  if (range && state.pickFrom) {
    const names = shownProjects().map((p) => p.name);
    const [a, b] = [names.indexOf(state.pickFrom), names.indexOf(name)].sort((x, y) => x - y);
    if (a >= 0) picked = [...new Set([...picked, ...names.slice(a, b + 1)])];
  } else if (state.view === 'multi' || name !== shown) {
    picked = picked.includes(name) ? picked.filter((n) => n !== name) : [...picked, name];
    state.pickFrom = name;
  }
  if (!picked.length) {
    clearPicks();
    return;
  }
  setView('multi');
  state.picked = picked;
  state.selected = '';
  stopLogs();
  renderProjects();
  renderDetail();
}

// clearPicks goes back to the project shown before the selection began.
function clearPicks() {
  const back = state.pickReturn;
  state.pickReturn = '';
  if (back && state.projects.some((p) => p.name === back)) selectProject(back); else goHome();
}

// Chips that narrow the list to running or stopped projects, with counts.
function renderChips() {
  const up = state.projects.filter((p) => isUp(p.status)).length;
  const chips = [['all', 'All', state.projects.length], ['up', 'Running', up], ['down', 'Stopped', state.projects.length - up]];
  $('status-chips').replaceChildren(...chips.map(([id, label, n]) => h('button', {
    class: 'chip' + (state.show === id ? ' active' : ''),
    'aria-pressed': state.show === id ? 'true' : 'false',
    onclick: () => { state.show = id; localStorage.setItem('ddev.show', id); renderProjects(); selectShown(); },
  }, label + ' ', h('span', { class: 'chip-count' }, String(n)))));
}

function shownProjects() {
  const f = state.filter.toLowerCase();
  return state.projects.filter((p) =>
    (state.show === 'all' || (state.show === 'up') === isUp(p.status)) &&
    (!f || p.name.toLowerCase().includes(f) || p.type.toLowerCase().includes(f) || p.status.toLowerCase().includes(f)));
}

// selectShown keeps the detail on a listed project after the user narrows
// the list. Status updates do not call it, so a project stopped under the
// Running chip stays selected.
function selectShown() {
  if (state.view !== 'project') return;
  const shown = shownProjects();
  if (shown.length && !shown.some((p) => p.name === state.selected)) selectProject(shown[0].name);
}

function renderProjects() {
  renderChips();
  const list = $('project-list');
  const shown = shownProjects();
  const focused = list.contains(document.activeElement);
  const picking = state.view === 'multi';
  list.classList.toggle('picking', picking);
  list.setAttribute('aria-multiselectable', String(picking));
  list.replaceChildren(...shown.map((p, i) => {
    const busy = runningJobFor(p.name);
    const selected = picking ? state.picked.includes(p.name) : p.name === state.selected && state.view === 'project';
    return h('li', {
      class: selected ? 'selected' : '',
      role: 'option',
      'aria-selected': String(selected),
      // One item takes Tab; the arrow keys move between them.
      tabindex: selected || (i === 0 && state.view !== 'project') ? '0' : '-1',
      'data-name': p.name,
      onclick: (e) => {
        if (e.ctrlKey || e.metaKey || e.shiftKey) {
          e.preventDefault();
          pick(p.name, { range: e.shiftKey });
        } else {
          selectProject(p.name);
        }
      },
      oncontextmenu: (e) => { e.preventDefault(); projectMenu(p, e.clientX, e.clientY); },
    },
    h('span', {
      class: 'lead',
      title: 'Select several projects, also with Ctrl+click or Shift+click',
      onclick: (e) => { e.stopPropagation(); e.preventDefault(); pick(p.name, { range: e.shiftKey, fresh: true }); },
    },
    h('input', { type: 'checkbox', class: 'pick', tabindex: '-1', checked: picking && selected, 'aria-label': `Select ${p.name}` }),
      h('span', { class: 'dot ' + (busy ? 'busy' : statusClass(p.status)) })),
    h('span', { class: 'name' }, p.name),
    h('span', { class: 'type' }, p.type),
    h('span', { class: 'sub' }, busy ? busy.title + '…' : (p.url || p.status)));
  }));
  // A rebuilt list keeps the keyboard focus on its selected item.
  if (focused) (list.querySelector('li.selected') || list.querySelector('li'))?.focus();
  // An error such as Docker being unreachable empties the list; the banner
  // explains it, and "No projects yet" would not be true.
  $('project-empty').classList.toggle('hidden', state.projects.length > 0 || !!state.error);
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
  const running = state.jobs.filter((j) => !j.done).map((j) => [j.id, j.project]);
  const key = JSON.stringify(state.view === 'home' ? ['home', state.messages, state.news, state.tipIndex, state.projects.map((x) => x.status), state.error]
      : state.view === 'shared' ? ['shared', state.shared, state.sharedLog]
        : state.view === 'multi' ? ['multi', state.picked, state.projects, running]
          : [p, state.tab, state.detail, state.xdebugWanted, running]);
  if (key === detailKey && root.childElementCount) return;
  detailKey = key;
  if (state.view !== 'project') {
    // The shared view keeps its log stream while its status refreshes.
    if (state.view === 'shared' && root.querySelector('.logs-output')) {
      root.querySelector('.shared-cards').replaceWith(sharedCards());
      return;
    }
    stopLogs();
    root.dataset.project = '';
    root.replaceChildren(({ shared: sharedView, multi: multiView })[state.view]?.() || homeView());
    return;
  }
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

// quickstarts maps project types to their section of the quickstart page.
const quickstarts = {
  asterios: 'asterios', backdrop: 'backdrop', cakephp: 'cakephp', codeigniter: 'codeigniter', craftcms: 'craft-cms',
  generic: 'generic', joomla: 'joomla', laravel: 'laravel', magento: 'openmage', magento2: 'magento-2', maho: 'maho',
  modx: 'modx-revolution', shopware6: 'shopware', silverstripe: 'silverstripe-cms', symfony: 'symfony', typo3: 'typo3',
  wordpress: 'wordpress', 'wp-bedrock': 'wordpress',
};

function quickstartPage(type) {
  const anchor = type.startsWith('drupal') ? 'drupal' : quickstarts[type];
  return 'users/quickstart/' + (anchor ? '#' + anchor : '');
}

function detailHead(p) {
  const busy = runningJobFor(p.name);
  return h('div', { class: 'detail-head' },
    h('div', {},
      h('h1', {}, p.name,
        h('span', { class: 'pill ' + (p.status === 'running' ? 'ok' : statusClass(p.status) === 'bad' ? 'bad' : '') }, busy ? busy.title + '…' : p.status),
        h('a', {
          class: 'badge type-link', href: DOCS + quickstartPage(p.type), target: '_blank', rel: 'noopener noreferrer',
          title: `The ${p.type} quickstart on docs.ddev.com`,
        }, p.type)),
      h('div', { class: 'sub-line' },
        p.url && isUp(p.status) ? copyable(p.url, link(p.url), 'url') : null,
        h('div', { class: 'path' }, copyable(p.approot)))),
    closeButton('Back to the start page', goHome));
}

function projectAction(p, a, params) {
  return runJob('/api/projects/' + encodeURIComponent(p.name) + '/actions/' + a, params);
}

async function openIn(p, target) {
  try { await post('/api/projects/' + encodeURIComponent(p.name) + '/open/' + target); } catch (e) { toast(e.message, true); }
}

function openURL(url) {
  if (state.desktop) post('/api/open-url', { url }).catch((err) => toast(err.message, true));
  else window.open(url, '_blank', 'noopener,noreferrer');
}

function projectMenu(p, x, y) {
  const busy = !!runningJobFor(p.name);
  const running = isUp(p.status);
  openMenu({ x, y }, [
    running
      ? { label: 'Stop', disabled: busy, run: () => projectAction(p, 'stop') }
      : { label: 'Start', disabled: busy, run: () => projectAction(p, 'start') },
    { label: 'Restart', disabled: busy || !running, run: () => projectAction(p, 'restart') },
    null,
    { label: 'Open site', disabled: !running || !p.url, run: () => openURL(p.url) },
    ...editors.map(([t, label]) => ({ label, run: () => openIn(p, t) })),
    { label: 'Copy path', run: () => copy(p.approot) },
    null,
    { label: 'Delete…', danger: true, disabled: busy, run: () => confirmDelete(p) },
  ]);
}

// editors are where "Open in" opens a project. The toolbar shows the folder
// on its own, and a button for the editor last chosen.
const editors = [['folder', 'Open folder'], ['vscode', 'Open in VS Code'], ['phpstorm', 'Open in PhpStorm']];
const codeEditors = editors.slice(1);

function lastEditor() {
  const t = localStorage.getItem('ddev.openIn');
  return codeEditors.find(([id]) => id === t) || codeEditors[0];
}

// splitButton is a main button with an arrow beside it for a menu of
// related actions.
function splitButton(main, label, items) {
  const caret = h('button', { class: 'btn caret i-chevron', type: 'button', title: label, 'aria-label': label, 'aria-haspopup': 'menu' });
  caret.addEventListener('click', () => (menuAnchor === caret ? menuClose() : openMenu(caret, items())));
  return h('span', { class: 'split' }, main, caret);
}

function toolbar(p) {
  const d = state.detail && !state.detail.error ? state.detail.describe : null;
  const busy = !!runningJobFor(p.name);
  // A project that is starting or unhealthy is up too, and needs Stop.
  const running = isUp(p.status);
  const act = (a, params) => () => projectAction(p, a, params);
  const open = (t) => () => openIn(p, t);
  const mailpit = d && (d.mailpit_https_url || d.mailpit_url);
  const xhgui = d && d.xhgui_status === 'enabled' && (d.xhgui_https_url || d.xhgui_url);
  const [editor, editorLabel] = lastEditor();
  const more = h('button', { class: 'btn icon-btn', type: 'button', title: 'More actions', 'aria-label': 'More actions', 'aria-haspopup': 'menu' }, icon('more'));
  const detail = state.detail && !state.detail.error ? state.detail : null;
  more.addEventListener('click', () => (menuAnchor === more ? menuClose() : openMenu(more, [
    { label: 'Create snapshot', disabled: busy || !(d && d.dbinfo), run: act('snapshot') },
    { label: 'Restore the latest snapshot…', disabled: busy || !(detail && detail.snapshots.length), run: () => restoreLatest(p) },
    { label: 'Update add-ons', disabled: busy || !(detail && detail.addons.length), run: act('addon-update') },
    null,
    { label: 'Copy site URL', disabled: !p.url, run: () => copy(p.url) },
    { label: 'Copy path', run: () => copy(p.approot) },
    { label: 'Copy the command that opens a shell', run: () => copy(`cd ${shellQuote(p.approot)} && ddev ssh`) },
    null,
    { label: 'Delete…', danger: true, disabled: busy, run: () => confirmDelete(p) },
  ])));
  const groups = [
    [running
      ? h('button', { class: 'btn', disabled: busy, onclick: act('stop') }, 'Stop')
      : h('button', { class: 'btn primary', disabled: busy, onclick: act('start') }, 'Start'),
    h('button', { class: 'btn', disabled: busy || !running, onclick: act('restart') }, 'Restart')],
    [p.url && running ? h('a', { class: 'btn', href: p.url, target: '_blank', rel: 'noopener noreferrer' }, 'Open site', icon('ext', 'ext')) : null,
      mailpit && running ? h('a', { class: 'btn', href: mailpit, target: '_blank', rel: 'noopener noreferrer' }, 'Mailpit', icon('ext', 'ext')) : null,
      xhgui && running ? h('a', { class: 'btn', href: xhgui, target: '_blank', rel: 'noopener noreferrer' }, 'XHGui', icon('ext', 'ext')) : null],
    [h('button', { class: 'btn', type: 'button', onclick: open('folder') }, 'Open folder'),
      splitButton(h('button', { class: 'btn', type: 'button', onclick: open(editor) }, editorLabel), 'Open in…', () => codeEditors.map(([t, label]) => ({
        label, run: () => { localStorage.setItem('ddev.openIn', t); openIn(p, t); detailKey = ''; renderDetail(); },
      }))), more],
  ].map((g) => g.filter(Boolean)).filter((g) => g.length);
  return h('div', { class: 'toolbar' }, groups.flatMap((g, i) => (i ? [h('span', { class: 'sep' }), ...g] : g)));
}

function kv(rows) {
  return h('table', { class: 'kv' }, rows.filter((r) => r && r[1] !== undefined && r[1] !== '' && r[1] !== null)
    .map(([k, v]) => h('tr', {}, h('th', {}, k), h('td', {}, v))));
}

function overviewTab(p, d) {
  const desc = d.describe;
  const running = p.status === 'running';
  const busy = !!runningJobFor(p.name);
  const urls = desc.urls || [];
  const shell = `cd ${shellQuote(p.approot)} && ddev ssh`;
  const xdebugOn = d.xdebug === 'on';
  const wanted = state.xdebugWanted[p.name];
  const config = h('div', { class: 'card' }, h('h3', {}, 'Configuration', helpLink('users/configuration/config/', 'the project configuration')), kv([
    ['PHP', desc.php_version],
    ['Web server', desc.webserver_type],
    ['Node.js', desc.nodejs_version],
    ['Docroot', desc.docroot || '(project root)'],
    ['Database', desc.database_type ? `${desc.database_type} ${desc.database_version}` : 'none'],
    ['Performance', desc.performance_mode],
    ['Mutagen', running ? desc.mutagen_status : undefined],
    ['XHProf mode', desc.xhprof_mode],
    ['Router', desc.router_disabled ? 'disabled' : desc.router],
  ]));
  const terminal = h('div', { class: 'card' }, h('h3', {}, 'Terminal', helpLink('users/usage/commands/#ssh', 'ddev ssh')),
    h('p', { class: 'muted' }, 'To open a shell, paste this into a terminal:'),
    copyable(shell));
  const urlCard = running ? h('div', { class: 'card' }, h('h3', {}, 'URLs', helpLink('users/usage/networking/', 'URLs and ports')),
    urls.length
      ? h('ul', { class: 'url-list' }, urls.map((u) => h('li', {}, copyable(u, link(u)))))
      : h('p', { class: 'muted' }, 'No URLs.')) : null;
  const debugging = h('div', { class: 'card' }, h('h3', {}, 'Debugging', helpLink('users/debugging-profiling/step-debugging/', 'step debugging')),
    h('label', { class: 'row-line xdebug-switch' },
      h('span', {}, 'Xdebug ', h('span', { class: 'muted' }, !running ? '(project stopped)'
        : wanted !== undefined ? (wanted ? 'turning on…' : 'turning off…') : (xdebugOn ? 'on' : 'off'))),
      h('input', {
        type: 'checkbox',
        class: 'toggle',
        role: 'switch',
        checked: wanted ?? xdebugOn,
        disabled: !running || busy,
        onchange: async (e) => {
          state.xdebugWanted[p.name] = e.target.checked;
          // The switch's label shows progress, and the Activity panel
          // would cover it. A failure still shows as a toast.
          if (!await runJob('/api/projects/' + encodeURIComponent(p.name) + '/actions/xdebug-' + (e.target.checked ? 'on' : 'off'), undefined, false)) {
            // The panel may not rebuild, since its key can match the last one.
            delete state.xdebugWanted[p.name];
            e.target.checked = !e.target.checked;
            renderDetail();
          }
        },
      })));
  // A stopped project has no URLs; this says why, where they would be.
  const stopped = p.status === 'stopped' ? h('div', { class: 'callout' },
    h('div', {}, h('strong', {}, `${p.name} is stopped.`), h('p', {}, 'Starting it builds its containers and gives it URLs.')),
    h('button', { class: 'btn primary', type: 'button', disabled: busy, onclick: () => projectAction(p, 'start') }, icon('play', 'lead'), 'Start')) : null;
  return h('div', {}, stopped,
    h('div', { class: 'columns' }, h('div', {}, config, terminal), h('div', {}, urlCard, debugging)));
}

function shellQuote(s) {
  return /^[\w@%+=:,./~-]+$/.test(s) ? s : "'" + s.replaceAll("'", "'\\''") + "'";
}

function servicesTab(p, d) {
  const services = Object.entries(d.describe.services || {});
  services.sort(([a], [b]) => (a === 'web' ? -1 : b === 'web' ? 1 : a === 'db' ? -1 : b === 'db' ? 1 : a.localeCompare(b)));
  if (!services.length) return h('p', { class: 'muted' }, 'No services.');
  return h('div', {}, h('p', { class: 'muted tab-intro' }, 'The containers of this project. ', helpLink('users/usage/architecture/', 'the containers')), h('table', { class: 'grid' },
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
    })));
}

function databaseTab(p, d) {
  const db = d.describe.dbinfo;
  if (!db) return h('p', { class: 'muted' }, 'This project has no database container.');
  const host = (d.describe.services?.web?.host_https_url || d.describe.services?.web?.host_http_url || '').replace(/^https?:\/\/([^:/]+).*$/, '$1') || '127.0.0.1';
  const cli = (db.database_type || d.describe.database_type) === 'postgres' ? 'ddev psql' : 'ddev mysql';
  return h('div', { class: 'cards' },
    h('div', { class: 'card' }, h('h3', {}, 'Connection', helpLink('users/usage/database-management/', 'database management')), kv([
      ['Type', `${db.database_type || d.describe.database_type} ${db.database_version || d.describe.database_version}`],
      ['Database', db.dbname && copyable(db.dbname)],
      ['User', db.username && copyable(db.username)],
      ['Password', db.password && copyable(db.password)],
      ['From this project', copyable(`db:${db.dbPort}`)],
      ['From other projects', copyable(`ddev-${p.name}-db:${db.dbPort}`)],
      ['From the host', db.published_port > 0 ? copyable(`${host}:${db.published_port}`) : (p.status === 'running' ? 'not published' : 'start the project')],
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
  const input = h('input', { type: 'text', placeholder: 'ddev/ddev-redis', 'aria-label': 'Add-on repository', autocomplete: 'off', role: 'combobox', 'aria-autocomplete': 'list' });
  // WebKitGTK draws a <datalist> in the GTK theme, so the page draws the
  // suggestions, as it does for dropdowns.
  const suggest = async () => {
    const q = input.value.trim().toLowerCase();
    const addons = (await loadRegistry()).filter((a) => !q || a.repo.toLowerCase().includes(q) || (a.title || '').toLowerCase().includes(q));
    if (document.activeElement !== input) return;
    if (!addons.length) { if (menuAnchor === input) menuClose(); return; }
    openMenu(input, addons.map((a) => ({
      label: a.repo,
      run: () => { input.value = a.repo; input.focus(); },
    })), { keepFocus: true });
  };
  input.addEventListener('input', suggest);
  input.addEventListener('focus', suggest);
  return h('div', {},
    h('form', {
      class: 'row',
      onsubmit: (e) => {
        e.preventDefault();
        if (input.value.trim()) act('addon-get', { repo: input.value.trim() });
      },
    }, input,
    h('button', { class: 'btn primary', disabled: busy }, 'Install'),
    h('button', { class: 'btn', type: 'button', disabled: busy || !d.addons.length, onclick: () => act('addon-update') }, 'Update all'),
    h('a', { href: 'https://addons.ddev.com', target: '_blank', rel: 'noopener noreferrer' }, 'Browse add-ons', icon('ext', 'ext')),
    helpLink('users/extend/using-add-ons/', 'add-ons')),
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
      h('button', { class: 'btn', disabled: busy || !d.snapshots.length, onclick: () => restore(null) }, 'Restore latest'),
      helpLink('users/usage/database-management/#snapshots', 'snapshots')),
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
  const select = dropdown(services.map((s) => [s, s]), state.logService,
    (v) => { state.logService = v; startLogs(p, out); }, { 'aria-label': 'Service' });
  const view = h('div', {},
    h('div', { class: 'row' }, 'Service', select,
      h('button', { class: 'btn', onclick: () => startLogs(p, out) }, 'Reconnect'),
      h('button', { class: 'btn', onclick: () => { out.textContent = ''; } }, 'Clear'),
      helpLink('users/usage/commands/#logs', 'ddev logs')),
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
// appendOutput adds lines to el, before the node `before` when given, which
// holds a line still being written.
function appendOutput(el, lines, before = null) {
  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30;
  el.insertBefore(document.createTextNode(lines.join('\n') + '\n'), before);
  // Long streams are trimmed so the page stays responsive.
  if (el.textContent.length > 400000) {
    el.textContent = el.textContent.slice(-200000);
    if (before) el.append(before);
  }
  if (atBottom) el.scrollTop = el.scrollHeight;
}

// ---------- start page ----------

async function loadMessages() {
  try {
    const res = await fetch('/api/messages');
    if (!res.ok) throw new Error((await res.text()).trim());
    state.messages = await res.json();
  } catch {
    state.messages = { tips: [], infos: [], warnings: [] };
  }
  renderNotices();
  renderDetail();
}

function dismissedNotices() {
  try { return new Set(JSON.parse(localStorage.getItem('ddev.dismissed') || '[]')); } catch { return new Set(); }
}

// renderNotices shows DDEV's info and warning notices until dismissed.
function renderNotices() {
  const m = state.messages || {};
  const dismissed = dismissedNotices();
  const notices = [...(m.warnings || []).map((n) => ['warn', n]), ...(m.infos || []).map((n) => ['info', n])]
    .filter(([, n]) => !dismissed.has(n.message));
  $('notices').replaceChildren(...notices.map(([kind, n]) => h('div', { class: 'notice ' + kind, role: kind === 'warn' ? 'alert' : 'status' },
    h('div', {}, h('strong', {}, n.title || (kind === 'warn' ? 'Important warning' : 'Important message')), ' ', richText(n.message)),
    closeButton('Dismiss', () => {
      dismissed.add(n.message);
      localStorage.setItem('ddev.dismissed', JSON.stringify([...dismissed]));
      renderNotices();
    }))));
}

function greeting() {
  const hour = new Date().getHours();
  if (hour < 5) return 'Working late?';
  return hour < 12 ? 'Good morning' : hour < 18 ? 'Good afternoon' : 'Good evening';
}

function homeView() {
  const up = state.projects.filter((p) => isUp(p.status)).length;
  const tips = state.messages ? state.messages.tips : [];
  const n = tips.length;
  const i = n ? ((state.tipIndex % n) + n) % n : 0;
  const tip = n ? tips[i] : null;
  const step = (d) => () => { state.tipIndex = i + d; renderDetail(); };
  const count = state.projects.length;
  const posts = state.news || [];
  const arrow = (label, d, cls) => h('button', { class: 'btn small icon-btn', type: 'button', title: label, 'aria-label': label, onclick: step(d) }, icon('chevron', cls));
  const ext = (href, label) => h('a', { class: 'doc-link', href, target: '_blank', rel: 'noopener noreferrer' }, label, icon('ext', 'ext'));
  return h('div', { class: 'home' },
    h('div', { class: 'detail-head' }, h('div', {},
      h('h1', {}, greeting()),
      h('p', { class: 'muted' }, count
        ? `${count} ${count === 1 ? 'project' : 'projects'}, ${up} running. Select a project, or right-click it for its actions.`
        : state.error ? '' : 'Add a project to get started.'))),
    count || state.error ? null : h('p', {}, h('button', { class: 'btn primary', type: 'button', onclick: addProject }, icon('plus', 'lead'), 'Add project')),
    h('div', { class: 'card tip' },
      h('div', { class: 'card-head' }, h('h3', {}, tip && tip.title ? tip.title : 'Tip of the day'),
        n > 1 ? h('div', { class: 'tip-nav' },
          arrow('Previous tip', -1, 'left'),
          h('span', { class: 'muted' }, `${i + 1} of ${n}`),
          arrow('Next tip', 1, 'right')) : null),
      tip ? h('p', { class: 'tip-text' }, richText(tip.message))
        : h('p', { class: 'muted' }, state.messages ? 'No tips right now.' : 'Loading…')),
    posts.length ? h('div', { class: 'card' },
      h('div', { class: 'card-head' }, h('h3', {}, 'From the DDEV blog'), ext('https://ddev.com/blog/', 'All posts')),
      h('ul', { class: 'news' }, posts.map((post) => h('li', {},
        link(post.link, post.title),
        h('span', { class: 'meta' }, h('span', { class: 'muted' }, post.date), post.category ? h('span', { class: 'badge' }, post.category) : null))))) : null,
    h('p', { class: 'home-links muted' },
      docLink('users/quickstart/', 'Quickstarts'), docLink('users/usage/faq/', 'FAQ'),
      docLink('users/usage/troubleshooting/', 'Troubleshooting'), ext('https://ddev.com/s/discord', 'Discord')));
}

async function loadNews() {
  try {
    const res = await fetch('/api/news');
    state.news = res.ok ? (await res.json()).posts : [];
  } catch {
    state.news = [];
  }
  renderDetail();
}

// ---------- several projects ----------

// multiView acts on the picked projects at once, as `ddev start a b`.
function multiView() {
  const ps = state.picked.map((n) => state.projects.find((p) => p.name === n)).filter(Boolean);
  const up = ps.filter((p) => isUp(p.status));
  const down = ps.filter((p) => !isUp(p.status));
  const busy = state.jobs.some((j) => !j.done && (j.project === '' || state.picked.includes(j.project)));
  const run = (action, list) => runJob('/api/actions/' + action, list.map((p) => ['project', p.name]));
  return h('div', {},
    h('div', { class: 'detail-head' },
      h('div', {},
        h('h1', {}, `${ps.length} ${ps.length === 1 ? 'project' : 'projects'} selected`),
        h('p', { class: 'muted' }, 'Ctrl+click or Shift+click in the list to change the selection. Esc clears it.')),
      closeButton('Clear the selection', clearPicks)),
    h('div', { class: 'toolbar' },
      h('button', { class: 'btn primary', type: 'button', disabled: busy || !down.length, onclick: () => run('start', down) }, icon('play', 'lead'), `Start ${down.length}`),
      h('button', { class: 'btn', type: 'button', disabled: busy || !up.length, onclick: () => run('stop', up) }, icon('stop', 'lead'), `Stop ${up.length}`),
      h('button', { class: 'btn', type: 'button', disabled: busy || !up.length, onclick: () => run('restart', up) }, `Restart ${up.length}`),
      h('span', { class: 'sep' }),
      h('button', { class: 'btn', type: 'button', onclick: clearPicks }, 'Clear selection')),
    h('table', { class: 'grid' },
      h('tr', {}, h('th', {}, 'Project'), h('th', {}, 'Status'), h('th', {}, 'Type'), h('th', {}, 'URL')),
      ps.map((p) => h('tr', {},
        h('td', {}, h('button', { class: 'linkish', type: 'button', onclick: () => selectProject(p.name) }, p.name)),
        h('td', {}, h('span', { class: 'dot ' + statusClass(p.status) }), ' ', p.status),
        h('td', {}, p.type),
        h('td', {}, p.url && isUp(p.status) ? link(p.url) : '')))));
}

// ---------- shared services ----------

function showShared() {
  setView('shared');
  state.selected = '';
  renderProjects();
  renderDetail();
  loadShared();
}

async function loadShared() {
  try {
    const res = await fetch('/api/shared');
    if (!res.ok) throw new Error((await res.text()).trim());
    state.shared = await res.json();
  } catch (e) {
    state.shared = { error: e.message };
  }
  renderDetail();
}

function sharedCards() {
  const st = state.shared;
  if (!st) return h('div', { class: 'shared-cards' }, h('p', { class: 'muted' }, 'Loading…'));
  if (st.error) return h('div', { class: 'shared-cards' }, h('p', { class: 'muted' }, st.error));
  return h('div', { class: 'cards shared-cards' }, st.map((svc) => h('div', { class: 'card' },
    h('h3', {}, svc.label, helpLink(svc.docs, `the ${svc.label.toLowerCase()}`)),
    h('p', {}, h('span', { class: 'dot ' + statusClass(svc.status) }), ' ', svc.status,
      svc.image ? h('span', { class: 'muted' }, ' · ' + svc.image) : null),
    svc.dashboard ? h('p', {}, link(svc.dashboard, ['Open the Traefik dashboard', icon('ext', 'ext')])) : null,
    svc.status === 'stopped' ? h('p', { class: 'muted' }, svc.name === 'ddev-router'
      ? 'It starts with the first project that uses it.'
      : 'It starts with the first project, unless global settings leave it out.') : null,
    svc.health && svc.status !== 'healthy' ? h('pre', { class: 'output small-output' }, svc.health) : null,
    svc.errors ? [h('p', { class: 'warn-text' }, 'Router configuration problems:'), h('pre', { class: 'output small-output' }, svc.errors)] : null)));
}

function sharedView() {
  const head = h('div', { class: 'detail-head' },
    h('div', {},
      h('h1', {}, 'Shared services'),
      h('p', { class: 'muted' }, 'Containers that all projects use. “ddev poweroff” stops them.')),
    closeButton('Back to the start page', goHome));
  const st = Array.isArray(state.shared) ? state.shared : [];
  const out = h('pre', { class: 'output logs-output' });
  const running = st.filter((svc) => svc.status !== 'stopped').map((svc) => [svc.name, svc.label]);
  if (!running.some(([n]) => n === state.sharedLog) && running.length) state.sharedLog = running[0][0];
  const view = h('div', {}, head, sharedCards());
  if (running.length) {
    const pick = dropdown(running, state.sharedLog, (v) => { state.sharedLog = v; startSharedLogs(out); }, { 'aria-label': 'Logs of' });
    view.append(h('h2', { class: 'section-title' }, 'Logs'),
      h('div', { class: 'row' }, pick,
        h('button', { class: 'btn', type: 'button', onclick: () => startSharedLogs(out) }, 'Reconnect'),
        h('button', { class: 'btn', type: 'button', onclick: () => { out.textContent = ''; } }, 'Clear')),
      out);
    startSharedLogs(out);
  }
  return view;
}

function startSharedLogs(out) {
  stopLogs();
  out.textContent = '';
  const es = new EventSource(`/api/shared/${encodeURIComponent(state.sharedLog)}/logs`);
  state.logSource = es;
  es.addEventListener('lines', (ev) => appendOutput(out, JSON.parse(ev.data)));
  es.addEventListener('end', (ev) => {
    const { err } = JSON.parse(ev.data);
    appendOutput(out, [err ? `[log stream failed: ${err}]` : '[log stream ended]']);
    es.close();
  });
  es.addEventListener('error', () => es.close());
}

// ---------- activity ----------

function setActivityOpen(open) {
  $('activity').classList.toggle('collapsed', !open);
  $('activity-toggle').setAttribute('aria-expanded', String(open));
}

function renderJobs() {
  const running = state.jobs.filter((j) => !j.done).length;
  $('activity-count').textContent = running ? String(running) : '';
  $('activity-clear').disabled = !state.jobs.some((j) => j.done);
  // A new job, even one started from the tray or another window, takes the
  // panel unless the job shown is still running.
  const current = state.jobs.find((j) => j.id === state.jobId);
  const latest = state.jobs.at(-1);
  if (latest && latest.id > state.jobId && (!current || current.done) && !state.seenJobs.has(latest.id) && !state.shownJobs.has(latest.id)) {
    showJob(latest.id, false);
    return;
  }
  const jobs = [...state.jobs].reverse();
  if (!jobs.length) {
    $('job-list').replaceChildren(h('li', { class: 'empty' }, 'Actions you run, such as Start, show their output here.'));
  } else $('job-list').replaceChildren(...jobs.map((j) => h('li', {
    class: j.id === state.jobId ? 'selected' : '',
    onclick: () => showJob(j.id, false),
    title: j.err || '',
  },
  h('span', { class: 'dot ' + (!j.done ? 'busy' : j.err ? 'bad' : 'running') }),
  h('span', { class: 'job-name' }, j.title),
  h('span', { class: 'when' }, new Date(j.started).toLocaleTimeString()),
  // A running action has nothing to remove yet, but keeps the space.
  h('button', {
    class: 'close-btn small i-x',
    'aria-label': `Remove ${j.title} from the list`,
    title: 'Remove from the list',
    disabled: !j.done,
    onclick: (e) => {
      e.stopPropagation();
      removeJob(j.id);
    },
  }))));
  if (current) renderJobTitle(current);
  for (const j of state.jobs) {
    if (j.done && !state.seenJobs.has(j.id)) {
      state.seenJobs.add(j.id);
      if (j.err) toast(`${j.title} failed: ${j.err}`, true);
      if (j.project === '' || j.project === state.selected) loadDetail();
    }
    if (!j.done) state.seenJobs.delete(j.id);
  }
}

async function removeJob(id) {
  try {
    await post(`/api/jobs/${id}/remove`);
    if (state.jobId === id) clearJobView();
  } catch (err) {
    toast(err.message, true);
  }
}

function renderJobTitle(j) {
  const [label, cls] = !j.done ? ['running', 'warn'] : j.err ? ['failed', 'bad'] : ['done', 'ok'];
  $('activity-title').replaceChildren(h('span', { class: 'job-title' }, j.title), ' ', h('span', { class: 'pill ' + cls }, label));
}

function clearJobView() {
  if (state.jobSource) state.jobSource.close();
  state.jobId = 0;
  $('activity-title').replaceChildren();
  $('job-output').textContent = '';
}

function showJob(id, open) {
  state.jobId = id;
  state.shownJobs.add(id);
  if (open) setActivityOpen(true);
  if (state.jobSource) state.jobSource.close();
  const out = $('job-output');
  out.textContent = '';
  const job = state.jobs.find((j) => j.id === id);
  if (job) renderJobTitle(job); else $('activity-title').replaceChildren();
  const es = new EventSource('/api/jobs/' + id);
  state.jobSource = es;
  // ddev writes some lines a piece at a time, such as the dots while it waits.
  const partial = document.createTextNode('');
  out.append(partial);
  es.addEventListener('lines', (ev) => {
    partial.data = '';
    appendOutput(out, JSON.parse(ev.data), partial);
  });
  es.addEventListener('partial', (ev) => {
    const atBottom = out.scrollHeight - out.scrollTop - out.clientHeight < 30;
    partial.data = JSON.parse(ev.data);
    if (atBottom) out.scrollTop = out.scrollHeight;
  });
  es.addEventListener('end', (ev) => {
    const { err } = JSON.parse(ev.data);
    partial.data = '';
    appendOutput(out, [err ? `\n✗ Failed: ${err}` : '\n✓ Done'], partial);
    es.close();
  });
  es.addEventListener('error', () => es.close());
  renderJobs();
}

// isUp is whether a project's containers run, healthy or not.
function isUp(status) {
  return status === 'running' || status === 'unhealthy' || status.startsWith('starting');
}

// ---------- menus ----------

let menuClose = null;
// menuAnchor is the element whose menu is open, which closes it again.
let menuAnchor = null;

// openMenu shows items below an anchor element, or at a point {x, y}. An
// item is {label, run, disabled, danger, checked}; null is a separator.
// With keepFocus the anchor, such as a text field, keeps the focus until
// the arrow keys move into the menu.
function openMenu(at, items, { keepFocus } = {}) {
  if (menuClose) menuClose();
  const opener = document.activeElement;
  const radio = items.some((it) => it && it.checked !== undefined);
  const menu = h('div', { class: 'menu', role: radio ? 'listbox' : 'menu' }, items.map((it) => (it === null
    ? h('div', { class: 'menu-sep', role: 'separator' })
    : h('button', {
      type: 'button',
      class: 'menu-item' + (it.danger ? ' danger' : '') + (it.checked ? ' checked' : ''),
      role: radio ? 'option' : 'menuitem',
      'aria-selected': radio ? String(!!it.checked) : null,
      disabled: it.disabled,
      onclick: () => { close(); it.run(); },
    }, it.label))));
  // A modal dialog is drawn above everything outside it.
  ((at instanceof Element && at.closest('dialog')) || document.body).append(menu);
  const box = at instanceof Element ? at.getBoundingClientRect() : null;
  if (box) menu.style.minWidth = box.width + 'px';
  const x = box ? box.left : at.x;
  const y = box ? box.bottom + 2 : at.y;
  const hgt = menu.offsetHeight;
  menu.style.left = Math.max(4, Math.min(x, innerWidth - menu.offsetWidth - 4)) + 'px';
  // Below, unless it doesn't fit there and there's more room above.
  const top = box ? box.top - 2 : y;
  const below = innerHeight - y - 4;
  const above = top - 4;
  const up = hgt > below && above > below;
  menu.style.maxHeight = Math.min(up ? above : below, 420) + 'px';
  menu.style.top = (up ? top - Math.min(hgt, above) : y) + 'px';
  const buttons = [...menu.querySelectorAll('button:not(:disabled)')];
  if (!keepFocus) (menu.querySelector('.checked:not(:disabled)') || buttons[0])?.focus();
  const onKey = (e) => {
    if (e.key === 'Escape') { e.preventDefault(); close(); opener?.focus(); return; }
    if (e.key === 'Tab' || (keepFocus && e.key === 'Enter' && document.activeElement === opener)) { close(); return; }
    const step = { ArrowDown: 1, ArrowUp: -1 }[e.key];
    if (!step) return;
    e.preventDefault();
    const i = buttons.indexOf(document.activeElement);
    buttons[(i + step + buttons.length) % buttons.length]?.focus();
  };
  const onDown = (e) => { if (!menu.contains(e.target) && !(menuAnchor && menuAnchor.contains(e.target))) close(); };
  function close() {
    menu.remove();
    document.removeEventListener('keydown', onKey, true);
    document.removeEventListener('mousedown', onDown, true);
    removeEventListener('blur', close);
    removeEventListener('resize', close);
    menuClose = null;
    menuAnchor = null;
  }
  document.addEventListener('keydown', onKey, true);
  document.addEventListener('mousedown', onDown, true);
  addEventListener('blur', close);
  addEventListener('resize', close);
  menuClose = close;
  menuAnchor = at instanceof Element ? at : null;
}

// dropdown replaces <select>, whose list WebKitGTK draws in the GTK theme
// instead of the page's. options are [value, label] pairs; the button's
// value is the chosen one.
function dropdown(options, value, onchange, attrs) {
  const label = h('span');
  const btn = h('button', { type: 'button', class: 'dropdown i-chevron', 'aria-haspopup': 'listbox', ...attrs }, label);
  const set = (v) => {
    btn.value = v;
    label.textContent = (options.find(([o]) => o === v) || options[0] || ['', ''])[1];
  };
  set(value);
  btn.addEventListener('click', () => (menuAnchor === btn ? menuClose() : openMenu(btn, options.map(([v, l]) => ({
    label: l,
    checked: v === btn.value,
    run: () => { set(v); btn.focus(); if (onchange) onchange(v); },
  })))));
  btn.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); btn.click(); }
  });
  return btn;
}

// ---------- dialogs ----------

// openDialog shows a modal dialog and resolves to whether OK was chosen.
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
  const dir = h('input', { type: 'text', id: 'f-dir', placeholder: 'Full path to the project directory', required: true });
  const type = dropdown([['', 'detect automatically'], ...state.types.map((t) => [t, t])], '', null, { id: 'f-type', 'aria-labelledby': 'f-type-label' });
  const name = h('input', { type: 'text', id: 'f-name', placeholder: 'defaults to the directory name' });
  const docroot = h('input', { type: 'text', id: 'f-docroot', placeholder: 'e.g. web or public' });
  const field = (label, input, hint) => h('div', { class: 'field' },
    h('label', input.classList.contains('dropdown') ? { id: input.id + '-label' } : { for: input.id }, label),
    input, hint ? h('span', { class: 'hint' }, hint) : null);
  const ok = await openDialog('Add project', [
    field('Project directory', dir, 'An existing directory. This runs “ddev config” there.'),
    field('Project type', type),
    field('Project name', name),
    field('Docroot', docroot),
  ], 'Configure');
  if (!ok) return;
  await runJob('/api/projects', { dir: dir.value, type: type.value, name: name.value, docroot: docroot.value });
}

// ---------- keyboard ----------

// sidebarItems are what the arrow keys move between: the start page, the
// shared services, and the listed projects.
function sidebarItems() {
  return [
    { el: $('nav-home'), go: goHome, current: state.view === 'home' },
    { el: $('nav-shared'), go: showShared, current: state.view === 'shared' },
    ...shownProjects().map((p) => ({
      el: () => $('project-list').querySelector(`li[data-name="${CSS.escape(p.name)}"]`),
      go: () => selectProject(p.name),
      current: state.view === 'project' && state.selected === p.name,
      project: p,
    })),
  ];
}

function focusSidebarItem() {
  const item = sidebarItems().find((it) => it.current);
  const el = item && (typeof item.el === 'function' ? item.el() : item.el);
  if (el) el.focus();
}

function restoreLatest(p) {
  confirmDialog('Restore the latest snapshot?', `This replaces the current database of ${p.name}.`, 'Restore')
    .then((ok) => ok && projectAction(p, 'restore', { snapshot: '' }));
}

function refresh() {
  loadDetail();
  if (state.view === 'shared') loadShared();
}

function onKeyDown(e) {
  if (e.defaultPrevented || menuClose || $('dialog').open) return;
  const el = document.activeElement;
  const typing = el && el.closest && el.closest('input, textarea, [contenteditable]');
  if (typing || e.ctrlKey || e.metaKey || e.altKey) return;
  const inSidebar = el === document.body || (el && el.closest && el.closest('.sidebar'));
  if (e.key === '/') {
    e.preventDefault();
    $('filter').focus();
  } else if (e.key === 'Escape' && ['shared', 'multi'].includes(state.view)) {
    e.preventDefault();
    (state.view === 'shared' ? goHome : clearPicks)();
  } else if (inSidebar && ['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(e.key)) {
    e.preventDefault();
    const items = sidebarItems();
    const i = items.findIndex((it) => it.current);
    const next = { ArrowDown: Math.min(i + 1, items.length - 1), ArrowUp: Math.max(i - 1, 0), Home: 0, End: items.length - 1 }[e.key];
    if (next !== i && items[next]) items[next].go();
    focusSidebarItem();
  } else if (inSidebar && (e.key === 'ContextMenu' || (e.shiftKey && e.key === 'F10')) && state.view === 'project') {
    const p = selectedProject();
    const li = p && $('project-list').querySelector('li.selected');
    if (!li) return;
    e.preventDefault();
    const box = li.getBoundingClientRect();
    projectMenu(p, box.left + 24, box.bottom);
  }
}

// ---------- startup ----------

function init() {
  $('filter').addEventListener('input', (e) => { state.filter = e.target.value; renderProjects(); selectShown(); });
  document.addEventListener('keydown', onKeyDown);
  $('filter').addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      e.preventDefault();
      e.target.value = '';
      state.filter = '';
      renderProjects();
      e.target.blur();
    } else if (e.key === 'ArrowDown' || e.key === 'Enter') {
      e.preventDefault();
      const first = shownProjects()[0];
      if (first) { selectProject(first.name); focusSidebarItem(); }
    }
  });
  // The whole header opens and closes the panel, except its other buttons.
  $('activity-head').addEventListener('click', (e) => {
    const b = e.target.closest('button');
    if (b && b.id !== 'activity-toggle') return;
    setActivityOpen($('activity').classList.contains('collapsed'));
  });
  $('activity-clear').addEventListener('click', async () => {
    try {
      const shown = state.jobs.find((j) => j.id === state.jobId);
      await post('/api/jobs/clear');
      if (!shown || shown.done) clearJobView();
    } catch (err) {
      toast(err.message, true);
    }
  });
  $('brand').addEventListener('click', goHome);
  $('nav-home').addEventListener('click', goHome);
  $('nav-shared').addEventListener('click', showShared);
  $('status').addEventListener('click', () => {
    // The pill names a failing SSH agent; its logs say why.
    if (state.sshAgent && !['healthy', 'stopped', 'disabled'].includes(state.sshAgent) && state.router === 'healthy') state.sharedLog = 'ddev-ssh-agent';
    showShared();
  });
  $('btn-add').addEventListener('click', addProject);
  const more = $('btn-more');
  more.addEventListener('click', () => (menuAnchor === more ? menuClose() : openMenu(more, [
    { label: 'Refresh', run: refresh },
    { label: 'DDEV documentation', run: () => openURL(DOCS) },
  ])));
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
    openURL(a.href);
  });
  // A browser shows where a link goes; the app's window doesn't, so the page
  // does, at the bottom left.
  const preview = h('div', { id: 'link-preview', class: 'link-preview', hidden: true });
  document.body.append(preview);
  document.addEventListener('mouseover', (e) => {
    const a = state.desktop && e.target.closest && e.target.closest('a[href^="http"]');
    preview.hidden = !a;
    if (a) preview.textContent = a.href;
  });
  // The app keeps WebKit's own menu for text, where it can copy and paste.
  // Elsewhere it would offer Back and Reload.
  document.addEventListener('contextmenu', (e) => {
    if (!state.desktop || e.defaultPrevented) return;
    const editable = e.target.closest && e.target.closest('input, textarea, [contenteditable]');
    if (!editable && String(getSelection()).trim() === '') e.preventDefault();
  });
  // A link such as #project=d11 selects that project; the GUI app's
  // tray uses it.
  const fromHash = () => {
    const m = /(?:^#|&)project=([^&]+)/.exec(location.hash);
    if (!m) return;
    const name = decodeURIComponent(m[1].replace(/\+/g, ' '));
    // Clear the hash, so that the same link selects the project again later.
    history.replaceState(null, '', location.pathname + location.search);
    if (state.show !== 'all') { state.show = 'all'; localStorage.setItem('ddev.show', 'all'); }
    selectProject(name);
    renderProjects();
  };
  window.addEventListener('hashchange', fromHash);
  fromHash();
  connectEvents();
  loadMessages();
  loadNews();
  renderJobs();
}

init();
