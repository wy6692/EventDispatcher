// 视图模块：下游订阅方管理与连通性测试 (Subscribers)
import { fetchAPI } from '../api.js';
import { escapeHtml, pushToast, openModal, closeModal, confirmAction, renderTableSkeleton, setButtonLoading } from '../utils.js';

let cachedSubscribers = [];

export async function populateSubscriberAccountFilter() {
  const select = document.getElementById('subscriberFilterAccount');
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

export function renderSubscriberTable(list) {
  const tbody = document.getElementById('subscribersTable');
  if (!tbody) return;

  if (list.length === 0) {
    tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--text-muted); padding: 24px;">暂无符合条件的下游订阅方</td></tr>';
    return;
  }

  tbody.innerHTML = list.map(s => `
    <tr>
      <td>
        <strong style="color: var(--text-primary);">${escapeHtml(s.name)}</strong>
        <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); margin-top: 2px;">${escapeHtml(s.id)}</div>
      </td>
      <td><span class="mono-tag">${escapeHtml(s.accountId)}</span></td>
      <td style="word-break: break-all;">
        <a href="${escapeHtml(s.callbackUrl)}" target="_blank" rel="noopener noreferrer" style="color: var(--accent-blue); text-decoration: none;">${escapeHtml(s.callbackUrl)}</a>
      </td>
      <td>
        <span class="status-pill ${s.enabled ? 'status-pill-success' : 'status-pill-muted'}"
              style="cursor: pointer;"
              data-action="toggle-subscriber-enabled"
              data-id="${escapeHtml(s.id)}"
              data-enabled="${s.enabled ? 'true' : 'false'}"
              title="点击快速切换启用/停用状态">
          ${s.enabled ? '已启用' : '已停用'}
        </span>
      </td>
      <td><span style="font-family: var(--font-mono);">${s.timeoutMs}ms / ${s.maxAttempts}次</span></td>
      <td><pre style="font-size: 10px; max-width: 260px; overflow: hidden; text-overflow: ellipsis; background: #090a0f; padding: 4px; border-radius: 3px; border: 1px solid var(--border-subtle);">${escapeHtml(JSON.stringify(s.filter || {}))}</pre></td>
      <td style="text-align: right; white-space: nowrap;">
        <button class="btn btn-sm" data-action="test-subscriber" data-id="${escapeHtml(s.id)}" title="发送测试事件验证下游连通性">测试连接</button>
        <button class="btn btn-sm" data-action="edit-subscriber" data-id="${escapeHtml(s.id)}">编辑</button>
        <button class="btn btn-danger-subtle btn-sm" data-action="delete-subscriber" data-id="${escapeHtml(s.id)}">删除</button>
      </td>
    </tr>
  `).join('');
}

export function applySubscriberFilters() {
  const acc = document.getElementById('subscriberFilterAccount')?.value || '';
  const status = document.getElementById('subscriberFilterStatus')?.value || '';
  const keyword = (document.getElementById('subscriberFilterKeyword')?.value || '').toLowerCase().trim();

  let filtered = cachedSubscribers;
  if (acc) {
    filtered = filtered.filter(s => s.accountId === acc);
  }
  if (status === 'enabled') {
    filtered = filtered.filter(s => s.enabled);
  } else if (status === 'disabled') {
    filtered = filtered.filter(s => !s.enabled);
  }
  if (keyword) {
    filtered = filtered.filter(s =>
      (s.name && s.name.toLowerCase().includes(keyword)) ||
      (s.id && s.id.toLowerCase().includes(keyword)) ||
      (s.callbackUrl && s.callbackUrl.toLowerCase().includes(keyword))
    );
  }

  renderSubscriberTable(filtered);
}

export async function loadSubscribers() {
  const tbody = document.getElementById('subscribersTable');
  if (!tbody) return;
  tbody.innerHTML = renderTableSkeleton(7, 3);

  populateSubscriberAccountFilter();

  try {
    const res = await fetchAPI('/admin/subscribers');
    cachedSubscribers = (res && res.data && res.data.items) || [];
    applySubscriberFilters();
  } catch (e) {
    tbody.innerHTML = '<tr><td colspan="7" style="text-align: center; color: var(--accent-red); padding: 20px;">加载失败</td></tr>';
  }
}

