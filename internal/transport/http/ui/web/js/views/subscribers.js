import { fetchAPI } from '../api.js';
import { escapeHtml, pushToast, openModal, closeModal } from '../utils.js';

export async function loadSubscribers() {
  const tbody = document.getElementById('subscribersTable');
  if (!tbody) return;
  tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--text-muted);">正在加载...</td></tr>';

  try {
    const res = await fetchAPI('/admin/subscribers');
    const list = (res && res.data && res.data.items) || [];
    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--text-muted); padding: 24px;">暂无下游订阅方</td></tr>';
      return;
    }
    tbody.innerHTML = list.map(s => `
      <tr>
        <td>
          <strong style="color: var(--text-primary);">${escapeHtml(s.name)}</strong>
          <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); margin-top: 2px;">${escapeHtml(s.id)}</div>
        </td>
        <td><span class="mono-tag">${escapeHtml(s.accountId)}</span></td>
        <td style="word-break: break-all;"><a href="${escapeHtml(s.callbackUrl)}" target="_blank" rel="noopener noreferrer" style="color: var(--accent-blue); text-decoration: none;">${escapeHtml(s.callbackUrl)}</a></td>
        <td><span class="status-pill ${s.enabled ? 'status-pill-success' : 'status-pill-muted'}">${s.enabled ? '已启用' : '已停用'}</span></td>
        <td><span style="font-family: var(--font-mono);">${s.timeoutMs}ms / ${s.maxAttempts}次</span></td>
        <td><pre style="font-size: 10px; max-width: 260px; overflow: hidden; text-overflow: ellipsis; background: #090a0f; padding: 4px; border-radius: 3px; border: 1px solid var(--border-subtle);">${escapeHtml(JSON.stringify(s.filter || {}))}</pre></td>
        <td style="text-align: right;">
          <button class="btn btn-danger-subtle btn-sm" data-action="delete-subscriber" data-id="${escapeHtml(s.id)}">删除</button>
        </td>
      </tr>
    `).join('');
  } catch (e) {
    tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--accent-red);">加载失败</td></tr>';
  }
}

export async function deleteSubscriber(id) {
  if (!confirm('确定移除该下游订阅方？')) return;
  try {
    await fetchAPI('/admin/subscribers/' + id, { method: 'DELETE' });
    pushToast('订阅方已移除', 'success');
    loadSubscribers();
  } catch (e) {}
}

export function openCreateSubscriberModal() {
  document.getElementById('subFormName').value = '';
  document.getElementById('subFormAccountId').value = '';
  document.getElementById('subFormCallbackUrl').value = '';
  document.getElementById('subFormSecret').value = '';
  document.getElementById('subFormTimeout').value = '3000';
  document.getElementById('subFormMaxAttempts').value = '6';
  document.getElementById('subFormFilter').value = '{}';
  openModal('subscriberModal');
}

export async function submitSubscriberForm() {
  const name = document.getElementById('subFormName').value.trim();
  const accountId = document.getElementById('subFormAccountId').value.trim();
  const callbackUrl = document.getElementById('subFormCallbackUrl').value.trim();
  const secret = document.getElementById('subFormSecret').value.trim();
  const timeoutMs = parseInt(document.getElementById('subFormTimeout').value, 10);
  const maxAttempts = parseInt(document.getElementById('subFormMaxAttempts').value, 10);
  let filter = {};

  try {
    const raw = document.getElementById('subFormFilter').value.trim();
    if (raw) filter = JSON.parse(raw);
  } catch (e) {
    pushToast('Filter JSON 格式错误', 'error');
    return;
  }

  if (!name || !accountId || !callbackUrl || !secret) {
    pushToast('请完整填写必填项', 'error');
    return;
  }

  try {
    await fetchAPI('/admin/subscribers', {
      method: 'POST',
      body: JSON.stringify({ name, accountId, callbackUrl, secret, timeoutMs, maxAttempts, filter })
    });
    pushToast('订阅方已配置', 'success');
    closeModal('subscriberModal');
    loadSubscribers();
  } catch (e) {}
}
