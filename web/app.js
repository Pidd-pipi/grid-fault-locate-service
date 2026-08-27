/**
 * app.js —— 前端应用入口：hash 路由 + 五个页面。
 * 页面与后端 REST API 一一对应，全部真实消费接口。
 */
(function () {
  const app = document.getElementById('app');
  const fmtDT = (t) => (t ? new Date(t).toLocaleString('zh-CN', { hour12: false }) : '—');
  const esc = (s) => String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

  // ---------- 页面：配网总览 ----------
  async function overviewPage() {
    app.innerHTML = '<div class="loading">加载总览…</div>';
    let data;
    try {
      data = await GridAPI.get('/api/overview');
    } catch (e) {
      app.innerHTML = '<div class="error-box">总览加载失败：' + esc(e.message) + '</div>';
      return;
    }
    const longList = (data.longOutageFaults || []).map((f) =>
      '<li>⚠ <strong>' + esc(f.feederName || f.feederId) + '</strong> · ' + esc(f.id) + ' · 定位 ' + fmtDT(f.locatedAt) + ' 已超 ' + GridEnums.LONG_OUTAGE_MINUTES + ' 分钟未复电</li>').join('') || '<li class="muted">当前无长时停电关注</li>';

    const feederCards = (data.feeders || []).map((f) =>
      '<div class="card feeder-card">' +
        '<div class="card-head"><strong>' + esc(f.name) + '</strong><span class="badge ' + (f.status === 'active' ? 'badge-green' : 'badge-gray') + '">' + (GridLabels.feederStatus[f.status] || f.status) + '</span></div>' +
        '<div class="card-body">' +
          '<div class="kv"><span>变电站</span><b>' + esc(f.substation) + '</b></div>' +
          '<div class="kv"><span>电压等级</span><b>' + esc(f.voltageLevel) + '</b></div>' +
          '<div class="kv"><span>开关/区段</span><b>' + f.switchCount + ' / ' + f.sectionCount + '</b></div>' +
          '<div class="kv"><span>指示器</span><b>' + f.indicatorCount + '</b></div>' +
          (f.activeFaults > 0 ? '<div class="kv"><span>在途故障</span><b class="text-red">' + f.activeFaults + '</b></div>' : '') +
        '</div></div>').join('') || '<div class="empty">暂无线路</div>';

    const activeCards = data.activeFaults.length
      ? '<div class="fault-list" id="overview-faults"></div>'
      : '<div class="empty">当前无在途故障</div>';

    app.innerHTML =
      '<h2>配网总览</h2>' +
      '<div class="stat-row">' +
        stat('线路', data.feederCount) + stat('开关', data.switchCount) + stat('区段', data.sectionCount) +
        stat('指示器', data.indicatorCount) + stat('翻牌中', data.triggeredCount, data.triggeredCount > 0 ? 'red' : '') +
        stat('在途故障', data.activeFaults.length, data.activeFaults.length > 0 ? 'red' : '') +
      '</div>' +
      '<div class="grid-2col">' +
        '<section><h3>线路状态</h3><div class="feeder-grid">' + feederCards + '</div></section>' +
        '<section><h3>长时停电关注（&gt; ' + GridEnums.LONG_OUTAGE_MINUTES + ' 分钟）</h3><ul class="watch-list">' + longList + '</ul></section>' +
      '</div>' +
      '<section><h3>在途故障事件</h3>' + activeCards + '</section>';

    if (data.activeFaults.length) {
      const box = document.getElementById('overview-faults');
      data.activeFaults.forEach((f) => {
        const holder = document.createElement('div');
        GridComponents.FaultCard(holder, { fault: f, compact: true, sections: {} });
        box.appendChild(holder);
      });
    }
  }

  // ---------- 页面：拓扑管理 ----------
  async function topologyPage() {
    app.innerHTML = '<div class="loading">加载拓扑…</div>';
    const feeders = await GridAPI.get('/api/feeders').catch(() => []);
    let feederList = feeders;
    if (!Array.isArray(feederList)) feederList = [];

    if (!feederList.length) {
      app.innerHTML = '<div class="empty">暂无线路，请先创建线路</div>';
      return;
    }
    const hashSel = document.getElementById('feeder-select');
    app.innerHTML =
      '<h2>拓扑管理</h2>' +
      '<div class="toolbar"><label>选择线路</label><select id="feeder-select">' +
        feederList.map((f) => '<option value="' + f.id + '">' + esc(f.name) + '</option>').join('') +
      '</select>' +
      '<button class="btn btn-primary" id="btn-add-switch">新增开关</button>' +
      '<button class="btn btn-primary" id="btn-add-section">新增区段</button>' +
      '<button class="btn btn-ghost" id="btn-refresh-topo">刷新</button></div>' +
      '<div id="topo-graph"></div>' +
      '<div class="grid-2col">' +
        '<section><h3>开关节点</h3><div id="switch-table"></div></section>' +
        '<section><h3>线路区段</h3><div id="section-table"></div></section>' +
      '</div>' +
      '<div id="modal-root"></div>';

    const sel = document.getElementById('feeder-select');
    sel.addEventListener('change', renderTopo);
    document.getElementById('btn-refresh-topo').addEventListener('click', renderTopo);
    document.getElementById('btn-add-switch').addEventListener('click', () => promptAddSwitch(sel.value));
    document.getElementById('btn-add-section').addEventListener('click', () => promptAddSection(sel.value));

    async function renderTopo() {
      const fid = sel.value;
      const graph = document.getElementById('topo-graph');
      const switchBox = document.getElementById('switch-table');
      const sectionBox = document.getElementById('section-table');
      graph.innerHTML = '<div class="loading">加载拓扑…</div>';
      let tp;
      try {
        tp = await GridAPI.get('/api/feeders/' + encodeURIComponent(fid) + '/topology');
      } catch (e) {
        graph.innerHTML = '<div class="error-box">' + esc(e.message) + '</div>';
        return;
      }
      GridComponents.TopologyGraph(graph, {
        topology: tp,
        interactive: true,
        onSwitchClick: (sw) => toggleSwitch(fid, sw),
        onSectionClick: (sec) => { if (confirm('删除区段 ' + sec.name + ' ？')) removeSection(fid, sec.id); },
      });
      // 开关表格
      const rows = tp.switches.map((sw) =>
        '<tr><td><strong>' + esc(sw.name) + '</strong></td>' +
        '<td>' + (GridLabels.switchType[sw.switchType] || sw.switchType) + '</td>' +
        '<td><span class="badge ' + (sw.status === 'closed' ? 'badge-green' : 'badge-red') + '">' + (GridLabels.switchStatus[sw.status] || sw.status) + '</span></td>' +
        '<td class="ops"><button class="btn btn-sm btn-warn" data-toggle="' + sw.id + '">分/合闸</button>' +
        '<button class="btn btn-sm btn-danger" data-del-sw="' + sw.id + '">删除</button></td></tr>').join('');
      switchBox.innerHTML = '<table class="table"><thead><tr><th>名称</th><th>类型</th><th>状态</th><th>操作</th></tr></thead><tbody>' + rows + '</tbody></table>';
      switchBox.querySelectorAll('[data-toggle]').forEach((b) => b.addEventListener('click', () => toggleSwitch(fid, b.getAttribute('data-toggle'))));
      switchBox.querySelectorAll('[data-del-sw]').forEach((b) => b.addEventListener('click', () => removeSwitch(fid, b.getAttribute('data-del-sw'))));

      // 区段表格
      const srows = tp.sections.map((sec) =>
        '<tr><td><strong>' + esc(sec.name) + '</strong></td>' +
        '<td>' + esc(sec.upstreamSwitchId) + ' → ' + esc(sec.downstreamSwitchId) + '</td>' +
        '<td>' + sec.lengthKm + ' km</td>' +
        '<td>' + (sec.isCandidate ? '<span class="badge badge-orange">候选</span>' : '') + (sec.isolated ? '<span class="badge badge-red">隔离</span>' : '') + '</td>' +
        '<td class="ops"><button class="btn btn-sm btn-danger" data-del-sec="' + sec.id + '">删除</button></td></tr>').join('');
      sectionBox.innerHTML = '<table class="table"><thead><tr><th>名称</th><th>端点</th><th>长度</th><th>标记</th><th>操作</th></tr></thead><tbody>' + srows + '</tbody></table>';
      sectionBox.querySelectorAll('[data-del-sec]').forEach((b) => b.addEventListener('click', () => removeSection(fid, b.getAttribute('data-del-sec'))));
    }

    async function toggleSwitch(fid, swId) {
      try {
        await GridAPI.post('/api/feeders/' + fid + '/switches/' + swId + '/toggle');
        renderTopo();
      } catch (e) { alert('分合闸失败：' + e.message); }
    }
    async function removeSwitch(fid, swId) {
      if (!confirm('删除开关？')) return;
      try { await GridAPI.del('/api/feeders/' + fid + '/switches/' + swId); renderTopo(); }
      catch (e) { alert('删除失败：' + e.message); }
    }
    async function removeSection(fid, secId) {
      try { await GridAPI.del('/api/feeders/' + fid + '/sections/' + secId); renderTopo(); }
      catch (e) { alert('删除失败：' + e.message); }
    }

    function promptAddSwitch(fid) {
      const type = prompt('开关类型 sectionalizer / tie / feeder_outlet：', 'sectionalizer');
      if (!type) return;
      const name = prompt('开关名称：', '分段开关' + Math.floor(Math.random() * 100));
      if (!name) return;
      GridAPI.post('/api/feeders/' + fid + '/switches', { name, switchType: type })
        .then(renderTopo)
        .catch((e) => alert('新增开关失败：' + e.message));
    }

    function promptAddSection(fid) {
      const up = prompt('上游开关 ID：');
      const down = prompt('下游开关 ID：');
      const name = prompt('区段名称：', '区段' + Math.floor(Math.random() * 100));
      const len = prompt('长度(km)：', '1.0');
      if (!up || !down || !name) return;
      GridAPI.post('/api/feeders/' + fid + '/sections', { name, upstreamSwitchId: up, downstreamSwitchId: down, lengthKm: parseFloat(len) || 1 })
        .then(renderTopo)
        .catch((e) => alert('新增区段失败：' + e.message));
    }

    renderTopo();
  }

  // ---------- 页面：指示器 ----------
  async function indicatorsPage() {
    app.innerHTML = '<div class="loading">加载指示器…</div>';
    let feeders = [];
    try { feeders = await GridAPI.get('/api/feeders'); } catch (e) { /* ignore */ }
    const feederNames = {};
    feeders.forEach((f) => { feederNames[f.id] = f.name; });
    const [indicators, topologies] = await Promise.all([
      GridAPI.get('/api/indicators').catch(() => []),
      Promise.all(feeders.map((f) => GridAPI.get('/api/feeders/' + f.id + '/topology').catch(() => null))),
    ]);
    const sectionNames = {};
    topologies.forEach((tp) => { if (tp) tp.sections.forEach((s) => { sectionNames[s.id] = s.name; }); });

    app.innerHTML =
      '<h2>故障指示器</h2>' +
      '<div class="toolbar">' +
        '<label>过滤</label>' +
        '<select id="ind-filter"><option value="">全部</option><option value="triggered">仅翻牌</option><option value="suspicious">仅可疑</option></select>' +
        '<button class="btn btn-primary" id="btn-add-ind">新增指示器</button>' +
        '<button class="btn btn-ghost" id="btn-refresh-ind">刷新</button>' +
      '</div><div id="ind-table"></div>';

    function renderTable() {
      const filter = document.getElementById('ind-filter').value;
      let list = indicators;
      if (filter === 'triggered') list = list.filter((i) => i.status === 'triggered');
      if (filter === 'suspicious') list = list.filter((i) => i.suspicious);
      const box = document.getElementById('ind-table');
      GridComponents.IndicatorTable(box, {
        indicators: list,
        sectionNames,
        feederNames,
        onSignal: async (ind, status) => {
          try {
            await GridAPI.post('/api/indicators/' + ind.id + '/signal', { status });
            const idx = indicators.findIndex((x) => x.id === ind.id);
            if (idx >= 0) indicators[idx] = await GridAPI.get('/api/indicators/' + ind.id);
            renderTable();
          } catch (e) { alert('信号上报失败：' + e.message); }
        },
        onSuspicious: async (ind, suspicious) => {
          try {
            await GridAPI.post('/api/indicators/' + ind.id + '/suspicious', { suspicious, reason: suspicious ? '人工核验标记' : '' });
            const idx = indicators.findIndex((x) => x.id === ind.id);
            if (idx >= 0) indicators[idx] = await GridAPI.get('/api/indicators/' + ind.id);
            renderTable();
          } catch (e) { alert('标记失败：' + e.message); }
        },
      });
    }

    document.getElementById('ind-filter').addEventListener('change', renderTable);
    document.getElementById('btn-refresh-ind').addEventListener('click', () => { location.reload(); });
    document.getElementById('btn-add-ind').addEventListener('click', () => {
      const sectionId = prompt('挂接区段 ID：');
      const name = prompt('指示器名称：', 'FI-' + Math.floor(Math.random() * 1000));
      if (!sectionId || !name) return;
      GridAPI.post('/api/indicators', { name, sectionId, position: 0.5 })
        .then(() => { location.reload(); })
        .catch((e) => alert('新增指示器失败：' + e.message));
    });
    renderTable();
  }

  // ---------- 页面：故障事件 ----------
  async function faultsPage() {
    app.innerHTML = '<div class="loading">加载故障事件…</div>';
    let feeders = [];
    try { feeders = await GridAPI.get('/api/feeders'); } catch (e) { /* ignore */ }
    const feederNames = {};
    feeders.forEach((f) => { feederNames[f.id] = f.name; });
    const allTopo = await Promise.all(feeders.map((f) => GridAPI.get('/api/feeders/' + f.id + '/topology').catch(() => null)));
    const sectionNames = {};
    allTopo.forEach((tp) => { if (tp) tp.sections.forEach((s) => { sectionNames[s.id] = s.name; }); });
    let faults = await GridAPI.get('/api/faults').catch(() => []);

    app.innerHTML =
      '<h2>故障事件</h2>' +
      '<div class="toolbar">' +
        '<label>线路</label><select id="fault-feeder"><option value="">全部</option>' + feeders.map((f) => '<option value="' + f.id + '">' + esc(f.name) + '</option>').join('') + '</select>' +
        '<label>状态</label><select id="fault-status"><option value="">全部</option>' +
          Object.keys(GridLabels.faultStatus).map((k) => '<option value="' + k + '">' + GridLabels.faultStatus[k] + '</option>').join('') +
        '</select>' +
        '<label><input type="checkbox" id="fault-long"> 仅长时停电</label>' +
        '<button class="btn btn-danger" id="btn-locate">⚡ 故障定位</button>' +
      '</div><div id="fault-list"></div>';

    function renderList() {
      const fid = document.getElementById('fault-feeder').value;
      const status = document.getElementById('fault-status').value;
      const longOnly = document.getElementById('fault-long').checked;
      let list = faults.slice();
      if (fid) list = list.filter((f) => f.feederId === fid);
      if (status) list = list.filter((f) => f.status === status);
      if (longOnly) list = list.filter((f) => f.longOutage);
      const box = document.getElementById('fault-list');
      box.innerHTML = '';
      if (!list.length) { box.innerHTML = '<div class="empty">无符合条件的故障事件</div>'; return; }
      list.forEach((f) => {
        const holder = document.createElement('div');
        GridComponents.FaultCard(holder, {
          fault: f,
          sections: sectionNames,
          onAction: async (action, faultId, extra) => {
            const operator = prompt('操作人：', '调度员');
            if (!operator) return;
            const body = { operator, note: '', ...extra };
            try {
              await GridAPI.post('/api/faults/' + faultId + '/' + action, body);
              faults = await GridAPI.get('/api/faults');
              renderList();
            } catch (e) { alert(action + ' 失败：' + e.message); }
          },
        });
        box.appendChild(holder);
      });
    }

    document.getElementById('fault-feeder').addEventListener('change', renderList);
    document.getElementById('fault-status').addEventListener('change', renderList);
    document.getElementById('fault-long').addEventListener('change', renderList);
    document.getElementById('btn-locate').addEventListener('click', async () => {
      if (!feeders.length) { alert('请先创建线路'); return; }
      const fid = prompt('选择线路 ID 执行定位（可用线路：' + feeders.map((f) => f.id + ':' + f.name).join('，') + '）', feeders[0].id);
      if (!fid) return;
      const operator = prompt('操作人：', '调度员');
      try {
        const data = await GridAPI.post('/api/faults/locate', { feederId: fid, operator });
        const loc = data.locate;
        let msg = '定位成功！主候选区段：' + (sectionNames[loc.primarySectionId] || loc.primarySectionId) + '\n\n候选：' +
          (loc.candidates || []).map((c) => sectionNames[c.sectionId] || c.sectionId).join('、') +
          '\n可疑指示器：' + (loc.suspicious || []).length + ' 个';
        alert(msg);
        faults = await GridAPI.get('/api/faults');
        renderList();
      } catch (e) { alert('定位失败：' + e.message); }
    });

    renderList();
  }

  // ---------- 页面：停电统计 ----------
  async function outagesPage() {
    app.innerHTML = '<div class="loading">加载停电统计…</div>';
    let summary;
    try { summary = await GridAPI.get('/api/outages/summary'); } catch (e) { summary = { totalRecords: 0 }; }
    let records = [];
    try { records = await GridAPI.get('/api/outages'); } catch (e) { /* ignore */ }

    app.innerHTML =
      '<h2>停电统计</h2>' +
      '<div class="stat-row">' +
        stat('停电记录', summary.totalRecords) +
        stat('累计停电时长', summary.totalMinutes + ' 分钟') +
        stat('平均停电时长', (summary.avgMinutes || 0).toFixed(1) + ' 分钟') +
        stat('最长停电', summary.maxMinutes + ' 分钟') +
        stat('长时停电次数', summary.longOutageCount, summary.longOutageCount > 0 ? 'red' : '') +
      '</div>' +
      (summary.byFeeder && summary.byFeeder.length
        ? '<section><h3>按线路统计</h3><table class="table"><thead><tr><th>线路</th><th>记录数</th><th>累计时长(分)</th><th>长时停电</th></tr></thead><tbody>' +
          summary.byFeeder.map((b) => '<tr><td>' + esc(b.feederName || b.feederId) + '</td><td>' + b.recordCount + '</td><td>' + b.totalMinutes + '</td><td>' + b.longOutages + '</td></tr>').join('') +
          '</tbody></table></section>' : '') +
      '<section><h3>停电记录明细</h3><div id="outage-table"></div></section>';

    const rows = records.map((r) =>
      '<tr><td>' + esc(r.id) + '</td><td>' + esc(r.feederName || r.feederId) + '</td>' +
      '<td>' + fmtDT(r.outageStart) + '</td><td>' + fmtDT(r.outageEnd) + '</td>' +
      '<td>' + r.durationMinutes + ' 分钟</td>' +
      '<td>' + (r.longOutage ? '<span class="badge badge-red">长时</span>' : '<span class="muted">正常</span>') + '</td></tr>').join('');
    document.getElementById('outage-table').innerHTML = records.length
      ? '<table class="table"><thead><tr><th>记录</th><th>线路</th><th>停电开始</th><th>复电结束</th><th>时长</th><th>类型</th></tr></thead><tbody>' + rows + '</tbody></table>'
      : '<div class="empty">暂无停电记录（完成故障复电后生成）</div>';
  }

  function stat(label, value, cls) {
    return '<div class="stat"><span class="stat-label">' + label + '</span><span class="stat-value ' + (cls || '') + '">' + value + '</span></div>';
  }

  // ---------- 路由 ----------
  const routes = {
    '': overviewPage,
    overview: overviewPage,
    topology: topologyPage,
    indicators: indicatorsPage,
    faults: faultsPage,
    outages: outagesPage,
  };

  async function route() {
    const hash = (location.hash || '#/').replace(/^#\//, '').replace(/^#/, '');
    const key = hash.split('?')[0] || 'overview';
    document.querySelectorAll('.nav a').forEach((a) => {
      a.classList.toggle('active', a.getAttribute('data-route') === key);
    });
    const page = routes[key] || overviewPage;
    try {
      await page();
    } catch (e) {
      app.innerHTML = '<div class="error-box">页面渲染失败：' + esc(e.message) + '</div>';
    }
  }

  window.addEventListener('hashchange', route);
  route();
})();
