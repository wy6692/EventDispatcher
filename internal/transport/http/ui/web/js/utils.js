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

/**
 * 切换按钮的加载状态与加载动画 (Level 3: Button Loading Animation)
 * @param {HTMLElement|string} btn - 目标按钮元素或元素 ID
 * @param {boolean} isLoading - 是否处于加载中
 * @param {string} loadingText - 加载中展示的文字
 */
export function setButtonLoading(btn, isLoading, loadingText = '处理中...') {
  const el = typeof btn === 'string' ? document.getElementById(btn) : btn;
  if (!el) return;

  if (isLoading) {
    if (!el.dataset.originalHtml) {
      el.dataset.originalHtml = el.innerHTML;
    }
    el.disabled = true;
    el.classList.add('btn-loading');
    el.innerHTML = `<span class="btn-spinner"></span><span>${escapeHtml(loadingText)}</span>`;
  } else {
    el.disabled = false;
    el.classList.remove('btn-loading');
    if (el.dataset.originalHtml) {
      el.innerHTML = el.dataset.originalHtml;
      delete el.dataset.originalHtml;
    }
  }
}

/**
 * 生成平滑波浪骨架屏占位行 (Level 2: Table Skeleton Loader)
 * @param {number} colCount - 列数
 * @param {number} rowCount - 骨架行数
 * @returns {string} 骨架 HTML 字符串
 */
export function renderTableSkeleton(colCount = 8, rowCount = 4) {
  const widths = ['80%', '60%', '90%', '70%', '45%', '85%', '50%'];
  let rows = '';
  for (let r = 0; r < rowCount; r++) {
    let cells = '';
    for (let c = 0; c < colCount; c++) {
      if (c === colCount - 1) {
        // 操作列
        cells += `<td style="text-align: right;"><div class="skeleton-box" style="width: 75px; height: 22px;"></div></td>`;
      } else {
        const w = widths[(r * 2 + c) % widths.length];
        cells += `<td><div class="skeleton-box" style="width: ${w}; height: 14px;"></div></td>`;
      }
    }
    rows += `<tr class="skeleton-row">${cells}</tr>`;
  }
  return rows;
}

/**
 * 渲染通用分页控制栏 (Pagination Component)
 * @param {Object} options
 * @param {string} options.containerId - 容器元素 ID
 * @param {number} options.page - 当前页码 (1-based)
 * @param {number} options.limit - 每页条数
 * @param {number} options.total - 总记录数
 * @param {number} options.totalPages - 总页数
 * @param {function(number)} options.onPageChange - 页码切换事件
 * @param {function(number)} options.onLimitChange - 每页数量变更事件
 */
export function renderPagination({
  containerId,
  page,
  limit,
  total,
  totalPages,
  onPageChange,
  onLimitChange
}) {
  const container = document.getElementById(containerId);
  if (!container) return;

  if (total <= 0) {
    container.innerHTML = `
      <div class="pagination-info">
        <span>暂无记录 (共 <strong>0</strong> 条)</span>
      </div>
    `;
    return;
  }

  // 计算展示的分页胶囊按钮 (最多显示 5 个数字)
  const maxButtons = 5;
  let startPage = Math.max(1, page - Math.floor(maxButtons / 2));
  let endPage = Math.min(totalPages, startPage + maxButtons - 1);
  if (endPage - startPage + 1 < maxButtons) {
    startPage = Math.max(1, endPage - maxButtons + 1);
  }

  let pillsHtml = '';
  if (startPage > 1) {
    pillsHtml += `<button class="page-btn" data-page="1">1</button>`;
    if (startPage > 2) pillsHtml += `<span class="page-ellipsis">...</span>`;
  }

  for (let i = startPage; i <= endPage; i++) {
    pillsHtml += `<button class="page-btn ${i === page ? 'active' : ''}" data-page="${i}">${i}</button>`;
  }

  if (endPage < totalPages) {
    if (endPage < totalPages - 1) pillsHtml += `<span class="page-ellipsis">...</span>`;
    pillsHtml += `<button class="page-btn" data-page="${totalPages}">${totalPages}</button>`;
  }

  const startRecord = Math.min(total, (page - 1) * limit + 1);
  const endRecord = Math.min(total, page * limit);

  container.innerHTML = `
    <div class="pagination-info">
      <span>显示第 <strong>${startRecord}-${endRecord}</strong> 条，共 <strong>${total}</strong> 条记录</span>
    </div>
    <div class="pagination-controls">
      <select class="pagination-size-select">
        <option value="10" ${limit === 10 ? 'selected' : ''}>10 条/页</option>
        <option value="15" ${limit === 15 ? 'selected' : ''}>15 条/页</option>
        <option value="30" ${limit === 30 ? 'selected' : ''}>30 条/页</option>
        <option value="50" ${limit === 50 ? 'selected' : ''}>50 条/页</option>
      </select>
      <button class="page-btn" data-action="prev" ${page <= 1 ? 'disabled' : ''} title="上一页">&laquo;</button>
      <div class="pagination-pills">${pillsHtml}</div>
      <button class="page-btn" data-action="next" ${page >= totalPages ? 'disabled' : ''} title="下一页">&raquo;</button>
    </div>
  `;

  // 绑定事件
  const select = container.querySelector('.pagination-size-select');
  if (select && onLimitChange) {
    select.addEventListener('change', (e) => {
      onLimitChange(parseInt(e.target.value, 10));
    });
  }

  container.querySelectorAll('.page-btn[data-page]').forEach(btn => {
    btn.addEventListener('click', () => {
      const p = parseInt(btn.getAttribute('data-page'), 10);
      if (p !== page && onPageChange) onPageChange(p);
    });
  });

  const prevBtn = container.querySelector('.page-btn[data-action="prev"]');
  if (prevBtn && onPageChange) {
    prevBtn.addEventListener('click', () => {
      if (page > 1) onPageChange(page - 1);
    });
  }

  const nextBtn = container.querySelector('.page-btn[data-action="next"]');
  if (nextBtn && onPageChange) {
    nextBtn.addEventListener('click', () => {
      if (page < totalPages) onPageChange(page + 1);
    });
  }
}