export function initSubscriberFilters() {
  const acc = document.getElementById('subscriberFilterAccount');
  if (acc) acc.addEventListener('change', applySubscriberFilters);

  const status = document.getElementById('subscriberFilterStatus');
  if (status) status.addEventListener('change', applySubscriberFilters);

  const kw = document.getElementById('subscriberFilterKeyword');
  if (kw) {
    kw.addEventListener('input', applySubscriberFilters);
  }

  const btnReset = document.getElementById('btnResetSubFilter');
  if (btnReset) {
    btnReset.addEventListener('click', () => {
      if (acc) acc.value = '';
      if (status) status.value = '';
      if (kw) kw.value = '';
      applySubscriberFilters();
    });
  }
}

export async function deleteSubscriber(id) {
  const ok = await confirmAction('确定移除该下游订阅方？', '其未完成的投递调度任务将一并清理。');
  if (!ok) return;
  try {
    await fetchAPI('/admin/subscribers/' + id, { method: 'DELETE' });
    pushToast('订阅方已移除', 'success');
    loadSubscribers();
  } catch (e) {}
}

export async function toggleSubscriberEnabled(id, currentEnabled) {
  try {
    await fetchAPI(`/admin/subscribers/${id}`, {
      method: 'PUT',
      body: JSON.stringify({ enabled: !currentEnabled })
    });
    pushToast(!currentEnabled ? '订阅方已启用' : '订阅方已停用', 'success');
    loadSubscribers();
  } catch (e) {
    pushToast('切换状态失败: ' + (e.message || e), 'error');
  }
}

export function openCreateSubscriberModal() {
  document.getElementById('subFormId').value = '';
  const row = document.getElementById('subFormIdRow');
  if (row) row.style.display = 'none';

  const title = document.getElementById('subscriberModalTitle');
  if (title) title.innerText = '新增下游订阅方';

  const btn = document.getElementById('btnSubmitSubscriber');
  if (btn) btn.innerText = '保存订阅方';

  const secretHint = document.getElementById('subFormSecretHint');
  if (secretHint) secretHint.innerText = '(必填，用于生成 X-Dispatcher-Signature)';

  document.getElementById('subFormName').value = '';
  document.getElementById('subFormAccountId').value = '';
  document.getElementById('subFormCallbackUrl').value = '';
  const secretInput = document.getElementById('subFormSecret');
  secretInput.value = '';
  secretInput.placeholder = '输入密钥字符串';

  document.getElementById('subFormEnabled').value = 'true';
  document.getElementById('subFormTimeout').value = '3000';
  document.getElementById('subFormMaxAttempts').value = '6';
  document.getElementById('subFormFilter').value = '{}';

  openModal('subscriberModal');
}

export async function openEditSubscriberModal(id) {
  try {
    const res = await fetchAPI(`/admin/subscribers/${id}`);
    const s = res && res.data;
    if (!s) {
      pushToast('获取订阅方详情失败', 'error');
      return;
    }

    document.getElementById('subFormId').value = s.id;
    const row = document.getElementById('subFormIdRow');
    if (row) row.style.display = 'flex';
    const display = document.getElementById('subFormIdDisplay');
    if (display) display.value = s.id;

    const title = document.getElementById('subscriberModalTitle');
    if (title) title.innerText = '编辑下游订阅方';

    const btn = document.getElementById('btnSubmitSubscriber');
    if (btn) btn.innerText = '保存修改';

    const secretHint = document.getElementById('subFormSecretHint');
    if (secretHint) secretHint.innerText = '(留空表示不修改已有 Secret)';

    document.getElementById('subFormName').value = s.name || '';
    document.getElementById('subFormAccountId').value = s.accountId || '';
    document.getElementById('subFormCallbackUrl').value = s.callbackUrl || '';

    const secretInput = document.getElementById('subFormSecret');
    secretInput.value = '';
    secretInput.placeholder = s.secretSet ? '【已配置】留空则保持不变' : '输入密钥字符串';

    document.getElementById('subFormEnabled').value = s.enabled ? 'true' : 'false';
    document.getElementById('subFormTimeout').value = s.timeoutMs || 3000;
    document.getElementById('subFormMaxAttempts').value = s.maxAttempts || 6;
    document.getElementById('subFormFilter').value = JSON.stringify(s.filter || {}, null, 2);

    openModal('subscriberModal');
  } catch (e) {
    pushToast('无法加载订阅方配置: ' + (e.message || e), 'error');
  }
}

