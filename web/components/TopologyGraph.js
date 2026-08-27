/**
 * TopologyGraph：SVG 拓扑图组件。
 * 被「拓扑管理」与「故障事件」页共用（故障页高亮候选故障区段）。
 * 用法：
 *   GridComponents.TopologyGraph(container, {
 *     topology,               // GET /api/feeders/{id}/topology 返回
 *     highlightSectionId,     // 可选，高亮候选故障区段
 *     interactive: true,      // 开关/区段可点击
 *     onSwitchClick, onSectionClick
 *   });
 */
window.GridComponents = window.GridComponents || {};

window.GridComponents.TopologyGraph = function (container, opts) {
  opts = opts || {};
  const tp = opts.topology;
  if (!tp || !tp.switches) { container.innerHTML = '<div class="empty">暂无拓扑数据</div>'; return; }

  const W = 900, H = 360;
  const switches = tp.switches.slice().sort((a, b) => a.order - b.order || (a.id < b.id ? -1 : 1));
  const sections = tp.sections || [];
  const indicators = tp.indicators || [];

  // 布局：从出线开关 BFS，按深度排 x、分支排 y。
  const outlet = switches.find((s) => s.switchType === GridEnums.SwitchType.FEEDER_OUTLET);
  const pos = {};
  if (outlet) {
    const adj = {};
    sections.forEach((sec) => {
      (adj[sec.upstreamSwitchId] = adj[sec.upstreamSwitchId] || []).push(sec);
    });
    const queue = [{ sw: outlet, depth: 0, branch: 0 }];
    const seen = new Set([outlet.id]);
    const usedBranch = new Map();
    usedBranch.set(0, 1);
    while (queue.length) {
      const cur = queue.shift();
      pos[cur.sw.id] = { x: 70 + cur.depth * 150, y: 70 + cur.branch * 75 };
      const children = adj[cur.sw.id] || [];
      children.forEach((sec, idx) => {
        if (seen.has(sec.downstreamSwitchId)) return;
        seen.add(sec.downstreamSwitchId);
        const branch = cur.branch + (idx > 0 ? idx : 0);
        queue.push({ sw: switches.find((s) => s.id === sec.downstreamSwitchId) || { id: sec.downstreamSwitchId }, depth: cur.depth + 1, branch });
      });
    }
  }
  // 未布局到的开关（无出线开关等）排到右侧兜底。
  let fallback = 0;
  switches.forEach((sw) => {
    if (!pos[sw.id]) {
      pos[sw.id] = { x: 70 + (switches.length + fallback) * 150, y: 70 };
      fallback++;
    }
  });

  const sectionById = {};
  sections.forEach((sec) => { sectionById[sec.id] = sec; });

  let svg = '<svg viewBox="0 0 ' + W + ' ' + H + '" width="100%" height="' + H + '" class="topology-svg">';
  // 网格背景
  svg += '<defs><marker id="arrow" markerWidth="8" markerHeight="6" refX="8" refY="3" orient="auto"><path d="M0,0 L8,3 L0,6 Z" fill="#888"/></marker></defs>';
  svg += '<rect x="0" y="0" width="' + W + '" height="' + H + '" fill="#fbfcfe" rx="8"/>';

  // 区段连线
  sections.forEach((sec) => {
    const p1 = pos[sec.upstreamSwitchId] || { x: 40, y: 40 };
    const p2 = pos[sec.downstreamSwitchId] || { x: W - 40, y: H - 40 };
    let color = '#9aa7b5';
    let stroke = 3;
    let dash = '';
    let glow = '';
    if (sec.isolated) { color = '#e5484d'; dash = 'stroke-dasharray="8 5"'; }
    else if (sec.isCandidate || opts.highlightSectionId === sec.id) { color = '#f59f00'; stroke = 6; glow = 'filter: drop-shadow(0 0 4px rgba(245,159,0,.9));'; }
    svg += '<line x1="' + p1.x + '" y1="' + p1.y + '" x2="' + p2.x + '" y2="' + p2.y +
      '" stroke="' + color + '" stroke-width="' + stroke + '" ' + dash +
      ' style="' + glow + '" class="section-line" data-id="' + sec.id + '"/>';
    // 中点标签
    const mx = (p1.x + p2.x) / 2, my = (p1.y + p2.y) / 2;
    svg += '<text x="' + mx + '" y="' + (my - 8) + '" text-anchor="middle" class="section-label">' + esc(sec.name) + '</text>';
    // 指示器
    indicators.filter((i) => i.sectionId === sec.id).forEach((ind, idx) => {
      const ix = mx + (idx - 0.5) * 30;
      const iy = my + 16;
      const trig = ind.status === GridEnums.IndicatorStatus.TRIGGERED;
      const col = trig ? '#e5484d' : (ind.suspicious ? '#f59f00' : '#2f9e44');
      svg += '<polygon points="' + (ix - 7) + ',' + (iy + 8) + ' ' + (ix + 7) + ',' + (iy + 8) + ' ' + ix + ',' + iy + '" fill="' + col + '" class="indicator-mark" title="' + esc(ind.name) + '"/>';
      svg += '<text x="' + ix + '" y="' + (iy + 20) + '" text-anchor="middle" class="indicator-label">' + esc(ind.name) + (trig ? '▲' : '') + '</text>';
    });
  });

  // 开关节点
  switches.forEach((sw) => {
    const p = pos[sw.id];
    const closed = sw.status === GridEnums.SwitchStatus.CLOSED;
    const fill = closed ? '#2f9e44' : '#e5484d';
    let shape = '';
    const r = 16;
    if (sw.switchType === GridEnums.SwitchType.FEEDER_OUTLET) {
      shape = '<rect x="' + (p.x - r) + '" y="' + (p.y - r) + '" width="' + (2 * r) + '" height="' + (2 * r) + '" rx="4" fill="' + fill + '" stroke="#fff" stroke-width="2"/>';
    } else if (sw.switchType === GridEnums.SwitchType.TIE) {
      shape = '<polygon points="' + p.x + ',' + (p.y - r) + ' ' + (p.x + r) + ',' + p.y + ' ' + p.x + ',' + (p.y + r) + ' ' + (p.x - r) + ',' + p.y + '" fill="' + fill + '" stroke="#fff" stroke-width="2"/>';
    } else {
      shape = '<circle cx="' + p.x + '" cy="' + p.y + '" r="' + r + '" fill="' + fill + '" stroke="#fff" stroke-width="2"/>';
    }
    svg += '<g class="switch-node" data-id="' + sw.id + '">' + shape +
      '<text x="' + p.x + '" y="' + p.y + '" text-anchor="middle" dominant-baseline="central" fill="#fff" font-size="13" font-weight="bold">' + (closed ? '合' : '分') + '</text>' +
      '<text x="' + p.x + '" y="' + (p.y + r + 16) + '" text-anchor="middle" class="switch-label">' + esc(sw.name) + '（' + GridLabels.switchType[sw.switchType] + '）</text></g>';
  });

  svg += '</svg>';

  // 图例
  svg = '<div class="topology-wrap">' + svg + '<div class="legend">' +
    '<span><i class="dot" style="background:#2f9e44"></i>合闸</span>' +
    '<span><i class="dot" style="background:#e5484d"></i>分闸</span>' +
    '<span><i class="line-s" style="background:#f59f00"></i>候选故障</span>' +
    '<span><i class="line-s dashed" style="background:#e5484d"></i>已隔离</span>' +
    '<span><i class="tri" style="border-bottom-color:#e5484d"></i>翻牌指示器</span>' +
    '</div></div>';

  container.innerHTML = svg;

  if (opts.interactive) {
    container.querySelectorAll('.switch-node').forEach((g) => {
      g.style.cursor = 'pointer';
      g.addEventListener('click', () => {
        const id = g.getAttribute('data-id');
        const sw = switches.find((s) => s.id === id);
        if (sw && opts.onSwitchClick) opts.onSwitchClick(sw);
      });
    });
    container.querySelectorAll('.section-line').forEach((line) => {
      line.style.cursor = 'pointer';
      line.addEventListener('click', () => {
        const id = line.getAttribute('data-id');
        const sec = sectionById[id];
        if (sec && opts.onSectionClick) opts.onSectionClick(sec);
      });
    });
  }
};

function esc(s) {
  return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
