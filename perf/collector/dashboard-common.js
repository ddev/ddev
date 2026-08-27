// Shared chart engine for the two dashboard pages (perf-dashboard.html,
// ci-dashboard.html). Each page calls initDashboard(config) once with its
// own dataset config (file, legKey, metrics, etc.) -- there is no dataset
// switcher here. The two datasets (nightly perf benchmarks, CI test
// runtime) answer unrelated questions for unrelated audiences (the perf
// numbers are plausibly public-interesting; CI build times never are), so
// they're two pages, not two modes of one page -- see HANDOFF.md.
function initDashboard(config) {
  const COLORS = ['#4c78a8', '#f58518', '#54a24b', '#e45756', '#72b7b2', '#ff9da6', '#9d755d', '#bab0ac', '#b279a2', '#eeca3b'];
  // Median of the last N nightly runs, not the single latest one -- smooths
  // over a one-off noisy run (a slow VM boot, a network hiccup) while still
  // reflecting how an environment is doing right now. Different legs run on
  // different machines with different raw capability, so this is a
  // comparison of relative overhead, not a hardware benchmark.
  const COMPARE_TRAILING_N = 7;

  // Two independent selection facets, not one flat "leg" checklist -- a
  // pipeline × runner cross-product (20+ workflows times several machines
  // each, on the CI test-runtime page) is unreadable as a single list.
  // workflows/runners are the distinct values available to check;
  // enabledWorkflows/enabledRunners are which ones currently are. Unused by
  // the perf-benchmarks page (its config.runnerOf always returns null), but
  // harmless to keep wired for both -- state.runners just stays empty there
  // and the runner box stays hidden.
  const state = {
    rows: [], metric: null, view: 'trend',
    workflows: [], enabledWorkflows: new Set(),
    runners: [], enabledRunners: new Set(),
    // excludeCancelled's name predates config.isFullSuccessfulRun (it grew
    // from "drop cancelled runs" to "only successful full runs" -- see that
    // function's per-page definition); kept as-is since it's internal, not
    // user-facing.
    excludeCancelled: true,
  };

  function legKey(row) {
    return config.legKey(row);
  }

  // True once the runner box has been narrowed below "every runner" -- the
  // signal rowSelected() uses to drop rows with no runner concept (e.g.
  // every GitHub Actions row on the CI test-runtime page) once the runner
  // box is actually being used to answer a runner-centric question.
  function runnerSelectionNarrowed() {
    return state.runners.length > 0 && state.enabledRunners.size < state.runners.length;
  }

  // Whether a row counts at all, independent of how a counted row then gets
  // grouped into a chart line (that's legKey's job). A row with a runner
  // facet is excluded if its machine is unchecked; a row with no runner
  // facet (config.runnerOf returns null) is included only while the runner
  // box isn't actually narrowing anything -- see runnerSelectionNarrowed().
  function rowSelected(row) {
    if (!state.enabledWorkflows.has(config.workflowKey(row))) return false;
    const runner = config.runnerOf(row);
    if (runner !== null) return state.enabledRunners.has(runner);
    return !runnerSelectionNarrowed();
  }

  function median(values) {
    const sorted = [...values].sort((a, b) => a - b);
    const n = sorted.length;
    if (n === 0) return null;
    return n % 2 === 1 ? sorted[(n - 1) / 2] : (sorted[n / 2 - 1] + sorted[n / 2]) / 2;
  }

  async function loadData() {
    const res = await fetch(config.file, { cache: 'no-store' });
    if (!res.ok) return [];
    const text = await res.text();
    return text
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean)
      .map((l) => {
        try { return JSON.parse(l); } catch (e) { return null; }
      })
      .filter(Boolean)
      .sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
  }

  // Builds one checkbox box (workflow or runner) generically -- the two
  // boxes are structurally identical (a set of available values, a set of
  // currently-enabled ones, All/None buttons), just over different facets.
  function buildCheckboxBox(containerId, allBtnId, noneBtnId, getValues, enabledSet, onChange) {
    const container = document.getElementById(containerId);
    const values = getValues();
    container.innerHTML = '';
    for (const value of values) {
      const label = document.createElement('label');
      const cb = document.createElement('input');
      cb.type = 'checkbox';
      cb.checked = enabledSet.has(value);
      cb.addEventListener('change', () => {
        if (cb.checked) enabledSet.add(value);
        else enabledSet.delete(value);
        onChange();
      });
      label.appendChild(cb);
      label.appendChild(document.createTextNode(value));
      container.appendChild(label);
    }
    // getValues(), called fresh here rather than closing over the `values`
    // snapshot above, matters because these All/None handlers are bound
    // once (guarded below) and outlive later rebuilds of this box (a new
    // nightly data pull can add a workflow/runner that didn't exist at
    // page load) -- a stale closure would keep "All" only re-checking
    // whatever was available the first time this box was ever built.
    function setAll(enabled) {
      const currentValues = getValues();
      for (const cb of container.querySelectorAll('input[type=checkbox]')) cb.checked = enabled;
      enabledSet.clear();
      if (enabled) for (const value of currentValues) enabledSet.add(value);
      onChange();
    }
    const allBtn = document.getElementById(allBtnId);
    const noneBtn = document.getElementById(noneBtnId);
    if (!allBtn.dataset.bound) {
      allBtn.dataset.bound = '1';
      allBtn.addEventListener('click', () => setAll(true));
      noneBtn.dataset.bound = '1';
      noneBtn.addEventListener('click', () => setAll(false));
    }
  }

  function populateControls() {
    document.getElementById('sub-text').innerHTML = config.subText;
    document.getElementById('workflow-label').textContent = config.legsLabel;
    document.getElementById('note').innerHTML = state.view === 'compare' ? config.compareNote(COMPARE_TRAILING_N) : config.trendNote;

    const viewSelect = document.getElementById('view-select');
    viewSelect.value = state.view;
    viewSelect.addEventListener('change', () => {
      state.view = viewSelect.value;
      document.getElementById('note').innerHTML = state.view === 'compare' ? config.compareNote(COMPARE_TRAILING_N) : config.trendNote;
      render();
    });

    const excludeCancelledCheckbox = document.getElementById('exclude-cancelled-checkbox');
    excludeCancelledCheckbox.checked = state.excludeCancelled;
    excludeCancelledCheckbox.addEventListener('change', () => {
      state.excludeCancelled = excludeCancelledCheckbox.checked;
      render();
    });

    const metricSelect = document.getElementById('metric-select');
    metricSelect.innerHTML = '';
    for (const m of config.metrics(state.rows)) {
      const opt = document.createElement('option');
      opt.value = m;
      opt.textContent = m;
      metricSelect.appendChild(opt);
    }
    state.metric = metricSelect.value;
    metricSelect.addEventListener('change', () => {
      state.metric = metricSelect.value;
      render();
    });

    // Two independent facets -- see rowSelected() and runnerSelectionNarrowed().
    const workflowSet = new Set();
    const runnerSet = new Set();
    for (const row of state.rows) {
      workflowSet.add(config.workflowKey(row));
      const runner = config.runnerOf(row);
      if (runner !== null) runnerSet.add(runner);
    }
    state.workflows = [...workflowSet].sort();
    state.runners = [...runnerSet].sort();
    for (const w of state.workflows) state.enabledWorkflows.add(w);
    for (const r of state.runners) state.enabledRunners.add(r);

    buildCheckboxBox('workflow-container', 'workflow-all', 'workflow-none', () => state.workflows, state.enabledWorkflows, render);
    buildCheckboxBox('runner-container', 'runner-all', 'runner-none', () => state.runners, state.enabledRunners, render);
    // Nothing to filter by runner on the perf-benchmarks page (no
    // runnerOf), or on the CI page if a load happens to have no Buildkite
    // rows in view -- hide the box rather than show it empty.
    document.getElementById('runner-box-wrap').hidden = state.runners.length === 0;
  }

  function render() {
    const result = state.view === 'compare' ? renderCompare() : renderTrend();
    updateCsvLink(result);
  }

  // Builds the CSV from the same array the chart just drew from, rather than
  // re-deriving it, so the download can never drift out of sync with what's
  // on screen.
  function updateCsvLink(result) {
    const link = document.getElementById('csv-link');
    let csv;
    if (state.view === 'compare') {
      csv = ['leg,metric,median_s,n_runs']
        .concat(result.map((b) => `${b.leg},${state.metric},${b.value},${b.n}`))
        .join('\n');
      link.download = `${config.csvPrefix}-compare-${state.metric}.csv`;
    } else {
      csv = ['timestamp,leg,metric,value_s']
        .concat(result.flatMap((s) => s.points.map((p) => `${new Date(p.t).toISOString()},${s.leg},${state.metric},${p.v}`)))
        .join('\n');
      link.download = `${config.csvPrefix}-trend-${state.metric}.csv`;
    }
    if (state.csvUrl) URL.revokeObjectURL(state.csvUrl);
    state.csvUrl = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }));
    link.href = state.csvUrl;
  }

  function renderTrend() {
    const canvas = document.getElementById('chart');
    canvas.height = 420;
    const ctx = canvas.getContext('2d');
    const emptyMsg = document.getElementById('empty-message');
    const legend = document.getElementById('legend');
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    legend.innerHTML = '';

    // Group by leg from the already-filtered rows, rather than iterating a
    // precomputed leg list -- which legs even exist depends on the current
    // workflow/runner selections (rowSelected), not just the dataset.
    const filteredRows = state.rows.filter(
      (r) => rowSelected(r) && config.metricValue(r, state.metric) != null && (!state.excludeCancelled || config.isFullSuccessfulRun(r))
    );
    const legOrder = [...new Set(filteredRows.map((r) => legKey(r)))].sort();
    const series = legOrder
      .map((leg, i) => ({
        leg,
        color: COLORS[i % COLORS.length],
        points: filteredRows.filter((r) => legKey(r) === leg).map((r) => ({ t: new Date(r.timestamp).getTime(), v: config.metricValue(r, state.metric) })),
      }))
      .filter((s) => s.points.length > 0);

    if (!series.length) {
      emptyMsg.hidden = false;
      return series;
    }
    emptyMsg.hidden = true;

    const pad = { l: 60, r: 20, t: 20, b: 30 };
    const w = canvas.width - pad.l - pad.r;
    const h = canvas.height - pad.t - pad.b;

    const allT = series.flatMap((s) => s.points.map((p) => p.t));
    const allV = series.flatMap((s) => s.points.map((p) => p.v));
    const tMin = Math.min(...allT), tMax = Math.max(...allT);
    const vMin = 0, vMax = Math.max(...allV) * 1.1;

    const x = (t) => pad.l + (tMax === tMin ? w / 2 : ((t - tMin) / (tMax - tMin)) * w);
    const y = (v) => pad.t + h - ((v - vMin) / (vMax - vMin || 1)) * h;

    const fg = getComputedStyle(document.body).getPropertyValue('--fg') || '#333';
    const muted = getComputedStyle(document.body).getPropertyValue('--muted') || '#888';

    // Axes
    ctx.strokeStyle = muted;
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(pad.l, pad.t);
    ctx.lineTo(pad.l, pad.t + h);
    ctx.lineTo(pad.l + w, pad.t + h);
    ctx.stroke();

    ctx.fillStyle = muted;
    ctx.font = '11px sans-serif';
    ctx.fillText(config.formatDuration(vMax), 4, pad.t + 8);
    ctx.fillText(config.formatDuration(0), 4, pad.t + h);
    ctx.fillText(new Date(tMin).toISOString().slice(0, 10), pad.l, pad.t + h + 16);
    const lastLabel = new Date(tMax).toISOString().slice(0, 10);
    ctx.fillText(lastLabel, pad.l + w - ctx.measureText(lastLabel).width, pad.t + h + 16);

    for (const s of series) {
      const trailing = [];
      ctx.strokeStyle = s.color;
      ctx.lineWidth = 2;
      ctx.beginPath();
      s.points.forEach((p, i) => {
        const px = x(p.t), py = y(p.v);
        if (i === 0) ctx.moveTo(px, py); else ctx.lineTo(px, py);
      });
      ctx.stroke();

      s.points.forEach((p) => {
        const baseline = median(trailing.slice(-10));
        const isRegression = baseline != null && p.v > baseline * 1.2;
        ctx.beginPath();
        ctx.fillStyle = isRegression ? '#e45756' : s.color;
        ctx.arc(x(p.t), y(p.v), isRegression ? 4 : 2.5, 0, Math.PI * 2);
        ctx.fill();
        trailing.push(p.v);
      });

      const item = document.createElement('div');
      item.className = 'legend-item';
      const swatch = document.createElement('span');
      swatch.className = 'swatch';
      swatch.style.background = s.color;
      item.appendChild(swatch);
      item.appendChild(document.createTextNode(s.leg));
      legend.appendChild(item);
    }

    return series;
  }

  function renderCompare() {
    const canvas = document.getElementById('chart');
    const ctx = canvas.getContext('2d');
    const emptyMsg = document.getElementById('empty-message');
    const legend = document.getElementById('legend');
    legend.innerHTML = '';

    // See renderTrend()'s matching comment -- legs come from the filtered
    // rows, not a precomputed list.
    const filteredRows = state.rows.filter(
      (r) => rowSelected(r) && config.metricValue(r, state.metric) != null && (!state.excludeCancelled || config.isFullSuccessfulRun(r))
    );
    const legOrder = [...new Set(filteredRows.map((r) => legKey(r)))].sort();
    const bars = legOrder
      .map((leg, i) => {
        const points = filteredRows.filter((r) => legKey(r) === leg).map((r) => config.metricValue(r, state.metric));
        const value = median(points.slice(-COMPARE_TRAILING_N));
        return { leg, color: COLORS[i % COLORS.length], value, n: Math.min(points.length, COMPARE_TRAILING_N) };
      })
      .filter((b) => b.value != null)
      .sort((a, b) => a.value - b.value);

    if (!bars.length) {
      canvas.height = 420;
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      emptyMsg.hidden = false;
      return bars;
    }
    emptyMsg.hidden = true;

    const rowH = 34;
    const pad = { l: 230, r: 70, t: 20, b: 10 };
    canvas.height = pad.t + pad.b + bars.length * rowH;
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const w = canvas.width - pad.l - pad.r;
    const vMax = Math.max(...bars.map((b) => b.value)) * 1.15;
    const barX = (v) => (v / vMax) * w;

    const fg = getComputedStyle(document.body).getPropertyValue('--fg') || '#333';
    const muted = getComputedStyle(document.body).getPropertyValue('--muted') || '#888';

    ctx.font = '12px sans-serif';
    bars.forEach((b, i) => {
      const rowY = pad.t + i * rowH;
      const barH = rowH * 0.55;
      const barY = rowY + (rowH - barH) / 2;

      ctx.fillStyle = fg;
      ctx.textAlign = 'right';
      ctx.fillText(b.leg, pad.l - 10, rowY + rowH / 2 + 4);

      ctx.fillStyle = b.color;
      ctx.fillRect(pad.l, barY, Math.max(barX(b.value), 1), barH);

      ctx.fillStyle = muted;
      ctx.textAlign = 'left';
      ctx.fillText(`${config.formatDuration(b.value)} (n=${b.n})`, pad.l + barX(b.value) + 8, rowY + rowH / 2 + 4);
    });
    ctx.textAlign = 'left';

    return bars;
  }

  loadData().then((rows) => {
    state.rows = rows;
    populateControls();
    render();
  });
}