export async function submitSubscriberForm() {
  const id = document.getElementById('subFormId').value.trim();
  const name = document.getElementById('subFormName').value.trim();
  const accountId = document.getElementById('subFormAccountId').value.trim();
  const callbackUrl = document.getElementById('subFormCallbackUrl').value.trim();
  const secret = document.getElementById('subFormSecret').value.trim();
  const enabled = document.getElementById('subFormEnabled').value === 'true';
  const timeoutMs = parseInt(document.getElementById('subFormTimeout').value, 10);
  const maxAttempts = parseInt(document.getElementById('subFormMaxAttempts').value, 10);
  let filter = {};

  try {
    const raw = document.getElementById('subFormFilter').value.trim();
    if (raw) filter = JSON.parse(raw);
  } catch (e) {
    pushToast('Filter JSON 格式错误，请检查 JSON 语法', 'error');
    return;
  }

  if (!id) {
    // 新增模式
    if (!name || !accountId || !callbackUrl || !secret) {
      pushToast('请完整填写必填项 (订阅方名称, 所属服务号 ID, 回调 URL, HMAC 密钥)', 'error');
      return;
    }

    setButtonLoading('btnSubmitSubscriber', true, '正在创建...');
    try {
      await fetchAPI('/admin/subscribers', {
        method: 'POST',
        body: JSON.stringify({ name, accountId, callbackUrl, secret, timeoutMs, maxAttempts, filter, enabled })
      });
      pushToast('下游订阅方配置成功', 'success');
      closeModal('subscriberModal');
      loadSubscribers();
    } catch (e) {
    } finally {
      setButtonLoading('btnSubmitSubscriber', false);
    }
  } else {
    // 编辑模式 (PUT)
    if (!name || !accountId || !callbackUrl) {
      pushToast('订阅方名称、所属服务号 ID、回调 URL 不能为空', 'error');
      return;
    }

    const payload = {
      name,
      accountId,
      callbackUrl,
      timeoutMs,
      maxAttempts,
      filter,
      enabled
    };
    if (secret) {
      payload.secret = secret;
    }

    setButtonLoading('btnSubmitSubscriber', true, '正在保存...');
    try {
      await fetchAPI(`/admin/subscribers/${id}`, {
        method: 'PUT',
        body: JSON.stringify(payload)
      });
      pushToast('订阅方配置已更新', 'success');
      closeModal('subscriberModal');
      loadSubscribers();
    } catch (e) {
    } finally {
      setButtonLoading('btnSubmitSubscriber', false);
    }
  }
}

// 连通性测试功能
export async function openTestSubscriberModal(id) {
  try {
    const res = await fetchAPI(`/admin/subscribers/${id}`);
    const s = res && res.data;
    if (!s) {
      pushToast('获取订阅方详情失败', 'error');
      return;
    }

    document.getElementById('subTestSubscriberId').value = s.id;
    document.getElementById('subTestTargetName').innerText = `${s.name} (${s.id})`;
    document.getElementById('subTestTargetUrl').innerText = s.callbackUrl;

    const samplePayload = {
      id: "evt_test_probe",
      accountKey: s.accountId || "target_account",
      msgType: "event",
      event: "test_ping",
      eventKey: "connectivity_test",
      fromUser: "EventDispatcher_Tester",
      toUser: s.accountId || "service_account",
      createTime: Math.floor(Date.now() / 1000),
      receivedAt: new Date().toISOString(),
      fields: {
        MsgType: "event",
        Event: "test_ping",
        EventKey: "connectivity_test",
        Content: "EventDispatcher 下游连通性测试 (Connectivity Test)"
      }
    };
    const pre = document.getElementById('subTestPayloadPreview');
    if (pre) pre.innerText = JSON.stringify(samplePayload, null, 2);

    // 重置结果展示区域
    document.getElementById('subTestLoading').style.display = 'none';
    document.getElementById('subTestResultArea').style.display = 'none';

    openModal('subscriberTestModal');
  } catch (e) {
    pushToast('无法打开测试弹窗: ' + (e.message || e), 'error');
  }
}

