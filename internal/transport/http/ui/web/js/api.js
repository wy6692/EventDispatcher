// API 客户端封装模块
import { state } from './state.js';
import { pushToast, openModal } from './utils.js';

export const API_BASE = window.location.origin;

export async function fetchAPI(path, opts = {}) {
  const headers = Object.assign({
    'Authorization': 'Bearer ' + state.token,
    'Content-Type': 'application/json'
  }, opts.headers || {});

  try {
    const resp = await fetch(API_BASE + path, Object.assign({}, opts, { headers }));
    if (resp.status === 204) return { code: 0, message: 'success', data: null };
    const json = await resp.json();
    if (resp.status === 401) {
      pushToast('管理员口令无效或未配置，请配置后重试', 'error');
      openModal('tokenModal');
      return null;
    }
    if (!resp.ok || (json.code && json.code !== 0)) {
      throw new Error(json.message || `操作失败 (HTTP ${resp.status})`);
    }
    return json;
  } catch (err) {
    pushToast(err.message, 'error');
    throw err;
  }
}

export async function checkHealth() {
  const dot = document.getElementById('systemDot');
  const text = document.getElementById('systemStatusText');
  try {
    const res = await fetch(API_BASE + '/healthz');
    const json = await res.json();
    if (res.ok && json.code === 0) {
      if (dot) dot.className = 'status-dot';
      if (text) text.innerText = '存储正常 (Online)';
    } else {
      if (dot) dot.className = 'status-dot error';
      if (text) text.innerText = '异常 (Degraded)';
    }
  } catch (e) {
    if (dot) dot.className = 'status-dot error';
    if (text) text.innerText = '离线 (Offline)';
  }
}

export const checkSystemHealth = checkHealth;
