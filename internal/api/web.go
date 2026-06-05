package api

const loginHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>账号调度监控登录</title>
  <style>
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; background: #eef2f7; color: #111827; font-family: "Noto Sans SC", "Microsoft YaHei", sans-serif; }
    form { width: min(420px, calc(100vw - 32px)); padding: 28px; border: 1px solid #dbe3ee; border-radius: 16px; background: #fff; box-shadow: 0 18px 42px rgba(15, 23, 42, .12); }
    h1 { margin: 0 0 10px; font-size: 20px; }
    p { margin: 0 0 18px; color: #64748b; font-size: 13px; line-height: 1.6; }
    input, button { width: 100%; height: 40px; border-radius: 9px; font-size: 14px; }
    input { border: 1px solid #dbe3ee; padding: 0 12px; margin-bottom: 12px; }
    button { border: 0; background: #2563eb; color: #fff; font-weight: 800; cursor: pointer; }
  </style>
</head>
<body>
  <form method="post" action="/login">
    <h1>访问验证</h1>
    <p>请输入服务端配置的 AAD_WEB_TOKEN。验证后会写入本域 Cookie，避免公网直接暴露账号状态和检测接口。</p>
    <input name="token" type="password" autofocus placeholder="AAD_WEB_TOKEN">
    <button type="submit">进入监控</button>
  </form>
</body>
</html>`

const authSetupHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>管理端未启用</title>
  <style>
    body { margin: 0; min-height: 100vh; display: grid; place-items: center; background: #f4f7fb; color: #111827; font-family: "Noto Sans SC", "Microsoft YaHei", Arial, sans-serif; }
    .box { width: min(520px, calc(100vw - 32px)); padding: 26px; border: 1px solid #d8e0ea; border-radius: 8px; background: #fff; box-shadow: 0 14px 34px rgba(15, 23, 42, .08); }
    h1 { margin: 0 0 10px; font-size: 20px; }
    p { margin: 0 0 12px; color: #667085; font-size: 14px; line-height: 1.7; }
    code { padding: 2px 5px; border-radius: 5px; background: #eef2f7; color: #111827; }
    a { color: #2563eb; font-weight: 800; text-decoration: none; }
  </style>
</head>
<body>
  <div class="box">
    <h1>管理端未启用</h1>
    <p>为了避免公网暴露管理接口，服务端必须配置 <code>AAD_WEB_TOKEN</code> 后才能访问后台。</p>
    <p>公开只读状态页仍可访问：<a href="/status">/status</a></p>
  </div>
</body>
</html>`

const statusHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Sub2API - &#x4e2d;&#x8f6c;&#x72b6;&#x6001;</title>
  <style>
    :root {
      --bg: #f9fafb;
      --ink: #111827;
      --muted: #6b7280;
      --panel: #ffffff;
      --line: #e5e7eb;
      --soft: #f8fafc;
      --online: #16a34a;
      --online-bg: #f0fdf4;
      --online-line: #bbf7d0;
      --degraded: #d97706;
      --degraded-bg: #fffbeb;
      --degraded-line: #fde68a;
      --offline: #dc2626;
      --offline-bg: #fef2f2;
      --offline-line: #fecaca;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--ink);
      background: var(--bg);
      font-family: Inter, "Noto Sans SC", "Microsoft YaHei", "PingFang SC", Arial, sans-serif;
      -webkit-font-smoothing: antialiased;
    }
    .shell {
      width: min(1120px, calc(100% - 32px));
      margin: 0 auto;
      padding: 32px 0 40px;
    }
    .header {
      display: flex;
      align-items: flex-end;
      justify-content: space-between;
      gap: 16px;
      margin-bottom: 18px;
    }
    h1 {
      margin: 0;
      font-size: 24px;
      line-height: 1.2;
      font-weight: 750;
      letter-spacing: 0;
    }
    .subtitle {
      margin-top: 6px;
      color: var(--muted);
      font-size: 13px;
    }
    .refresh {
      color: var(--muted);
      font-size: 12px;
      white-space: nowrap;
    }
    .panel {
      overflow: hidden;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 16px;
      box-shadow: 0 1px 3px rgba(0, 0, 0, .04), 0 1px 2px rgba(0, 0, 0, .06);
    }
    .panel-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      min-height: 52px;
      padding: 14px 18px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(180deg, #ffffff, #f8fafc);
    }
    .panel-title {
      font-size: 14px;
      font-weight: 700;
    }
    .count {
      color: var(--muted);
      font-size: 12px;
    }
    .groups {
      display: flex;
      flex-direction: column;
      gap: 18px;
      padding: 14px;
    }
    .group-section {
      min-width: 0;
    }
    .group-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      margin: 0 2px 10px;
    }
    .group-name {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      font-size: 14px;
      font-weight: 800;
    }
    .group-count {
      color: var(--muted);
      font-size: 12px;
      white-space: nowrap;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 12px;
    }
    .station {
      min-height: 172px;
      padding: 16px;
      border: 1px solid var(--line);
      border-radius: 12px;
      background: var(--panel);
      transition: border-color .2s ease, box-shadow .2s ease, transform .2s ease;
    }
    .station:hover {
      border-color: #d1d5db;
      box-shadow: 0 8px 22px rgba(15, 23, 42, .06);
      transform: translateY(-1px);
    }
    .station-top {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 10px;
    }
    .name {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      font-size: 15px;
      font-weight: 700;
    }
    .dot {
      width: 10px;
      height: 10px;
      flex: 0 0 auto;
      border-radius: 999px;
      background: var(--offline);
      box-shadow: 0 0 0 4px var(--offline-bg);
    }
    .station.online .dot { background: var(--online); box-shadow: 0 0 0 4px var(--online-bg); }
    .station.degraded .dot { background: var(--degraded); box-shadow: 0 0 0 4px var(--degraded-bg); }
    .station.offline .dot { background: var(--offline); box-shadow: 0 0 0 4px var(--offline-bg); }
    .tags {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      margin-top: 12px;
    }
    .tag {
      max-width: 100%;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      padding: 4px 8px;
      border: 1px solid var(--line);
      border-radius: 999px;
      color: var(--muted);
      background: var(--soft);
      font-size: 12px;
      line-height: 1;
      font-weight: 700;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      height: 28px;
      margin-top: 18px;
      padding: 0 10px;
      border-radius: 8px;
      border: 1px solid transparent;
      font-size: 13px;
      font-weight: 700;
      white-space: nowrap;
    }
    .badge.online { color: var(--online); background: var(--online-bg); border-color: var(--online-line); }
    .badge.degraded { color: var(--degraded); background: var(--degraded-bg); border-color: var(--degraded-line); }
    .badge.offline { color: var(--offline); background: var(--offline-bg); border-color: var(--offline-line); }
    .station-meta {
      display: flex;
      align-items: flex-end;
      justify-content: space-between;
      gap: 12px;
      margin-top: 18px;
      padding-top: 14px;
      border-top: 1px solid var(--line);
    }
    .meta-label {
      color: var(--muted);
      font-size: 12px;
    }
    .success-rate {
      color: var(--online);
      font-size: 26px;
      line-height: 1;
      font-weight: 800;
      letter-spacing: 0;
      white-space: nowrap;
    }
    .history-wrap {
      margin-top: 14px;
      padding: 12px;
      border: 1px solid var(--line);
      border-radius: 10px;
      background: var(--soft);
      opacity: 1;
      visibility: visible;
    }
    .history-head {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 10px;
      color: var(--muted);
      font-size: 12px;
      margin-bottom: 8px;
    }
    .history {
      display: grid;
      grid-template-columns: repeat(20, minmax(6px, 1fr));
      align-items: center;
      justify-items: center;
      gap: 4px;
      min-height: 18px;
      opacity: 1;
      visibility: visible;
    }
    .bar {
      display: block;
      width: 100%;
      max-width: 10px;
      min-width: 6px;
      height: auto;
      aspect-ratio: 1;
      border-radius: 999px;
      background: #e5e7eb;
      opacity: 1;
      visibility: visible;
      box-shadow: inset 0 0 0 1px rgba(15, 23, 42, .08);
    }
    .bar.success { background: #10b981; box-shadow: 0 0 0 3px rgba(16, 185, 129, .12); }
    .bar.failed { background: #ef4444; box-shadow: 0 0 0 3px rgba(239, 68, 68, .12); }
    .bar.empty { background: #e5e7eb; }
    .empty, .error {
      margin: 14px;
      padding: 30px 16px;
      color: var(--muted);
      text-align: center;
      background: var(--soft);
      border: 1px dashed #d1d5db;
      border-radius: 12px;
    }
    @media (max-width: 760px) {
      .shell { width: min(100% - 24px, 1120px); padding-top: 22px; }
      .header { align-items: flex-start; flex-direction: column; }
      .refresh { white-space: normal; }
      .groups { padding: 12px; gap: 16px; }
      .grid { grid-template-columns: 1fr; }
      .history { grid-template-columns: repeat(20, minmax(8px, 1fr)); }
    }
  </style>
</head>
<body>
  <main class="shell">
    <header class="header">
      <div>
        <h1>Sub2API &#x4e2d;&#x8f6c;&#x72b6;&#x6001;</h1>
        <div class="subtitle">&#x516c;&#x5f00;&#x76d1;&#x63a7;&#x89c6;&#x56fe;</div>
      </div>
      <div class="refresh" id="refresh">正在刷新...</div>
    </header>
    <section class="panel">
      <div class="panel-head">
        <div class="panel-title">&#x63a5;&#x53e3;&#x72b6;&#x6001;</div>
        <div class="count" id="count">0 &#x4e2a;&#x63a5;&#x53e3;</div>
      </div>
      <div class="groups" id="stations"></div>
    </section>
  </main>
  <script>
    async function loadStatus() {
      try {
        const res = await fetch('/v1/public/status', {cache: 'no-store'});
        if (!res.ok) throw new Error('status unavailable');
        const data = await res.json();
        render(data);
        document.getElementById('refresh').textContent = '\u521a\u521a\u5237\u65b0';
      } catch (err) {
        document.getElementById('count').textContent = '0 \u4e2a\u63a5\u53e3';
        document.getElementById('refresh').textContent = '\u5237\u65b0\u5931\u8d25';
        document.getElementById('stations').innerHTML = '<div class="error">\u72b6\u6001\u6682\u65f6\u4e0d\u53ef\u7528</div>';
      }
    }

    function render(data) {
      const stations = Array.isArray(data.stations) ? data.stations : [];
      const target = document.getElementById('stations');
      if (!stations.length) {
        document.getElementById('count').textContent = '0 \u4e2a\u63a5\u53e3';
        target.innerHTML = '<div class="empty">\u6682\u65e0\u63a5\u53e3\u72b6\u6001</div>';
        return;
      }
      const grouped = groupStations(stations);
      document.getElementById('count').textContent = stations.length + ' \u4e2a\u63a5\u53e3 / ' + grouped.length + ' \u4e2a\u5206\u7ec4';
      target.innerHTML = grouped.map(function(group) {
        return '<section class="group-section">' +
          '<div class="group-head"><div class="group-name">' + esc(group.name) + '</div><div class="group-count">' + group.items.length + ' \u4e2a\u63a5\u53e3</div></div>' +
          '<div class="grid">' + group.items.map(renderStation).join('') + '</div>' +
          '</section>';
      }).join('');
    }

    function groupStations(stations) {
      const groups = [];
      const seen = new Map();
      stations.forEach(function(station) {
        stationGroupNames(station).forEach(function(name) {
          if (!seen.has(name)) {
            const group = {name: name, items: []};
            seen.set(name, group);
            groups.push(group);
          }
          seen.get(name).items.push(station);
        });
      });
      return groups;
    }

    function renderStation(station) {
      const state = station.state === 'online' || station.state === 'degraded' || station.state === 'offline' ? station.state : 'offline';
      const history = Array.isArray(station.history) ? station.history.slice(-20) : [];
      const rate = typeof station.success_rate === 'number' ? station.success_rate.toFixed(2).replace(/\.?0+$/, '') + '%' : '-';
      const groups = stationGroupNames(station);
      return '<article class="station ' + state + '">' +
        '<div class="station-top"><div class="name">' + esc(station.name || '\u63a5\u53e3\u72b6\u6001') + '</div><span class="dot"></span></div>' +
        '<div class="tags">' + groups.map(function(group) { return '<span class="tag">' + esc(group) + '</span>'; }).join('') + '</div>' +
        '<span class="badge ' + state + '">' + esc(statusText(state)) + '</span>' +
        '<div class="station-meta"><div class="meta-label">\u53ef\u7528\u7387</div><div class="success-rate">' + esc(rate) + '</div></div>' +
        '<div class="history-wrap">' +
          '<div class="history-head"><span>\u8fd1 20 \u6b21\u68c0\u6d4b</span><span>' + history.length + '/20</span></div>' +
          '<div class="history">' + renderHistory(history) + '</div>' +
        '</div>' +
        '</article>';
    }

    function stationGroupNames(station) {
      const names = Array.isArray(station.groups) ? station.groups : [];
      const out = [];
      names.forEach(function(name) {
        name = String(name == null ? '' : name).trim();
        if (name && !out.includes(name)) out.push(name);
      });
      return out.length ? out : ['\u9ed8\u8ba4\u5206\u7ec4'];
    }

    function renderHistory(history) {
      const padded = Array(Math.max(0, 20 - history.length)).fill('empty').concat(history);
      return padded.map(function(item) {
        const cls = item === 'success' || item === 'failed' ? item : 'empty';
        return '<span class="bar ' + cls + '"></span>';
      }).join('');
    }

    function statusText(state) {
      if (state === 'online') return '\u6b63\u5e38';
      if (state === 'degraded') return '\u89c2\u5bdf\u4e2d';
      return '\u4e0d\u53ef\u7528';
    }

    function esc(value) {
      return String(value == null ? '' : value).replace(/[&<>"']/g, function(s) {
        return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[s];
      });
    }

    loadStatus();
    setInterval(loadStatus, 10000);
  </script>
</body>
</html>`

const indexHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>SUB 渠道状态</title>
  <style>
    :root {
      --bg: #eef2f7;
      --panel: #ffffff;
      --panel-soft: #f8fafc;
      --ink: #111827;
      --muted: #64748b;
      --line: #dbe3ee;
      --nav: #101828;
      --blue: #2563eb;
      --green: #16a34a;
      --amber: #d97706;
      --red: #dc2626;
      --shadow: 0 14px 34px rgba(15, 23, 42, .08);
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(37, 99, 235, .16), transparent 28rem),
        linear-gradient(180deg, #f8fafc 0%, var(--bg) 42%, #e9eef5 100%);
      font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", sans-serif;
    }
    header {
      min-height: 68px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
      padding: 0 26px;
      background: var(--nav);
      color: #fff;
      border-bottom: 1px solid rgba(255, 255, 255, .08);
      box-shadow: 0 2px 14px rgba(15, 23, 42, .2);
    }
    h1 { margin: 0; font-size: 19px; font-weight: 800; letter-spacing: .02em; }
    .header-sub { margin-top: 5px; color: #cbd5e1; font-size: 12px; }
    .header-meta { color: #dbeafe; font-size: 13px; white-space: nowrap; }
    main { padding: 20px 26px 30px; }
    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: stretch;
      gap: 14px;
      margin-bottom: 16px;
    }
    .filters, .control-actions, .control-fields { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
    .filters { flex: 1; }
    input {
      height: 36px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #fff;
      color: var(--ink);
      padding: 0 11px;
      min-width: 230px;
      outline: none;
      box-shadow: 0 1px 0 rgba(15, 23, 42, .03);
    }
    input:focus {
      border-color: var(--blue);
      box-shadow: 0 0 0 3px rgba(37, 99, 235, .12);
    }
    button {
      height: 36px;
      border: 1px solid var(--line);
      border-radius: 8px;
      background: #fff;
      color: var(--ink);
      padding: 0 12px;
      cursor: pointer;
      font-weight: 700;
      box-shadow: 0 1px 0 rgba(15, 23, 42, .04);
    }
    button.primary { background: var(--blue); border-color: var(--blue); color: #fff; }
    button.warn { color: #b42318; background: #fff7ed; border-color: #fed7aa; }
    button:disabled { cursor: not-allowed; opacity: .65; }
    .hint { color: var(--muted); font-size: 12px; line-height: 1.6; }
    .cards {
      display: grid;
      grid-template-columns: repeat(5, minmax(0, 1fr));
      gap: 14px;
      margin-bottom: 16px;
    }
    .card, .control-bar, .panel {
      background: rgba(255, 255, 255, .94);
      border: 1px solid var(--line);
      border-radius: 12px;
      box-shadow: var(--shadow);
    }
    .card { padding: 16px; position: relative; overflow: hidden; }
    .card:before {
      content: "";
      position: absolute;
      inset: 0 auto 0 0;
      width: 4px;
      background: var(--blue);
      opacity: .78;
    }
    .card-label { color: var(--muted); font-size: 13px; margin-bottom: 8px; }
    .card-value { font-size: 28px; font-weight: 850; letter-spacing: -.04em; }
    .card-foot { color: var(--muted); font-size: 12px; margin-top: 6px; }
    .control-bar {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: 12px;
      align-items: center;
      margin-bottom: 16px;
      padding: 14px 16px;
      background: linear-gradient(180deg, #fff, #f8fafc);
    }
    .control-fields strong { font-size: 14px; }
    .control-fields label { color: var(--muted); font-size: 12px; font-weight: 700; }
    .control-fields input { width: 110px; min-width: 90px; }
    .panel { overflow: hidden; }
    .jobs-panel { margin-top: 16px; }
    .panel-title {
      padding: 14px 16px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(180deg, #ffffff, #f8fafc);
    }
    .panel-title strong { font-size: 15px; }
    .table-wrap { overflow: auto; }
    table { width: 100%; border-collapse: separate; border-spacing: 0; min-width: 1260px; }
    th, td {
      padding: 12px 14px;
      border-bottom: 1px solid var(--line);
      text-align: left;
      font-size: 13px;
      vertical-align: middle;
    }
    th {
      color: #475569;
      background: #f1f5f9;
      font-weight: 800;
      white-space: nowrap;
    }
    tr:hover td { background: #f8fbff; }
    .name { font-weight: 800; color: #0f172a; }
    .sub { color: var(--muted); font-size: 12px; margin-top: 3px; }
    .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
    .badge {
      display: inline-flex;
      align-items: center;
      height: 24px;
      padding: 0 9px;
      border-radius: 999px;
      font-size: 12px;
      font-weight: 800;
      border: 1px solid transparent;
      white-space: nowrap;
    }
    .on, .healthy, .succeeded { color: #067647; background: #ecfdf3; border-color: #abefc6; }
    .suspect, .warning { color: #b54708; background: #fffaeb; border-color: #fedf89; }
    .off, .disabled, .failed { color: #b42318; background: #fef3f2; border-color: #fecdca; }
    .paused { color: #475467; background: #f2f4f7; border-color: #d0d5dd; }
    .running { color: #175cd3; background: #eff8ff; border-color: #b2ddff; }
    .rate-cell { min-width: 190px; }
    .rate-top {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 7px;
    }
    .rate-value { font-size: 18px; font-weight: 850; letter-spacing: -.03em; }
    .rate-bar {
      height: 8px;
      overflow: hidden;
      background: #e2e8f0;
      border-radius: 999px;
    }
    .rate-fill {
      height: 100%;
      width: 0%;
      border-radius: inherit;
      background: linear-gradient(90deg, #22c55e, #16a34a);
    }
    .rate-fill.mid { background: linear-gradient(90deg, #f59e0b, #d97706); }
    .rate-fill.low { background: linear-gradient(90deg, #ef4444, #dc2626); }
    .status-stack { display: grid; gap: 6px; justify-items: start; }
    .time-grid { display: grid; gap: 3px; color: var(--muted); }
    .empty {
      padding: 22px 16px;
      color: var(--muted);
      text-align: center;
      background: #fff;
    }
    pre {
      margin: 12px 0 0;
      padding: 13px;
      max-height: 340px;
      overflow: auto;
      background: #0b1220;
      color: #dbeafe;
      border-radius: 10px;
      font-size: 12px;
      line-height: 1.55;
      white-space: pre-wrap;
      box-shadow: inset 0 0 0 1px rgba(255, 255, 255, .06);
    }
    @media (max-width: 1100px) {
      .cards { grid-template-columns: repeat(2, minmax(0, 1fr)); }
      .toolbar, .control-bar { grid-template-columns: 1fr; flex-direction: column; }
      .control-actions { justify-content: flex-start; }
    }
    @media (max-width: 700px) {
      header { padding: 14px 16px; align-items: flex-start; flex-direction: column; }
      main { padding: 16px; }
      .cards { grid-template-columns: 1fr; }
      input { min-width: 100%; }
      .filters, .control-fields, .control-actions { width: 100%; }
      .control-fields label, .control-fields input, button { width: 100%; }
    }
  </style>
</head>
<body>
  <header>
    <div>
      <h1>SUB 渠道状态</h1>
      <div class="header-sub">API 账号检测 / 历史请求成功率 / 自动轮询控制</div>
    </div>
    <div class="header-meta" id="health">服务检查中...</div>
  </header>

  <main>
    <div class="toolbar">
      <div class="filters">
        <input id="search" placeholder="搜索渠道名称 / ID / 平台 / 错误信息" oninput="render()">
        <input id="model" value="gpt-4o-mini" title="手动检测时传入 sub2api 的模型">
        <button class="primary" onclick="loadAll()">刷新状态</button>
      </div>
      <div class="hint">账号源：sub2api accounts；导入所有 API key 凭据账号并排除邮箱账号；检测入口：sub2api 账号管理的检测连接。</div>
    </div>

    <div class="cards">
      <div class="card"><div class="card-label">渠道总数</div><div class="card-value" id="stat-total">0</div><div class="card-foot">API 凭据账号</div></div>
      <div class="card"><div class="card-label">调度开启</div><div class="card-value" id="stat-on">0</div><div class="card-foot">sub2api schedulable</div></div>
      <div class="card"><div class="card-label">停止检测</div><div class="card-value" id="stat-paused">0</div><div class="card-foot">不参与自动轮询</div></div>
      <div class="card"><div class="card-label">历史请求成功率</div><div class="card-value" id="stat-rate">0%</div><div class="card-foot" id="stat-rate-foot">0 / 0</div></div>
      <div class="card"><div class="card-label">最近任务</div><div class="card-value" id="stat-running">0</div><div class="card-foot">当前检测中</div></div>
    </div>

    <section class="control-bar">
      <div class="control-fields">
        <strong>轮询检测</strong>
        <label>间隔秒 <input id="control-interval" type="number" min="10" max="3600" value="240"></label>
        <label>每轮个数 <input id="control-batch" type="number" min="1" max="50" value="5"></label>
        <label>模型 <input id="control-model" value="gpt-4o-mini"></label>
        <span class="hint" id="control-status">未开启</span>
      </div>
      <div class="control-actions">
        <button class="primary" onclick="saveProbeControl(true)">保存并开启</button>
        <button onclick="saveProbeControl(false)">关闭轮询</button>
        <button onclick="runProbeBatchOnce()">立即检测一轮</button>
      </div>
    </section>

    <section class="panel">
      <div class="panel-title">
        <strong>渠道状态</strong>
        <span class="hint">历史请求成功率 = 当前服务累计检测成功数 / 累计检测次数，统计保存在 sub2api accounts.extra.aad_health。</span>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>渠道</th>
              <th>平台</th>
              <th>上游倍率</th>
              <th>优先级</th>
              <th>调度状态</th>
              <th>检测状态</th>
              <th>历史请求成功率</th>
              <th>最近请求</th>
              <th>最后错误</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody id="accounts"></tbody>
        </table>
      </div>
    </section>

    <section class="panel jobs-panel">
      <div class="panel-title">
        <strong>最近检测记录</strong>
        <span class="hint">详情只保留检测结果、错误、耗时、HTTP 状态和少量响应摘要；刷新后保留内存中的最近 20 条。</span>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>开始时间</th>
              <th>渠道</th>
              <th>触发</th>
              <th>状态</th>
              <th>结果</th>
              <th>耗时</th>
              <th>错误</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody id="jobs"></tbody>
        </table>
      </div>
    </section>

    <pre id="output">等待操作...</pre>
  </main>

  <script>
    let accounts = [];
    let jobs = [];
    let activeJobID = '';
    let probeControl = null;
    const probing = new Set();

    async function api(path, options = {}) {
      const timeoutMs = options.timeoutMs || 120000;
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), timeoutMs);
      const fetchOptions = Object.assign({}, options, {signal: controller.signal});
      delete fetchOptions.timeoutMs;
      try {
        const res = await fetch(path, fetchOptions);
        const text = await res.text();
        let data = null;
        try { data = text ? JSON.parse(text) : null; } catch { data = text; }
        if (!res.ok) throw data;
        return data;
      } finally {
        clearTimeout(timer);
      }
    }

    async function checkHealth() {
      try {
        const data = await api('/healthz');
        document.getElementById('health').textContent = '服务状态：' + data.status;
      } catch {
        document.getElementById('health').textContent = '服务状态：异常';
      }
    }

    async function loadAll() {
      await Promise.all([loadAccounts(), loadProbeJobs(), loadProbeControl()]);
    }

    async function loadAccounts() {
      try {
        accounts = await api('/v1/accounts', {timeoutMs: 30000});
        render();
      } catch (err) {
        showOutput({success: false, error: '加载账号失败', detail: normalizeError(err)});
      }
    }

    async function loadProbeJobs() {
      try {
        jobs = await api('/v1/dispatch/probe-jobs', {timeoutMs: 10000});
        probing.clear();
        jobs.forEach(job => {
          if (job.status === 'running') probing.add(job.account_id);
        });
        render();
        renderJobs();
        const active = jobs.find(job => job.job_id === activeJobID);
        if (active) {
          showOutput(active);
          if (active.status !== 'running') activeJobID = '';
        } else if (!activeJobID && jobs.length) {
          showOutput(jobs[0]);
        }
      } catch (err) {
        showOutput({success: false, error: '加载检测记录失败', detail: normalizeError(err)});
      }
    }

    async function loadProbeControl() {
      try {
        probeControl = await api('/v1/probe-control', {timeoutMs: 10000});
        document.getElementById('control-interval').value = probeControl.interval_seconds || 240;
        document.getElementById('control-batch').value = probeControl.batch_size || 5;
        document.getElementById('control-model').value = probeControl.model || 'gpt-4o-mini';
        renderProbeControl();
      } catch (err) {
        showOutput({success: false, error: '加载轮询配置失败', detail: normalizeError(err)});
      }
    }

    async function saveProbeControl(enabled) {
      try {
        probeControl = await api('/v1/probe-control', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({
            enabled: enabled,
            interval_seconds: Number(document.getElementById('control-interval').value || 240),
            batch_size: Number(document.getElementById('control-batch').value || 5),
            model: document.getElementById('control-model').value.trim() || 'gpt-4o-mini'
          }),
          timeoutMs: 10000
        });
        renderProbeControl();
        showOutput({status: '轮询配置已保存', control: probeControl});
      } catch (err) {
        showOutput({success: false, error: '保存轮询配置失败', detail: normalizeError(err)});
      }
    }

    async function runProbeBatchOnce() {
      try {
        const data = await api('/v1/probe-control/run-once', {method: 'POST', timeoutMs: 30000});
        showOutput(data);
        await loadProbeJobs();
        await loadAccounts();
      } catch (err) {
        showOutput({success: false, error: '立即检测一轮失败', detail: normalizeError(err)});
      }
    }

    async function setProbePaused(accountID, paused) {
      try {
        const account = await api('/v1/accounts/probe-paused', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({account_id: accountID, paused: paused}),
          timeoutMs: 10000
        });
        showOutput({status: paused ? '已停止检测' : '已恢复检测', account: account});
        await loadAccounts();
      } catch (err) {
        showOutput({success: false, error: '更新停检状态失败', detail: normalizeError(err)});
      }
    }

    function renderProbeControl() {
      if (!probeControl) return;
      const text = probeControl.enabled
        ? '已开启：每 ' + probeControl.interval_seconds + ' 秒检测 ' + probeControl.batch_size + ' 个'
        : '未开启';
      const extra = probeControl.last_run_at
        ? '；上次 ' + formatTime(probeControl.last_run_at) + '，创建 ' + probeControl.last_run_count + ' 个任务'
        : '';
      const error = probeControl.last_error ? '；错误：' + probeControl.last_error : '';
      document.getElementById('control-status').textContent = text + extra + error;
    }

    function render() {
      const keyword = document.getElementById('search').value.trim().toLowerCase();
      const filtered = accounts.filter(account => {
        const text = [account.account_id, account.name, account.platform, account.upstream_rate_multiplier, account.priority, account.health_status, account.last_error_type, account.last_error_message].join(' ').toLowerCase();
        return !keyword || text.includes(keyword);
      });
      const totals = successTotals(accounts);

      document.getElementById('stat-total').textContent = accounts.length;
      document.getElementById('stat-on').textContent = accounts.filter(a => a.dispatch_enabled).length;
      document.getElementById('stat-paused').textContent = accounts.filter(a => a.probe_paused).length;
      document.getElementById('stat-running').textContent = probing.size;
      document.getElementById('stat-rate').textContent = totals.total ? Math.round(totals.success * 100 / totals.total) + '%' : '0%';
      document.getElementById('stat-rate-foot').textContent = totals.success + ' / ' + totals.total + ' 成功';

      const target = document.getElementById('accounts');
      if (!filtered.length) {
        target.innerHTML = '<tr><td colspan="10" class="empty">暂无渠道账号，或搜索条件没有匹配结果</td></tr>';
        return;
      }
      target.innerHTML = filtered.map(account => {
        const isProbing = probing.has(account.account_id);
        const probeClass = account.probe_paused ? 'paused' : (isProbing ? 'running' : healthClass(account.health_status));
        const probeText = account.probe_paused ? '已停检' : (isProbing ? '检测中' : healthText(account.health_status));
        return '<tr>' +
          '<td><div class="name">' + esc(account.name || account.account_id) + '</div><div class="sub mono">ID ' + esc(account.account_id) + '</div></td>' +
          '<td><span class="badge running">' + esc(account.platform || '-') + '</span></td>' +
          '<td><span class="badge paused">' + esc(formatRateMultiplier(account.upstream_rate_multiplier)) + '</span></td>' +
          '<td><span class="badge paused">P' + esc(formatPriority(account.priority)) + '</span></td>' +
          '<td><div class="status-stack"><span class="badge ' + (account.dispatch_enabled ? 'on' : 'off') + '">' + (account.dispatch_enabled ? '开启' : '关闭') + '</span><span class="sub">' + esc(account.status || '-') + '</span></div></td>' +
          '<td><div class="status-stack"><span class="badge ' + probeClass + '">' + probeText + '</span><span class="sub">' + esc(account.health_status || 'unknown') + '</span></div></td>' +
          '<td class="rate-cell">' + renderRate(account) + '</td>' +
          '<td><div class="time-grid"><span>检测：' + formatTime(account.last_probe_at) + '</span><span>成功：' + formatTime(account.last_success_at) + '</span><span>失败：' + formatTime(account.last_failed_at) + '</span></div></td>' +
          '<td><div>' + esc(account.last_error_type || '-') + '</div><div class="sub">' + esc(account.last_error_message || '') + '</div></td>' +
          '<td><button data-probe-id="' + escAttr(account.account_id) + '" onclick="probeAccount(this.dataset.probeId)"' + (isProbing ? ' disabled' : '') + '>' + (isProbing ? '检测中...' : '检测连接') + '</button> ' +
          '<button class="' + (account.probe_paused ? '' : 'warn') + '" data-account-id="' + escAttr(account.account_id) + '" data-paused="' + (account.probe_paused ? 'false' : 'true') + '" onclick="setProbePaused(this.dataset.accountId, this.dataset.paused === ' + "'true'" + ')">' + (account.probe_paused ? '恢复检测' : '停止检测') + '</button></td>' +
          '</tr>';
      }).join('');
    }

    function renderJobs() {
      const target = document.getElementById('jobs');
      if (!jobs.length) {
        target.innerHTML = '<tr><td colspan="8" class="empty">暂无检测记录</td></tr>';
        return;
      }
      target.innerHTML = jobs.map(job => {
        const resultText = job.status === 'running' ? '检测中' : (job.success ? '成功' : '失败');
        const errorText = job.error_message || job.error || job.error_type || '-';
        return '<tr>' +
          '<td class="mono">' + formatTime(job.started_at) + '</td>' +
          '<td><div class="name">' + esc(job.name || job.account_id) + '</div><div class="sub mono">ID ' + esc(job.account_id) + '</div></td>' +
          '<td>' + esc(formatTrigger(job.trigger)) + '</td>' +
          '<td><span class="badge ' + esc(job.status) + '">' + esc(formatJobStatus(job.status)) + '</span></td>' +
          '<td>' + esc(resultText) + '</td>' +
          '<td class="mono">' + formatElapsed(job.elapsed_ms) + '</td>' +
          '<td><div>' + esc(job.error_type || '-') + '</div><div class="sub">' + esc(errorText) + '</div></td>' +
          '<td><button data-job-id="' + escAttr(job.job_id) + '" onclick="showJobDetail(this.dataset.jobId)">详情</button></td>' +
          '</tr>';
      }).join('');
    }

    async function probeAccount(accountID) {
      if (probing.has(accountID)) return;
      const model = document.getElementById('model').value.trim();
      const startedAt = Date.now();
      probing.add(accountID);
      render();
      showOutput({
        status: '检测中',
        account_id: accountID,
        model: model,
        message: '正在调用 sub2api 账号管理的检测连接接口，请等待返回。'
      });
      try {
        const job = await api('/v1/dispatch/probe', {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({account_id: accountID, model: model}),
          timeoutMs: 12000
        });
        job.client_elapsed_ms = Date.now() - startedAt;
        activeJobID = job.job_id;
        upsertJob(job);
        showOutput(job);
        await loadProbeJobs();
        await loadAccounts();
      } catch (err) {
        probing.delete(accountID);
        render();
        showOutput({
          success: false,
          account_id: accountID,
          error: '检测请求失败',
          elapsed_ms: Date.now() - startedAt,
          detail: normalizeError(err)
        });
      }
    }

    async function showJobDetail(jobID) {
      try {
        const job = await api('/v1/dispatch/probe-jobs/' + encodeURIComponent(jobID), {timeoutMs: 10000});
        activeJobID = job.job_id;
        upsertJob(job);
        showOutput(job);
      } catch (err) {
        showOutput({success: false, error: '加载检测详情失败', detail: normalizeError(err)});
      }
    }

    function upsertJob(job) {
      jobs = [job].concat(jobs.filter(item => item.job_id !== job.job_id)).slice(0, 20);
      if (job.status === 'running') {
        probing.add(job.account_id);
      } else {
        probing.delete(job.account_id);
        if (activeJobID === job.job_id) activeJobID = '';
      }
      render();
      renderJobs();
    }

    function renderRate(account) {
      const total = Number(account.probe_total_count || 0);
      const success = Number(account.probe_success_count || 0);
      const errors = Number(account.probe_error_count || 0);
      const pct = total ? Math.round(success * 100 / total) : 0;
      const fillClass = pct >= 90 || total === 0 ? '' : (pct >= 60 ? ' mid' : ' low');
      const label = total ? pct + '%' : '-';
      return '<div class="rate-top"><span class="rate-value">' + label + '</span><span class="sub mono">' + success + '/' + total + '</span></div>' +
        '<div class="rate-bar"><div class="rate-fill' + fillClass + '" style="width:' + pct + '%"></div></div>' +
        '<div class="sub">成功 ' + success + ' 次 / 失败 ' + errors + ' 次</div>';
    }

    function successTotals(items) {
      return items.reduce((acc, account) => {
        acc.total += Number(account.probe_total_count || 0);
        acc.success += Number(account.probe_success_count || 0);
        return acc;
      }, {total: 0, success: 0});
    }

    function healthClass(status) {
      if (status === 'healthy') return 'healthy';
      if (status === 'suspect' || status === 'probing') return 'warning';
      if (status === 'disabled') return 'disabled';
      return 'paused';
    }

    function healthText(status) {
      if (status === 'healthy') return '正常';
      if (status === 'suspect') return '可疑';
      if (status === 'probing') return '检测中';
      if (status === 'disabled') return '不可用';
      return '未知';
    }

    function formatTime(value) { return value ? new Date(value).toLocaleString() : '-'; }
    function formatPriority(value) {
      const priority = Number(value || 0);
      return priority > 0 ? priority : '-';
    }
    function formatRateMultiplier(value) {
      const rate = value === undefined || value === null || value === '' ? 1 : Number(value);
      if (!Number.isFinite(rate)) return '-';
      return rate.toLocaleString(undefined, {maximumFractionDigits: 4}) + 'x';
    }
    function formatElapsed(value) {
      const ms = Number(value || 0);
      if (ms < 1000) return ms + 'ms';
      return Math.round(ms / 1000) + 's';
    }
    function formatJobStatus(status) {
      if (status === 'running') return '检测中';
      if (status === 'succeeded') return '成功';
      if (status === 'failed') return '失败';
      return status || '-';
    }
    function formatTrigger(trigger) {
      if (trigger === 'manual') return '手动';
      if (trigger === 'manual_batch') return '手动批量';
      if (trigger === 'auto_batch') return '自动轮询';
      return trigger || '-';
    }
    function showOutput(value) {
      document.getElementById('output').textContent = typeof value === 'string' ? value : JSON.stringify(value, null, 2);
    }
    function normalizeError(err) {
      if (err && err.name === 'AbortError') {
        return '检测请求超过 10 秒未返回，已按超时失败处理；如果任务仍在列表中，请等待最近检测记录刷新。';
      }
      return err || '未知错误';
    }
    function esc(value) {
      return String(value ?? '').replace(/[&<>"']/g, s => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[s]));
    }
    function escAttr(value) { return esc(value).replace(/\\/g, '\\\\'); }

    checkHealth();
    loadAll();
    setInterval(loadAccounts, 10000);
    setInterval(loadProbeJobs, 2000);
    setInterval(loadProbeControl, 10000);
  </script>
</body>
</html>`