export async function runSubscriberTest() {
  const id = document.getElementById('subTestSubscriberId').value;
  if (!id) return;

  const btn = document.getElementById('btnExecuteSubTest');
  const loading = document.getElementById('subTestLoading');
  const resultArea = document.getElementById('subTestResultArea');
  const alertBox = document.getElementById('subTestAlertBox');
  const statusCodeEl = document.getElementById('subTestStatusCode');
  const latencyEl = document.getElementById('subTestLatency');
  const eventIdEl = document.getElementById('subTestEventId');
  const bodyPre = document.getElementById('subTestResponseBody');

  setButtonLoading(btn, true, '正在探测...');
  loading.style.display = 'flex';
  resultArea.style.display = 'none';

  try {
    const res = await fetchAPI(`/admin/subscribers/${id}/test`, { method: 'POST' });
    const data = res && res.data;
    if (!data) {
      throw new Error('未返回测试结果');
    }

    resultArea.style.display = 'block';

    if (data.success) {
      alertBox.className = 'alert-box alert-box-success';
      alertBox.innerHTML = `<strong>连通性测试成功！</strong> 下游端点已成功确认接收测试事件 (HTTP ${data.statusCode})。`;
    } else {
      alertBox.className = 'alert-box alert-box-error';
      alertBox.innerHTML = `<strong>连通性测试未通过：</strong> ${escapeHtml(data.error || '下游端点未正常响应')}`;
    }

    statusCodeEl.innerHTML = data.statusCode
      ? `<span class="mono-tag ${data.statusCode >= 200 && data.statusCode < 300 ? 'status-pill-success' : 'status-pill-failed'}">HTTP ${data.statusCode} (${escapeHtml(data.statusText || '')})</span>`
      : '<span class="status-pill status-pill-failed">网络超时或连接拒绝</span>';

    latencyEl.innerHTML = `<span style="font-family: var(--font-mono); font-weight: 600; color: ${data.latencyMs < 500 ? 'var(--accent-green)' : 'var(--accent-yellow)'}">${data.latencyMs} ms</span>`;
    eventIdEl.innerText = data.eventId || '-';
    bodyPre.innerText = data.responseBody || '(无响应内容)';

  } catch (e) {
    resultArea.style.display = 'block';
    alertBox.className = 'alert-box alert-box-error';
    alertBox.innerHTML = `<strong>测试请求失败：</strong> ${escapeHtml(e.message || e)}`;
    statusCodeEl.innerHTML = '<span class="status-pill status-pill-failed">错误</span>';
    latencyEl.innerText = '-';
    eventIdEl.innerText = '-';
    bodyPre.innerText = e.message || '网络连接或鉴权异常';
  } finally {
    setButtonLoading(btn, false);
    loading.style.display = 'none';
  }
}

// 在新增/编辑弹窗中直接测试当前输入的配置
export async function testSubscriberFormConfig() {
  const subId = document.getElementById('subFormId').value.trim();
  const accountId = document.getElementById('subFormAccountId').value.trim();
  const callbackUrl = document.getElementById('subFormCallbackUrl').value.trim();
  const secret = document.getElementById('subFormSecret').value.trim();
  const timeoutMs = parseInt(document.getElementById('subFormTimeout').value, 10) || 3000;

  if (!callbackUrl) {
    pushToast('请先输入下游 Webhook URL', 'error');
    return;
  }

  // 如果处于编辑状态且 secret 为空，说明复用已保存的 secret，调用已有测试端点
  if (subId && !secret) {
    openTestSubscriberModal(subId);
    return;
  }

  if (!secret) {
    pushToast('请先填写 HMAC 密钥 (Secret)', 'error');
    return;
  }

  setButtonLoading('btnTestSubForm', true, '正在探测...');

  try {
    const res = await fetchAPI('/admin/subscribers/test', {
      method: 'POST',
      body: JSON.stringify({ accountId, callbackUrl, secret, timeoutMs })
    });
    const data = res && res.data;
    if (data && data.success) {
      pushToast(`测试连通成功！HTTP ${data.statusCode}，耗时 ${data.latencyMs}ms`, 'success');
    } else {
      pushToast(`测试连通失败: ${data ? data.error : '连接异常'}`, 'error');
    }
  } catch (e) {
    pushToast('测试异常: ' + (e.message || e), 'error');
  } finally {
    setButtonLoading('btnTestSubForm', false);
  }
}
