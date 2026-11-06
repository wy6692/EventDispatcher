// 视图模块：投递调度、重试与错误详情分析 (Deliveries)
import { fetchAPI } from '../api.js';
import { escapeHtml, copyText, pushToast, openModal, closeModal } from '../utils.js';

let currentViewingDeliveryId = null;

export async function loadDeliveries() {
  const tbody = document.getElementById('deliveriesTable');
  const filterEl = document.getElementById('deliveryStatusFilter');
  if (!tbody) return;

  const status = filterEl ? filterEl.value : '';
  tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 20px;">正在加载...</td></tr>';

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
      let statusLabel = d.status;
      if (d.status === 'success') {
        pill = 'status-pill-success';
        statusLabel = '成功 (success)';
      } else if (d.status === 'failed') {
        pill = 'status-pill-failed';
        statusLabel = '失败 (failed)';
      } else if (d.status === 'pending') {
        pill = 'status-pill-pending';
        statusLabel = '重试中 (pending)';
      } else if (d.status === 'inflight') {
        pill = 'status-pill-pending';
        statusLabel = '投递中 (inflight)';
      }

      const errorSnippet = d.lastError
        ? `<div style="cursor: pointer; color: var(--accent-red); font-size: 11px; max-width: 260px; word-break: break-all; text-decoration: underline;"
                data-action="view-delivery"
                data-id="${escapeHtml(d.id)}"
                title="点击查看完整错误信息与载荷">
             ${escapeHtml(d.lastError.length > 38 ? d.lastError.slice(0, 38) + '...' : d.lastError)}
           </div>`
        : `<span style="font-size: 11px; color: var(--text-muted);">${escapeHtml(d.nextRetryAt || '-')}</span>`;

      return `
        <tr>
          <td>
            <span class="mono-tag" style="cursor: pointer;" data-action="view-delivery" data-id="${escapeHtml(d.id)}" title="点击查看投递详情">
              ${escapeHtml(d.id)}
            </span>
          </td>
          <td><span class="mono-tag">${escapeHtml(d.eventId)}</span></td>
          <td><strong style="color: var(--text-primary);">${escapeHtml(d.subscriberName || d.subscriberId)}</strong></td>
          <td><span class="status-pill ${pill}">${escapeHtml(statusLabel)}</span></td>
          <td style="font-family: var(--font-mono);">${d.attempt} 轮</td>
          <td>${d.lastStatusCode ? `<span class="mono-tag ${d.lastStatusCode >= 200 && d.lastStatusCode < 300 ? 'status-pill-success' : 'status-pill-failed'}">HTTP ${d.lastStatusCode}</span>` : '<span style="color: var(--text-muted);">-</span>'}</td>
          <td>${errorSnippet}</td>
          <td style="text-align: right; white-space: nowrap;">
            <button class="btn btn-sm" data-action="view-delivery" data-id="${escapeHtml(d.id)}" title="查看详细错误信息与事件载荷">详情</button>
            <button class="btn btn-sm" data-action="retry-delivery" data-id="${escapeHtml(d.id)}">重放</button>
          </td>
        </tr>
      `;
    }).join('');
  } catch (e) {
    tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--accent-red); padding: 20px;">加载失败</td></tr>';
  }
}

export async function retryDelivery(id) {
  try {
    await fetchAPI('/admin/deliveries/' + id + '/retry', { method: 'POST' });
    pushToast('已重置任务，Worker 正在调度重试', 'success');
    loadDeliveries();
    // 如果当前详情弹窗正在打开，同步刷新弹窗内容
    if (currentViewingDeliveryId === id) {
      openDeliveryDetail(id);
    }
  } catch (e) {}
}

