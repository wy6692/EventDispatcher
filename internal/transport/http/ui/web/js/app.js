import { getToken, setToken, updateTokenState } from './state.js';
import { checkHealth } from './api.js';
import { copyText, openModal, closeModal, pushToast } from './utils.js';
import { loadOverview } from './views/overview.js';
import { loadAccounts, deleteAccount, openCreateAccountModal, submitAccountForm, copyCallbackUrl } from './views/accounts.js';
import { loadSubscribers, deleteSubscriber, openCreateSubscriberModal, submitSubscriberForm } from './views/subscribers.js';
import { loadEvents, viewPayload } from './views/events.js';
import { loadDeliveries, retryDelivery } from './views/deliveries.js';

const VALID_TABS = ['overview', 'accounts', 'subscribers', 'events', 'deliveries'];
let activeTab = 'overview';

const TAB_LABELS = {
  'overview': '系统概览',
  'accounts': '微信服务号',
  'subscribers': '下游订阅与路由',
  'events': '事件接收日志',
  'deliveries': '投递调度与重试'
};

export function getTabFromHash() {
  const raw = window.location.hash.replace(/^#\/?/, '').trim();
  return VALID_TABS.includes(raw) ? raw : 'overview';
}

export function switchTab(tab, updateHash = true) {
  if (!VALID_TABS.includes(tab)) tab = 'overview';
  activeTab = tab;

  if (updateHash) {
    if (window.location.hash !== '#' + tab) {
      if (window.history && window.history.replaceState) {
        window.history.replaceState(null, '', '#' + tab);
      } else {
        window.location.hash = '#' + tab;
      }
    }
  }

  document.querySelectorAll('.nav-item').forEach(el => el.classList.remove('active'));
  document.querySelectorAll('.view-pane').forEach(el => el.style.display = 'none');

  const targetNav = document.querySelector(`.nav-item[data-tab="${tab}"]`);
  if (targetNav) targetNav.classList.add('active');

  const targetPane = document.getElementById('view-' + tab);
  if (targetPane) targetPane.style.display = 'block';

  const breadcrumb = document.getElementById('activeBreadcrumb');
  if (breadcrumb) {
    breadcrumb.innerText = TAB_LABELS[tab] || '工作台';
  }

  refreshCurrentTab();
}

export function refreshCurrentTab() {
  checkHealth();
  if (activeTab === 'overview') loadOverview();
  else if (activeTab === 'accounts') loadAccounts();
  else if (activeTab === 'subscribers') loadSubscribers();
  else if (activeTab === 'events') loadEvents();
  else if (activeTab === 'deliveries') loadDeliveries();
}

function openTokenModal() {
  const input = document.getElementById('inputToken');
  if (input) input.value = getToken();
  openModal('tokenModal');
}

function saveToken() {
  const input = document.getElementById('inputToken');
  if (!input) return;
  setToken(input.value);
  closeModal('tokenModal');
  pushToast('管理员凭据已保存', 'success');
  refreshCurrentTab();
}

// 全局 DOM 事件绑定与事件代理
function bindEvents() {
  // 侧边栏导航点击
  document.querySelectorAll('.nav-item[data-tab]').forEach(btn => {
    btn.addEventListener('click', () => {
      const tab = btn.getAttribute('data-tab');
      if (tab) switchTab(tab);
    });
  });

  // 顶部与操作按钮
  const btnRefresh = document.getElementById('btnRefresh');
  if (btnRefresh) btnRefresh.addEventListener('click', refreshCurrentTab);

  const btnOpenToken = document.getElementById('btnOpenToken');
  if (btnOpenToken) btnOpenToken.addEventListener('click', openTokenModal);

  const btnSaveToken = document.getElementById('btnSaveToken');
  if (btnSaveToken) btnSaveToken.addEventListener('click', saveToken);

  const btnCloseToken = document.getElementById('btnCloseToken');
  if (btnCloseToken) btnCloseToken.addEventListener('click', () => closeModal('tokenModal'));

  // 复制回调前缀辅助按钮
  const btnCopyHelperUrl = document.getElementById('btnCopyHelperUrl');
  if (btnCopyHelperUrl) {
    btnCopyHelperUrl.addEventListener('click', () => {
      const helper = document.getElementById('helperUrl');
      if (helper) copyText(helper.value);
    });
  }

  // 服务号相关
  const btnOpenCreateAccount = document.getElementById('btnOpenCreateAccount');
  if (btnOpenCreateAccount) btnOpenCreateAccount.addEventListener('click', openCreateAccountModal);

  const btnSubmitAccount = document.getElementById('btnSubmitAccount');
  if (btnSubmitAccount) btnSubmitAccount.addEventListener('click', submitAccountForm);

  const btnCloseAccountModal = document.getElementById('btnCloseAccountModal');
  if (btnCloseAccountModal) btnCloseAccountModal.addEventListener('click', () => closeModal('accountModal'));

  // 订阅者相关
  const btnOpenCreateSubscriber = document.getElementById('btnOpenCreateSubscriber');
  if (btnOpenCreateSubscriber) btnOpenCreateSubscriber.addEventListener('click', openCreateSubscriberModal);

  const btnSubmitSubscriber = document.getElementById('btnSubmitSubscriber');
  if (btnSubmitSubscriber) btnSubmitSubscriber.addEventListener('click', submitSubscriberForm);

  const btnCloseSubscriberModal = document.getElementById('btnCloseSubscriberModal');
  if (btnCloseSubscriberModal) btnCloseSubscriberModal.addEventListener('click', () => closeModal('subscriberModal'));

  // 载荷查看器弹窗
  const btnCopyPayload = document.getElementById('btnCopyPayload');
  if (btnCopyPayload) {
    btnCopyPayload.addEventListener('click', () => {
      const pre = document.getElementById('payloadViewText');
      if (pre) copyText(pre.innerText);
    });
  }

  const btnClosePayload = document.getElementById('btnClosePayload');
  if (btnClosePayload) btnClosePayload.addEventListener('click', () => closeModal('payloadModal'));

  const btnClosePayloadX = document.getElementById('btnClosePayloadX');
  if (btnClosePayloadX) btnClosePayloadX.addEventListener('click', () => closeModal('payloadModal'));

  // 投递状态筛选
  const deliveryFilter = document.getElementById('deliveryStatusFilter');
  if (deliveryFilter) {
    deliveryFilter.addEventListener('change', loadDeliveries);
  }

  // 动态表格代理操作 (委托至 container 或 document)
  document.addEventListener('click', (e) => {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    const action = target.getAttribute('data-action');
    const id = target.getAttribute('data-id');

    switch (action) {
      case 'copy-url': {
        const key = target.getAttribute('data-key');
        if (key) copyCallbackUrl(key);
        break;
      }
      case 'delete-account':
        if (id) deleteAccount(id);
        break;
      case 'delete-subscriber':
        if (id) deleteSubscriber(id);
        break;
      case 'retry-delivery':
        if (id) retryDelivery(id);
        break;
      case 'view-payload': {
        const raw = target.getAttribute('data-payload');
        if (raw) {
          try {
            viewPayload(JSON.parse(raw));
          } catch {
            viewPayload({ raw });
          }
        }
        break;
      }
      case 'close-modal': {
        const modalId = target.getAttribute('data-modal');
        if (modalId) closeModal(modalId);
        break;
      }
    }
  });
}

// 页面初始化入口
function initApp() {
  const helperUrl = document.getElementById('helperUrl');
  if (helperUrl) {
    helperUrl.value = `${window.location.origin}/w/{accountKey}`;
  }

  bindEvents();
  updateTokenState();

  // 监听浏览器前进、后退及哈希变更
  window.addEventListener('hashchange', () => {
    const tab = getTabFromHash();
    if (tab !== activeTab) {
      switchTab(tab, false);
    }
  });

  // 根据当前哈希定位初始 Tab，刷新页面不丢失选中项
  const initialTab = getTabFromHash();
  switchTab(initialTab, false);

  if (!getToken()) {
    openTokenModal();
  }
}

if (document.readyState === 'loading') {
  window.addEventListener('DOMContentLoaded', initApp);
} else {
  initApp();
}
