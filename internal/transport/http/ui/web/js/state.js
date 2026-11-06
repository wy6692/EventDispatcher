// 全局状态管理模块

const TOKEN_KEY = 'EVENT_DISPATCHER_ADMIN_TOKEN';

export const state = {
  activeTab: 'overview',
  token: localStorage.getItem(TOKEN_KEY) || '',
};

export function getToken() {
  return state.token;
}

export function setToken(token) {
  state.token = (token || '').trim();
  localStorage.setItem(TOKEN_KEY, state.token);
  updateTokenUI();
}

export function updateTokenUI() {
  const label = document.getElementById('tokenStatusLabel');
  if (label) {
    label.innerText = state.token ? '口令已配置' : '配置口令';
  }
}

export const updateTokenState = updateTokenUI;