export async function openDeliveryDetail(id) {
  currentViewingDeliveryId = id;
  try {
    const res = await fetchAPI('/admin/deliveries/' + id);
    const d = res && res.data;
    if (!d) {
      pushToast('获取投递详情失败', 'error');
      return;
    }

    document.getElementById('dlvDetailId').innerText = d.id;
    document.getElementById('dlvDetailEventId').innerText = d.eventId;
    document.getElementById('dlvDetailSubscriber').innerText = `${d.subscriberName || d.subscriberId} (${d.subscriberId})`;
    document.getElementById('dlvDetailUrl').innerText = d.callbackUrl || '-';

    let pill = 'status-pill-muted';
    let statusLabel = d.status;
    if (d.status === 'success') {
      pill = 'status-pill-success';
      statusLabel = '成功 (success)';
    } else if (d.status === 'failed') {
      pill = 'status-pill-failed';
      statusLabel = '最终失败 (failed)';
    } else if (d.status === 'pending') {
      pill = 'status-pill-pending';
      statusLabel = '待重试 (pending)';
    } else if (d.status === 'inflight') {
      pill = 'status-pill-pending';
      statusLabel = '投递中 (inflight)';
    }

    document.getElementById('dlvDetailStatus').innerHTML = `<span class="status-pill ${pill}">${escapeHtml(statusLabel)}</span>`;
    document.getElementById('dlvDetailAttempt').innerText = `${d.attempt} 轮次`;
    document.getElementById('dlvDetailStatusCode').innerHTML = d.lastStatusCode
      ? `<span class="mono-tag ${d.lastStatusCode >= 200 && d.lastStatusCode < 300 ? 'status-pill-success' : 'status-pill-failed'}">HTTP ${d.lastStatusCode}</span>`
      : '<span style="color: var(--text-muted); font-size: 11px;">未获取到 HTTP 状态码 (网络错误或未开始)</span>';

    document.getElementById('dlvDetailTime').innerText = `下次重试: ${d.nextRetryAt || '-'}  |  更新时间: ${d.updatedAt || '-'}`;

    // 错误信息展示模块
    const errorBlock = document.getElementById('dlvDetailErrorBlock');
    const errorText = document.getElementById('dlvDetailErrorText');
    const successBlock = document.getElementById('dlvDetailSuccessBlock');
    const pendingBlock = document.getElementById('dlvDetailPendingBlock');

    if (d.lastError) {
      errorBlock.style.display = 'block';
      errorText.innerText = d.lastError;
    } else {
      errorBlock.style.display = 'none';
      errorText.innerText = '';
    }

    if (d.status === 'success') {
      successBlock.style.display = 'block';
      pendingBlock.style.display = 'none';
    } else if (d.status === 'pending') {
      successBlock.style.display = 'none';
      pendingBlock.style.display = 'block';
    } else {
      successBlock.style.display = 'none';
      pendingBlock.style.display = 'none';
    }

    // 关联事件出站载荷预览
    const payloadPre = document.getElementById('dlvDetailPayloadText');
    payloadPre.innerText = '正在加载关联事件载荷...';

    // 异步加载原事件 JSON
    if (d.eventId) {
      fetchAPI('/admin/events/' + d.eventId).then(evtRes => {
        const evt = evtRes && evtRes.data;
        if (evt && evt.payload) {
          payloadPre.innerText = JSON.stringify(evt.payload, null, 2);
        } else {
          payloadPre.innerText = '(未找到对应的事件载荷)';
        }
      }).catch(err => {
        payloadPre.innerText = '加载事件失败: ' + (err.message || err);
      });
    }

    openModal('deliveryDetailModal');
  } catch (e) {
    pushToast('无法加载投递任务详情: ' + (e.message || e), 'error');
  }
}

// 绑定详情弹窗内部辅助按钮
export function initDeliveryDetailActions() {
  const btnRetry = document.getElementById('btnDlvModalRetry');
  if (btnRetry) {
    btnRetry.addEventListener('click', () => {
      if (currentViewingDeliveryId) {
        retryDelivery(currentViewingDeliveryId);
      }
    });
  }

  const btnCopyId = document.getElementById('btnCopyDlvId');
  if (btnCopyId) {
    btnCopyId.addEventListener('click', () => {
      const text = document.getElementById('dlvDetailId').innerText;
      if (text) copyText(text);
    });
  }

  const btnCopyEventId = document.getElementById('btnCopyDlvEventId');
  if (btnCopyEventId) {
    btnCopyEventId.addEventListener('click', () => {
      const text = document.getElementById('dlvDetailEventId').innerText;
      if (text) copyText(text);
    });
  }

  const btnCopyError = document.getElementById('btnCopyDlvError');
  if (btnCopyError) {
    btnCopyError.addEventListener('click', () => {
      const text = document.getElementById('dlvDetailErrorText').innerText;
      if (text) copyText(text);
    });
  }

  const btnCopyPayload = document.getElementById('btnCopyDlvPayload');
  if (btnCopyPayload) {
    btnCopyPayload.addEventListener('click', () => {
      const text = document.getElementById('dlvDetailPayloadText').innerText;
      if (text) copyText(text);
    });
  }
}
