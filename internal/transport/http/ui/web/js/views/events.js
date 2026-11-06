import { fetchAPI } from '../api.js';
import { escapeHtml, openModal, renderTableSkeleton, renderPagination } from '../utils.js';

let eventPage = 1;
let eventLimit = 15;

export async function populateEventAccountOptions() {
  const select = document.getElementById('eventFilterAccount');
  if (!select || select.dataset.loaded) return;
  try {
    const res = await fetchAPI('/admin/accounts');
    const items = (res && res.data && res.data.items) || [];
    const cur = select.value;
    select.innerHTML = '<option value="">全部服务号</option>' + items.map(a =>
      `<option value="${escapeHtml(a.id)}">${escapeHtml(a.name)} (${escapeHtml(a.accountKey)})</option>`
    ).join('');
    select.value = cur;
    select.dataset.loaded = 'true';
  } catch (e) {}
}

export async function loadEvents(page = eventPage, limit = eventLimit) {
  eventPage = page;
  eventLimit = limit;

  populateEventAccountOptions();

  const tbody = document.getElementById('eventsTable');
  if (!tbody) return;
  tbody.innerHTML = renderTableSkeleton(8, 4);

  const accountId = document.getElementById('eventFilterAccount')?.value.trim() || '';
  const msgType = document.getElementById('eventFilterMsgType')?.value.trim() || '';
  const event = document.getElementById('eventFilterEvent')?.value.trim() || '';
  const fromUser = document.getElementById('eventFilterFromUser')?.value.trim() || '';
  const eventId = document.getElementById('eventFilterEventId')?.value.trim() || '';

  try {
    let url = `/admin/events?page=${eventPage}&limit=${eventLimit}`;
    if (accountId) url += '&accountId=' + encodeURIComponent(accountId);
    if (msgType) url += '&msgType=' + encodeURIComponent(msgType);
    if (event) url += '&event=' + encodeURIComponent(event);
    if (fromUser) url += '&fromUser=' + encodeURIComponent(fromUser);
    if (eventId) url += '&eventId=' + encodeURIComponent(eventId);

    const res = await fetchAPI(url);
    const data = res && res.data;
    const list = (data && data.items) || [];
    const total = (data && typeof data.total === 'number') ? data.total : list.length;
    const totalPages = (data && data.totalPages) || Math.max(1, Math.ceil(total / eventLimit));

    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">暂无匹配的微信事件日志</td></tr>';
      renderPagination({
        containerId: 'eventsPagination',
        page: eventPage,
        limit: eventLimit,
        total: 0,
        totalPages: 1,
        onPageChange: (p) => loadEvents(p, eventLimit),
        onLimitChange: (l) => loadEvents(1, l)
      });
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

    renderPagination({
      containerId: 'eventsPagination',
      page: eventPage,
      limit: eventLimit,
      total,
      totalPages,
      onPageChange: (p) => loadEvents(p, eventLimit),
      onLimitChange: (l) => loadEvents(1, l)
    });
  } catch (e) {
    tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--accent-red);">加载失败</td></tr>';
  }
}

export function initEventFilters() {
  const btnSearch = document.getElementById('btnSearchEvents');
  if (btnSearch) {
    btnSearch.addEventListener('click', () => loadEvents(1));
  }

  const btnReset = document.getElementById('btnResetEvents');
  if (btnReset) {
    btnReset.addEventListener('click', () => {
      const acc = document.getElementById('eventFilterAccount');
      if (acc) acc.value = '';
      const msg = document.getElementById('eventFilterMsgType');
      if (msg) msg.value = '';
      const evt = document.getElementById('eventFilterEvent');
      if (evt) evt.value = '';
      const from = document.getElementById('eventFilterFromUser');
      if (from) from.value = '';
      const eid = document.getElementById('eventFilterEventId');
      if (eid) eid.value = '';
      loadEvents(1);
    });
  }

  ['eventFilterEvent', 'eventFilterFromUser', 'eventFilterEventId'].forEach(id => {
    const el = document.getElementById(id);
    if (el) {
      el.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          loadEvents(1);
        }
      });
    }
  });

  const accSelect = document.getElementById('eventFilterAccount');
  if (accSelect) accSelect.addEventListener('change', () => loadEvents(1));
  const msgSelect = document.getElementById('eventFilterMsgType');
  if (msgSelect) msgSelect.addEventListener('change', () => loadEvents(1));
}

export function viewPayload(payloadObj) {
  const pre = document.getElementById('payloadViewText');
  if (pre) {
    pre.innerText = JSON.stringify(payloadObj, null, 2);
  }
  openModal('payloadModal');
}
