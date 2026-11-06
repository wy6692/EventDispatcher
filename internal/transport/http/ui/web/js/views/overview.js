// 视图模块：系统概览 (Overview) - 包含数据统计、ECharts 动态图表与接入指引
import { fetchAPI } from '../api.js';
import { copyText, pushToast } from '../utils.js';

let throughputChart = null;
let statusPieChart = null;
let tutorialInitialized = false;

/**
 * 调整图表尺寸
 */
export function resizeCharts() {
  if (throughputChart) {
    try { throughputChart.resize(); } catch (e) { /* ignore */ }
  }
  if (statusPieChart) {
    try { statusPieChart.resize(); } catch (e) { /* ignore */ }
  }
}

/**
 * 初始化图表实例与主题配置
 */
function initCharts() {
  if (typeof echarts === 'undefined') {
    return;
  }

  const elThroughput = document.getElementById('chartThroughput');
  if (elThroughput && !throughputChart) {
    throughputChart = echarts.init(elThroughput);
  }

  const elStatusPie = document.getElementById('chartStatusPie');
  if (elStatusPie && !statusPieChart) {
    statusPieChart = echarts.init(elStatusPie);
  }
}

/**
 * 渲染 24 小时事件吞吐时序面积图
 */
function renderThroughputChart(events) {
  if (!throughputChart) return;

  // 生成近 24 小时的时间轴槽位 (每小时 1 个点)
  const now = new Date();
  const hours = [];
  const counts = [];
  const hourMap = new Map();

  for (let i = 23; i >= 0; i--) {
    const d = new Date(now.getTime() - i * 3600 * 1000);
    const key = `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:00`;
    const label = `${String(d.getHours()).padStart(2, '0')}:00`;
    hours.push(label);
    hourMap.set(key, 0);
  }

  // 统计事件落在各小时槽位中的频次
  (events || []).forEach(evt => {
    if (!evt.created_at) return;
    const d = new Date(evt.created_at);
    const key = `${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:00`;
    if (hourMap.has(key)) {
      hourMap.set(key, hourMap.get(key) + 1);
    }
  });

  const hourValues = Array.from(hourMap.values());
  for (let i = 0; i < hourValues.length; i++) {
    counts.push(hourValues[i]);
  }

  const totalRecent = counts.reduce((a, b) => a + b, 0);
  const metaEl = document.getElementById('throughputChartMeta');
  if (metaEl) {
    metaEl.innerText = `近24小时事件: ${totalRecent} 笔`;
  }

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#181c24',
      borderColor: '#30363d',
      borderWidth: 1,
      textStyle: { color: '#e6edf3', fontSize: 12 },
      formatter: (params) => {
        const item = params[0];
        return `<div style="font-size:11px;color:#8b949e;margin-bottom:4px;">时段: ${item.axisValue}</div>
                <div style="font-size:13px;font-weight:600;color:#38bdf8;">接收事件: ${item.data} 笔</div>`;
      }
    },
    grid: {
      top: 30,
      right: 20,
      bottom: 25,
      left: 42,
      containLabel: false
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: hours,
      axisLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.1)' } },
      axisTick: { show: false },
      axisLabel: {
        color: '#6e7681',
        fontSize: 10,
        interval: 3 // 每 4 小时显示一个刻度标签，避免拥挤
      }
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      splitLine: { lineStyle: { color: 'rgba(255, 255, 255, 0.05)', type: 'dashed' } },
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: '#6e7681', fontSize: 10 }
    },
    series: [
      {
        name: '接收事件',
        type: 'line',
        smooth: 0.35,
        symbol: 'circle',
        symbolSize: 4,
        showSymbol: false,
        itemStyle: { color: '#38bdf8' },
        lineStyle: { width: 2, color: '#38bdf8' },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: 'rgba(56, 189, 248, 0.35)' },
              { offset: 1, color: 'rgba(56, 189, 248, 0.01)' }
            ]
          }
        },
        data: counts
      }
    ]
  };

  throughputChart.setOption(option);
}

/**
 * 渲染投递状态与健康度占比环形图
 */
function renderDistributionChart(deliveries) {
  if (!statusPieChart) return;

  let success = 0;
  let failed = 0;
  let pending = 0;
  let retrying = 0;

  (deliveries || []).forEach(d => {
    const s = (d.status || '').toLowerCase();
    if (s === 'success') success++;
    else if (s === 'failed') failed++;
    else if (s === 'retrying') retrying++;
    else pending++;
  });

  const total = success + failed + pending + retrying;
  const metaEl = document.getElementById('distributionChartMeta');
  if (metaEl) {
    const rate = total > 0 ? ((success / total) * 100).toFixed(1) : '100.0';
    metaEl.innerText = `投递成功率: ${rate}% (总计 ${total})`;
  }

  const pieData = total === 0 ? [
    { value: 1, name: '暂无投递', itemStyle: { color: '#30363d' } }
  ] : [
    { value: success, name: '投递成功', itemStyle: { color: '#3fb950' } },
    { value: failed, name: '投递失败', itemStyle: { color: '#f85149' } },
    { value: retrying, name: '重试中', itemStyle: { color: '#d29922' } },
    { value: pending, name: '等待调度', itemStyle: { color: '#58a6ff' } }
  ].filter(item => item.value > 0);

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: '#181c24',
      borderColor: '#30363d',
      borderWidth: 1,
      textStyle: { color: '#e6edf3', fontSize: 12 },
      formatter: '{b}: <strong style="color:#fff;">{c}</strong> ({d}%)'
    },
    legend: {
      bottom: 6,
      left: 'center',
      textStyle: { color: '#8b949e', fontSize: 11 },
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 14
    },
    series: [
      {
        name: '投递状态',
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['50%', '44%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 4,
          borderColor: '#121418',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 13,
            fontWeight: '600',
            color: '#fff',
            formatter: '{b}\n{d}%'
          }
        },
        labelLine: { show: false },
        data: pieData
      }
    ]
  };

  statusPieChart.setOption(option);
}

