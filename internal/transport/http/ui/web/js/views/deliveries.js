import { fetchAPI } from '../api.js';
import { escapeHtml, pushToast } from '../utils.js';

export async function loadDeliveries() {
  const tbody = document.getElementById('deliveriesTable');
  const filterEl = document.getElementById('deliveryStatusFilter');
  if (!tbody) return;

  const status = filterEl ? filterEl.value : '';
  tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted);">正在加载...</td></tr>';

  try {
    let url = '/admin/deliveries?limit=50';
    if (status) url += '&status=' + encodeURIComponent(status);
    const res = await fetchAPI(url);
    const list = (res && res.data && res.data.items) || [];
    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">暂无投递记录</td></tr>';
      return;
    }
    tbody.innerHTML = list.map(d => {
      let pill = 'status-pill-muted';
      if (d.status === 'success') pill = 'status-pill-success';
      else if (d.status === 'failed') pill = 'status-pill-failed';
      else if (d.status === 'pending') pill = 'status-pill-pending';

      return `
        <tr>
          <td><span class="mono-tag">${escapeHtml(d.id)}</span></td>
          <td><span class="mono-tag">${escapeHtml(d.eventId)}</span></td>
          <td><strong style="color: var(--text-primary);">${escapeHtml(d.subscriberName || d.subscriberId)}</strong></td>
          <td><span class="status-pill ${pill}">${escapeHtml(d.status)}</span></td>
          <td style="font-family: var(--font-mono);">${d.attempt} 轮</td>
          <td>${d.lastStatusCode ? `<span class="mono-tag ${d.lastStatusCode >= 200 && d.lastStatusCode < 300 ? 'status-pill-success' : 'status-pill-failed'}">HTTP ${d.lastStatusCode}</span>` : '-'}</td>
          <td style="font-size: 11px; color: ${d.status === 'failed' ? 'var(--accent-red)' : 'var(--text-muted)'}; max-width: 260px; word-break: break-all;">
            ${escapeHtml(d.lastError || d.nextRetryAt || '-')}
          </td>
          <td style="text-align: right;">
            <button class="btn btn-sm" data-action="retry-delivery" data-id="${escapeHtml(d.id)}">重放任务</button>
          </td>
        </tr>
      `;
    }).join('');
  } catch (e) {
    tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--accent-red);">加载失败</td></tr>';
  }
}

export async function retryDelivery(id) {
  try {
    await fetchAPI('/admin/deliveries/' + id + '/retry', { method: 'POST' });
    pushToast('已重置任务，Worker 正在调度重试', 'success');
    loadDeliveries();
  } catch (e) {}
}
