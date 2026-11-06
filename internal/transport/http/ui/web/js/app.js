import { getToken, setToken, updateTokenState } from './state.js';
import { checkHealth } from './api.js';
import { copyText, openModal, closeModal, pushToast, setButtonLoading } from './utils.js';
import { loadOverview, resizeCharts } from './views/overview.js';
import { loadAccounts, deleteAccount, openCreateAccountModal, openEditAccountModal, submitAccountForm, copyCallbackUrl } from './views/accounts.js';
import { loadSubscribers, deleteSubscriber, openCreateSubscriberModal, openEditSubscriberModal, submitSubscriberForm, toggleSubscriberEnabled, openTestSubscriberModal, runSubscriberTest, testSubscriberFormConfig, initSubscriberFilters } from './views/subscribers.js';
import { loadEvents, viewPayload, initEventFilters } from './views/events.js';
import { loadDeliveries, retryDelivery, openDeliveryDetail, initDeliveryDetailActions, initDeliveryFilters } from './views/deliveries.js';

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
  if (tab === 'overview') {
    setTimeout(resizeCharts, 50);
  }

  const breadcrumb = document.getElementById('activeBreadcrumb');
  if (breadcrumb) {
    breadcrumb.innerText = TAB_LABELS[tab] || '工作台';
  }

  refreshCurrentTab();
}

export function refreshCurrentTab() {
  const btn = document.getElementById('btnRefresh');
  const icon = btn ? btn.querySelector('.refresh-icon') : null;
  if (icon) icon.classList.add('spinning');
  if (btn) btn.disabled = true;

  checkHealth();
  let p;
  if (activeTab === 'overview') p = loadOverview();
  else if (activeTab === 'accounts') p = loadAccounts();
  else if (activeTab === 'subscribers') p = loadSubscribers();
  else if (activeTab === 'events') p = loadEvents();
  else if (activeTab === 'deliveries') p = loadDeliveries();

  Promise.resolve(p).finally(() => {
    setTimeout(() => {
      if (icon) icon.classList.remove('spinning');
      if (btn) btn.disabled = false;
    }, 400);
  });
}

function openTokenModal() {
  const input = document.getElementById('inputToken');
  if (input) input.value = getToken();
  openModal('tokenModal');
}

function saveToken() {
  const input = document.getElementById('inputToken');
  if (!input) return;
  setButtonLoading('btnSaveToken', true, '保存中...');
  setTimeout(() => {
    setToken(input.value);
    setButtonLoading('btnSaveToken', false);
    closeModal('tokenModal');
    pushToast('管理员凭据已保存', 'success');
    refreshCurrentTab();
  }, 250);
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

  // 连通性测试按钮
  const btnTestSubForm = document.getElementById('btnTestSubForm');
  if (btnTestSubForm) btnTestSubForm.addEventListener('click', testSubscriberFormConfig);

  const btnExecuteSubTest = document.getElementById('btnExecuteSubTest');
  if (btnExecuteSubTest) btnExecuteSubTest.addEventListener('click', runSubscriberTest);

  // 初始化投递详情弹窗事件
  initDeliveryDetailActions();

  // 查看投递任务关联事件载荷
  const btnViewDlvEvent = document.getElementById('btnViewDlvEventPayload');
  if (btnViewDlvEvent) {
    btnViewDlvEvent.addEventListener('click', () => {
      const text = document.getElementById('dlvDetailPayloadText').innerText;
      if (text && text !== '载荷加载中...' && text !== '正在加载关联事件载荷...') {
        try {
          viewPayload(JSON.parse(text));
        } catch {
          viewPayload({ payload: text });
        }
      } else {
        pushToast('载荷尚未加载完成', 'error');
      }
    });
  }

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

  // 初始化各视图筛选器
  initSubscriberFilters();
  initEventFilters();
  initDeliveryFilters();

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
      case 'edit-account':
        if (id) openEditAccountModal(id);
        break;
      case 'delete-account':
        if (id) deleteAccount(id);
        break;
      case 'edit-subscriber':
        if (id) openEditSubscriberModal(id);
        break;
      case 'toggle-subscriber-enabled': {
        const cur = target.getAttribute('data-enabled') === 'true';
        if (id) toggleSubscriberEnabled(id, cur);
        break;
      }
      case 'test-subscriber':
        if (id) openTestSubscriberModal(id);
        break;
      case 'delete-subscriber':
        if (id) deleteSubscriber(id);
        break;
      case 'view-delivery':
        if (id) openDeliveryDetail(id);
        break;
      case 'retry-delivery':
        if (id) retryDelivery(id, target);
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
