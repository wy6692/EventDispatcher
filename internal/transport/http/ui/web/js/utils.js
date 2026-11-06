// 通用工具函数模块

export function escapeHtml(str) {
  if (!str) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

export function copyText(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(() => {
      pushToast('已复制到剪贴板: ' + text, 'success');
    }).catch(() => {
      promptCopy(text);
    });
  } else {
    promptCopy(text);
  }
}

function promptCopy(text) {
  window.prompt('请手动复制内容：', text);
}

export function pushToast(msg, type = 'success') {
  const feed = document.getElementById('toastFeed');
  if (!feed) return;
  const item = document.createElement('div');
  item.className = 'toast-msg ' + type;
  item.innerText = msg;
  feed.appendChild(item);
  setTimeout(() => { item.remove(); }, 3200);
}

export function openModal(id) {
  const el = document.getElementById(id);
  if (el) el.classList.add('open');
}

export function closeModal(id) {
  const el = document.getElementById(id);
  if (el) el.classList.remove('open');
}
