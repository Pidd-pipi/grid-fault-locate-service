/**
 * FaultCard：故障事件卡片组件。
 * 被「配网总览」与「故障事件」页共用。
 * 用法：
 *   GridComponents.FaultCard(container, {
 *     fault,
 *     sections: { id: name },   // 可选，隔离操作选择区段用
 *     onAction: (action, fault) => {},
 *     compact: false
 *   });
 */
window.GridComponents = window.GridComponents || {};

window.GridComponents.FaultCard = function (container, opts) {
  opts = opts || {};
  const f = opts.fault;
  if (!f) { container.innerHTML = ''; return; }
  const sections = opts.sections || {};
  const fmt = (t) => (t ? new Date(t).toLocaleString('zh-CN', { hour12: false }) : '—');
  const statusLabel = GridLabels.faultStatus[f.status] || f.status;

  const candidateNames = (f.candidateSectionIds || []).map((id) => sections[id] || id).join('、') || '—';

  let isolateSelect = '';
  if (f.status === GridEnums.FaultStatus.LOCATED || f.status === GridEnums.FaultStatus.REPAIRING) {
    const options = (f.candidateSectionIds && f.candidateSectionIds.length
      ? f.candidateSectionIds
      : [f.primarySectionId].filter(Boolean)
    ).map((id) => '<option value="' + id + '">' + (sections[id] || id) + '</option>').join('');
    isolateSelect = '<select class="isolate-select" data-fault="' + f.id + '">' + options + '</select>';
  }

  const actionBtn = (action, label, cls) =>
    '<button class="btn ' + cls + '" data-action="' + action + '" data-fault="' + f.id + '">' + label + '</button>';

  let actions = '';
  if (f.status === GridEnums.FaultStatus.LOCATED) {
    actions = actionBtn('repair', '开始抢修', 'btn-primary') + isolateSelect + actionBtn('isolate', '隔离确认', 'btn-warn');
  } else if (f.status === GridEnums.FaultStatus.REPAIRING) {
    actions = isolateSelect + actionBtn('isolate', '隔离确认', 'btn-warn') + actionBtn('restore', '复电完成', 'btn-primary');
  } else if (f.status === GridEnums.FaultStatus.RESTORED) {
    actions = actionBtn('archive', '归档', 'btn-ghost');
  }

  const longBadge = f.longOutage ? '<span class="badge badge-red">长时停电 ⚠</span>' : '';

  container.innerHTML =
    '<div class="card fault-card ' + (opts.compact ? 'compact' : '') + '" data-id="' + f.id + '">' +
      '<div class="card-head">' +
        '<span class="badge status-' + f.status + '">' + statusLabel + '</span>' +
        '<strong class="fault-title">' + esc(f.id) + ' · ' + esc(f.feederName || f.feederId) + '</strong>' +
        longBadge +
      '</div>' +
      '<div class="card-body">' +
        '<div class="kv"><span>主候选区段</span><b>' + esc(sections[f.primarySectionId] || f.primarySectionId || '—') + '</b></div>' +
        '<div class="kv"><span>候选区段</span><b>' + esc(candidateNames) + '</b></div>' +
        '<div class="kv"><span>定位时间</span><b>' + fmt(f.locatedAt) + '</b></div>' +
        (f.isolatedAt ? '<div class="kv"><span>隔离时间</span><b>' + fmt(f.isolatedAt) + ' · ' + esc(f.isolatedBy || '') + '</b></div>' : '') +
        (f.restoredAt ? '<div class="kv"><span>复电时间</span><b>' + fmt(f.restoredAt) + ' · ' + esc(f.restoredBy || '') + '</b></div>' : '') +
        '<div class="kv"><span>停电时长</span><b>' + (f.outageMinutes != null ? f.outageMinutes : calcMinutes(f)) + ' 分钟</b></div>' +
        '<div class="evidence" title="' + esc(f.evidence || '') + '">' + esc(f.evidence || '无定位依据') + '</div>' +
        (f.suspiciousIndicatorIds && f.suspiciousIndicatorIds.length
          ? '<div class="suspicious-hint">⚠ 可疑指示器 ' + f.suspiciousIndicatorIds.length + ' 个，需人工核验</div>' : '') +
      '</div>' +
      (actions ? '<div class="card-actions">' + actions + '</div>' : '') +
    '</div>';

  container.querySelectorAll('[data-action]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const action = btn.getAttribute('data-action');
      const faultId = btn.getAttribute('data-fault');
      let extra = {};
      if (action === 'isolate') {
        const sel = container.querySelector('.isolate-select[data-fault="' + faultId + '"]');
        extra.sectionId = sel ? sel.value : '';
      }
      if (opts.onAction) opts.onAction(action, faultId, extra);
    });
  });
};

function calcMinutes(f) {
  if (f.restoredAt && f.locatedAt) {
    return Math.max(0, Math.round((new Date(f.restoredAt) - new Date(f.locatedAt)) / 60000));
  }
  return 0;
}

function esc(s) {
  return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
