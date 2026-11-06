import { fetchAPI } from '../api.js';
import { escapeHtml, openModal } from '../utils.js';

export async function loadEvents() {
  const tbody = document.getElementById('eventsTable');
  if (!tbody) return;
  tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted);">正在加载...</td></tr>';

  try {
    const res = await fetchAPI('/admin/events?limit=50');
    const list = (res && res.data && res.data.items) || [];
    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">暂无微信事件日志</td></tr>';
      return;
    }
    tbody.innerHTML = list.map(e => `
      <tr>
        <td><span class="mono-tag">${escapeHtml(e.id)}</span></td>
        <td><span class="mono-tag">${escapeHtml(e.accountId)}</span></td>
        <td><span class="status-pill status-pill-muted">${escapeHtml(e.msgType || '-')}</span></td>
        <td><span class="mono-tag" style="color: var(--accent-blue);">${escapeHtml(e.event || '-')}</span></td>
        <td style="max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${escapeHtml(e.eventKey || '-')}">
          <span style="font-family: var(--font-mono);">${escapeHtml(e.eventKey || '-')}</span>
        </td>
        <td style="font-size: 11px; font-family: var(--font-mono); max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;" title="${escapeHtml(e.fromUser || '-')}">
          ${escapeHtml(e.fromUser || '-')}
        </td>
        <td style="color: var(--text-muted); font-size: 11px; white-space: nowrap;">${escapeHtml(e.receivedAt)}</td>
        <td style="text-align: right; white-space: nowrap;">
          <button class="btn btn-sm" data-action="view-payload" data-payload='${escapeHtml(JSON.stringify(e.payload || {}))}'>查看载荷</button>
        </td>
      </tr>
    `).join('');
  } catch (e) {
    tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--accent-red);">加载失败</td></tr>';
  }
}

export function viewPayload(payloadObj) {
  const pre = document.getElementById('payloadViewText');
  if (pre) {
    pre.innerText = JSON.stringify(payloadObj, null, 2);
  }
  openModal('payloadModal');
}