/**
 * 绑定网关接入指引步骤切换与多语言代码复制
 */
function initTutorialInteractions() {
  if (tutorialInitialized) return;
  tutorialInitialized = true;

  // 步骤卡片切换
  document.querySelectorAll('.tutorial-step-card').forEach(card => {
    card.addEventListener('click', () => {
      const step = card.getAttribute('data-step');
      if (!step) return;

      document.querySelectorAll('.tutorial-step-card').forEach(c => c.classList.remove('active'));
      card.classList.add('active');

      document.querySelectorAll('.tutorial-pane').forEach(p => p.classList.remove('active'));
      const targetPane = document.getElementById(`tutorial-pane-${step}`);
      if (targetPane) targetPane.classList.add('active');
    });
  });

  // 语言选项卡切换
  document.querySelectorAll('.lang-tab-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const lang = btn.getAttribute('data-lang');
      if (!lang) return;

      document.querySelectorAll('.lang-tab-btn').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');

      document.querySelectorAll('.lang-snippet').forEach(s => s.style.display = 'none');
      const targetSnippet = document.getElementById(`snippet-${lang}`);
      if (targetSnippet) targetSnippet.style.display = 'block';
    });
  });

  // 代码与配置一键复制
  document.querySelectorAll('.btn-code-copy').forEach(btn => {
    btn.addEventListener('click', () => {
      const targetId = btn.getAttribute('data-target');
      if (!targetId) return;
      const targetEl = document.getElementById(targetId);
      if (!targetEl) return;

      const text = targetEl.innerText || targetEl.textContent || '';
      copyText(text);

      const origText = btn.innerText;
      btn.innerText = '已复制!';
      btn.style.color = 'var(--accent-green)';
      btn.style.borderColor = 'var(--accent-green)';
      setTimeout(() => {
        btn.innerText = origText;
        btn.style.color = '';
        btn.style.borderColor = '';
      }, 1500);
    });
  });

  // 复制回调前缀
  const btnCopyHelper = document.getElementById('btnCopyHelperUrl');
  if (btnCopyHelper) {
    btnCopyHelper.addEventListener('click', () => {
      const input = document.getElementById('helperUrl');
      if (input && input.value) {
        copyText(input.value);
        pushToast('微信服务器回调 URL 前缀已复制到剪贴板', 'info');
      }
    });
  }

  // 窗口重置自适应图表
  window.addEventListener('resize', () => {
    resizeCharts();
  });
}

/**
 * 加载系统概览数据
 */
export async function loadOverview() {
  const elAccs = document.getElementById('kpiAccounts');
  const elSubs = document.getElementById('kpiSubscribers');
  const elEvts = document.getElementById('kpiEvents');
  const elDels = document.getElementById('kpiFailedDeliveries');

  // 初始化指引交互
  initTutorialInteractions();

  // 更新辅助 URL 输入框
  const helperInput = document.getElementById('helperUrl');
  if (helperInput) {
    helperInput.value = `${window.location.origin}/wechat/{accountKey}`;
  }

  // 初始化图表实例
  initCharts();

  try {
    const [accs, subs, evts, dels] = await Promise.all([
      fetchAPI('/admin/accounts'),
      fetchAPI('/admin/subscribers'),
      fetchAPI('/admin/events?page=1&limit=200'),
      fetchAPI('/admin/deliveries?page=1&limit=500')
    ]);

    // 更新 KPI 卡片
    if (accs && accs.data && elAccs) {
      elAccs.innerText = (accs.data.items || []).length;
    }
    if (subs && subs.data && elSubs) {
      const activeCount = (subs.data.items || []).filter(s => s.enabled).length;
      elSubs.innerText = activeCount;
    }
    if (evts && evts.data && elEvts) {
      const totalEvents = evts.data.total !== undefined ? evts.data.total : (evts.data.items || []).length;
      elEvts.innerText = totalEvents;
    }
    if (dels && dels.data && elDels) {
      const failedCount = (dels.data.items || []).filter(d => d.status === 'failed').length;
      elDels.innerText = failedCount;
    }

    [elAccs, elSubs, elEvts, elDels].forEach(el => {
      if (el) {
        el.style.animation = 'none';
        void el.offsetHeight;
        el.style.animation = 'fadeInRow 0.25s ease-out';
      }
    });

    // 渲染动态图表
    renderThroughputChart((evts && evts.data && evts.data.items) || []);
    renderDistributionChart((dels && dels.data && dels.data.items) || []);

    // 延时触发一次 resize 确保在 DOM 完全挂载后尺寸精确
    setTimeout(() => {
      resizeCharts();
    }, 50);

  } catch (e) {
    // 错误已被 api.js 捕获并推至 toast
  }
}
