/**
 * IndicatorTable：指示器信号表格组件。
 * 被「指示器」页与「故障定位弹窗」共用。
 * 用法：
 *   GridComponents.IndicatorTable(container, {
 *     indicators,
 *     sectionNames: {id:name},
 *     feederNames: {id:name},
 *     onSignal: (indicator, status) => {},
 *     onSuspicious: (indicator, suspicious) => {},
 *     compact: false
 *   });
 */
window.GridComponents = window.GridComponents || {};

window.GridComponents.IndicatorTable = function (container, opts) {
  opts = opts || {};
  const list = opts.indicators || [];
  const sectionNames = opts.sectionNames || {};
  const feederNames = opts.feederNames || {};
  const fmt = (t) => (t ? new Date(t).toLocaleString('zh-CN', { hour12: false }) : '—');

  if (!list.length) {
    container.innerHTML = '<div class="empty">暂无指示器数据</div>';
    return;
  }

  let html = '<table class="table"><thead><tr>' +
    '<th>名称</th><th>线路</th><th>区段</th><th>状态</th><th>上报时间</th><th>可疑</th>' +
    (opts.compact ? '' : '<th>操作</th>') + '</tr></thead><tbody>';

  list.forEach((ind) => {
    const trig = ind.status === GridEnums.IndicatorStatus.TRIGGERED;
    html += '<tr data-ind="' + ind.id + '" class="' + (ind.suspicious ? 'row-suspicious' : '') + '">' +
      '<td><strong>' + esc(ind.name) + '</strong></td>' +
      '<td>' + esc(feederNames[ind.feederId] || ind.feederId) + '</td>' +
      '<td>' + esc(sectionNames[ind.sectionId] || ind.sectionId) + '</td>' +
      '<td><span class="badge ' + (trig ? 'badge-red' : 'badge-green') + '">' + GridLabels.indicatorStatus[ind.status] + '</span></td>' +
      '<td>' + fmt(ind.reportedAt) + '</td>' +
      '<td>' + (ind.suspicious ? '<span class="badge badge-orange" title="' + esc(ind.suspiciousReason || '') + '">可疑</span>' : '<span class="muted">—</span>') + '</td>';
    if (!opts.compact) {
      html += '<td class="ops">' +
        '<button class="btn btn-sm ' + (trig ? 'btn-ghost' : 'btn-danger') + '" data-act="signal" data-status="triggered">翻牌</button>' +
        '<button class="btn btn-sm ' + (!trig ? 'btn-ghost' : 'btn-primary') + '" data-act="signal" data-status="reset">复位</button>' +
        '<button class="btn btn-sm ' + (ind.suspicious ? 'btn-ghost' : 'btn-warn') + '" data-act="sus">' + (ind.suspicious ? '解除可疑' : '标可疑') + '</button>' +
        '</td>';
    }
    html += '</tr>';
  });
  html += '</tbody></table>';
  container.innerHTML = html;

  container.querySelectorAll('[data-act]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const row = btn.closest('tr');
      if (!row) return;
      const id = row.getAttribute('data-ind');
      const ind = list.find((i) => i.id === id);
      if (!ind) return;
      const act = btn.getAttribute('data-act');
      if (act === 'signal') {
        if (opts.onSignal) opts.onSignal(ind, btn.getAttribute('data-status'));
      } else if (act === 'sus') {
        if (opts.onSuspicious) opts.onSuspicious(ind, !ind.suspicious);
      }
    });
  });
};

function esc(s) {
  return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
