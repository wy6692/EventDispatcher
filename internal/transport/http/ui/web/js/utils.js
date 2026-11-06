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
  if (window.Swal) {
    const Toast = window.Swal.mixin({
      toast: true,
      position: 'top-end',
      showConfirmButton: false,
      timer: 3200,
      timerProgressBar: true,
      background: '#161b22',
      color: '#f0f6fc'
    });
    Toast.fire({
      icon: type === 'error' ? 'error' : 'success',
      title: msg
    });
    return;
  }

  const feed = document.getElementById('toastFeed');
  if (!feed) return;
  const item = document.createElement('div');
  item.className = 'toast-msg ' + type;
  item.innerText = msg;
  feed.appendChild(item);
  setTimeout(() => { item.remove(); }, 3200);
}

export async function confirmAction(title, text = '') {
  if (window.Swal) {
    const res = await window.Swal.fire({
      title,
      text,
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      confirmButtonColor: '#2b5680',
      cancelButtonColor: '#30363d',
      background: '#161b22',
      color: '#f0f6fc'
    });
    return res.isConfirmed;
  }
  return window.confirm(title);
}

export function openModal(id) {
  if (window.$) {
    $('#' + id).addClass('open');
    return;
  }
  const el = document.getElementById(id);
  if (el) el.classList.add('open');
}

export function closeModal(id) {
  if (window.$) {
    $('#' + id).removeClass('open');
    return;
  }
  const el = document.getElementById(id);
  if (el) el.classList.remove('open');
}
