// 视图模块：微信服务号管理 (Accounts)
import { fetchAPI, API_BASE } from '../api.js';
import { escapeHtml, copyText, pushToast, openModal, closeModal } from '../utils.js';

export async function loadAccounts() {
  const tbody = document.getElementById('accountsTable');
  if (!tbody) return;
  tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 20px;">正在加载...</td></tr>';

  try {
    const res = await fetchAPI('/admin/accounts');
    const list = (res && res.data && res.data.items) || [];
    if (list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--text-muted); padding: 24px;">暂无微信服务号配置，请点击右上角新增</td></tr>';
      return;
    }

    tbody.innerHTML = list.map(a => `
      <tr>
        <td><span class="mono-tag">${escapeHtml(a.id)}</span></td>
        <td><strong style="color: var(--text-primary);">${escapeHtml(a.name)}</strong></td>
        <td><span class="mono-tag" style="color: var(--accent-blue);">${escapeHtml(a.accountKey)}</span></td>
        <td>${escapeHtml(a.appId)}</td>
        <td><span class="status-pill ${a.tokenSet ? 'status-pill-success' : 'status-pill-muted'}">${a.tokenSet ? '已配置' : '未设置'}</span></td>
        <td><span class="status-pill ${a.encodingAesKeySet ? 'status-pill-success' : 'status-pill-muted'}">${a.encodingAesKeySet ? '已配置' : '未配置'}</span></td>
        <td style="color: var(--text-muted); font-size: 11px;">${escapeHtml(a.createdAt)}</td>
        <td style="text-align: right;">
          <button class="btn btn-sm" data-action="copy-url" data-key="${escapeHtml(a.accountKey)}">复制 URL</button>
          <button class="btn btn-danger-subtle btn-sm" data-action="delete-account" data-id="${escapeHtml(a.id)}">删除</button>
        </td>
      </tr>
    `).join('');
  } catch (e) {
    tbody.innerHTML = '<tr><td colspan="8" style="text-align: center; color: var(--accent-red); padding: 20px;">加载失败，请检查管理员口令</td></tr>';
  }
}

export function copyCallbackUrl(key) {
  const u = `${API_BASE}/w/${key}`;
  copyText(u);
}

export async function deleteAccount(id) {
  if (!confirm('确定删除该服务号？其下关联的全部订阅与历史事件将被级联清理。')) return;
  try {
    await fetchAPI(`/admin/accounts/${id}`, { method: 'DELETE' });
    pushToast('服务号已清理', 'success');
    loadAccounts();
  } catch (e) {}
}

export function openCreateAccountModal() {
  document.getElementById('accFormName').value = '';
  document.getElementById('accFormKey').value = '';
  document.getElementById('accFormAppId').value = '';
  document.getElementById('accFormToken').value = '';
  document.getElementById('accFormAesKey').value = '';
  openModal('accountModal');
}

export async function submitAccountForm() {
  const name = document.getElementById('accFormName').value.trim();
  const accountKey = document.getElementById('accFormKey').value.trim();
  const appId = document.getElementById('accFormAppId').value.trim();
  const token = document.getElementById('accFormToken').value.trim();
  const encodingAesKey = document.getElementById('accFormAesKey').value.trim();

  if (!name || !accountKey || !appId || !token) {
    pushToast('请完整填写必填项 (服务号名称, 路由标识, AppID, Token)', 'error');
    return;
  }

  try {
    await fetchAPI('/admin/accounts', {
      method: 'POST',
      body: JSON.stringify({ name, accountKey, appId, token, encodingAesKey })
    });
    pushToast('服务号创建成功', 'success');
    closeModal('accountModal');
    loadAccounts();
  } catch (e) {}
}
