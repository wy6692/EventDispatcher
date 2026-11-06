// 视图模块：微信服务号管理 (Accounts)
import { fetchAPI, API_BASE } from '../api.js';
import { escapeHtml, copyText, pushToast, openModal, closeModal, confirmAction } from '../utils.js';

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
        <td style="text-align: right; white-space: nowrap;">
          <button class="btn btn-sm" data-action="copy-url" data-key="${escapeHtml(a.accountKey)}" title="复制该服务号回调 URL">复制 URL</button>
          <button class="btn btn-sm" data-action="edit-account" data-id="${escapeHtml(a.id)}">编辑</button>
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
  const ok = await confirmAction('确定删除该服务号？', '其下关联的全部下游订阅与历史事件将被级联清理。');
  if (!ok) return;
  try {
    await fetchAPI(`/admin/accounts/${id}`, { method: 'DELETE' });
    pushToast('服务号已清理', 'success');
    loadAccounts();
  } catch (e) {}
}

export function openCreateAccountModal() {
  document.getElementById('accFormId').value = '';
  const row = document.getElementById('accFormIdRow');
  if (row) row.style.display = 'none';

  const title = document.getElementById('accountModalTitle');
  if (title) title.innerText = '新增微信服务号';

  const btn = document.getElementById('btnSubmitAccount');
  if (btn) btn.innerText = '保存服务号';

  const tokenHint = document.getElementById('accFormTokenHint');
  if (tokenHint) tokenHint.innerText = '(新增必填)';

  const aesHint = document.getElementById('accFormAesKeyHint');
  if (aesHint) aesHint.innerText = '(43位字符，留空为明文模式)';

  document.getElementById('accFormName').value = '';
  document.getElementById('accFormKey').value = '';
  document.getElementById('accFormAppId').value = '';
  const tokenInput = document.getElementById('accFormToken');
  tokenInput.value = '';
  tokenInput.placeholder = '微信后台配置的 Token';

  const aesInput = document.getElementById('accFormAesKey');
  aesInput.value = '';
  aesInput.placeholder = '43位字符，如留空则代表明文模式';

  openModal('accountModal');
}

export async function openEditAccountModal(id) {
  try {
    const res = await fetchAPI(`/admin/accounts/${id}`);
    const a = res && res.data;
    if (!a) {
      pushToast('获取服务号详情失败', 'error');
      return;
    }

    document.getElementById('accFormId').value = a.id;
    const row = document.getElementById('accFormIdRow');
    if (row) row.style.display = 'flex';
    const display = document.getElementById('accFormIdDisplay');
    if (display) display.value = a.id;

    const title = document.getElementById('accountModalTitle');
    if (title) title.innerText = '编辑微信服务号';

    const btn = document.getElementById('btnSubmitAccount');
    if (btn) btn.innerText = '保存修改';

    const tokenHint = document.getElementById('accFormTokenHint');
    if (tokenHint) tokenHint.innerText = '(留空表示不修改已有 Token)';

    const aesHint = document.getElementById('accFormAesKeyHint');
    if (aesHint) aesHint.innerText = '(留空表示不修改已有密钥)';

    document.getElementById('accFormName').value = a.name || '';
    document.getElementById('accFormKey').value = a.accountKey || '';
    document.getElementById('accFormAppId').value = a.appId || '';

    const tokenInput = document.getElementById('accFormToken');
    tokenInput.value = '';
    tokenInput.placeholder = a.tokenSet ? '【已配置】留空则保持不变' : '请输入微信 Token';

    const aesInput = document.getElementById('accFormAesKey');
    aesInput.value = '';
    aesInput.placeholder = a.encodingAesKeySet ? '【已配置】留空则保持不变' : '43位字符，留空为明文模式';

    openModal('accountModal');
  } catch (e) {
    pushToast('无法加载服务号配置: ' + (e.message || e), 'error');
  }
}

export async function submitAccountForm() {
  const id = document.getElementById('accFormId').value.trim();
  const name = document.getElementById('accFormName').value.trim();
  const accountKey = document.getElementById('accFormKey').value.trim();
  const appId = document.getElementById('accFormAppId').value.trim();
  const token = document.getElementById('accFormToken').value.trim();
  const encodingAesKey = document.getElementById('accFormAesKey').value.trim();

  if (!id) {
    // 新增模式
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
  } else {
    // 编辑模式 (PUT)
    if (!name || !accountKey || !appId) {
      pushToast('服务号名称、路由标识与 AppID 不能为空', 'error');
      return;
    }

    const payload = { name, accountKey, appId };
    if (token) payload.token = token;
    if (encodingAesKey) payload.encodingAesKey = encodingAesKey;

    try {
      await fetchAPI(`/admin/accounts/${id}`, {
        method: 'PUT',
        body: JSON.stringify(payload)
      });
      pushToast('服务号配置已更新', 'success');
      closeModal('accountModal');
      loadAccounts();
    } catch (e) {}
  }
}
