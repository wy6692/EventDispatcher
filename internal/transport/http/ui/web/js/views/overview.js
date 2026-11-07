// 系统概览视图模块
import { fetchAPI } from '../api.js';
import { copyText, pushToast } from '../utils.js';

let throughputChart = null;
let statusPieChart = null;
let tutorialInitialized = false;

// 调整 ECharts 图表渲染尺寸
export function resizeCharts() {
  if (throughputChart) {
    try { throughputChart.resize(); } catch (e) { /* ignore */ }
  }
  if (statusPieChart) {
    try { statusPieChart.resize(); } catch (e) { /* ignore */ }
  }
}

// 初始化图表实例
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

// 渲染 24 小时事件吞吐时序面积图
function renderThroughputChart(throughputData) {
  if (!throughputChart) return;

  const list = Array.isArray(throughputData) ? throughputData : [];
  const hours = list.map(item => item.hour || '');
  const counts = list.map(item => item.count || 0);
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
        interval: 3
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

// 渲染投递状态与健康度占比环形图
function renderDistributionChart(distData) {
  if (!statusPieChart) return;

  const dist = distData || {};
  const success = dist.success || 0;
  const failed = dist.failed || 0;
  const retrying = dist.retrying || 0;
  const pending = dist.pending || 0;

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

// 绑定接入指引交互与代码复制
function initTutorialInteractions() {
  if (tutorialInitialized) return;
  tutorialInitialized = true;

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

  window.addEventListener('resize', () => {
    resizeCharts();
  });
}

// 加载系统概览大盘数据
export async function loadOverview() {
  const elAccs = document.getElementById('kpiAccounts');
  const elSubs = document.getElementById('kpiSubscribers');
  const elEvts = document.getElementById('kpiEvents');
  const elDels = document.getElementById('kpiFailedDeliveries');

  initTutorialInteractions();

  const helperInput = document.getElementById('helperUrl');
  if (helperInput) {
    helperInput.value = `${window.location.origin}/w/{accountKey}`;
  }

  initCharts();

  try {
    const res = await fetchAPI('/admin/stats/overview');
    const stats = (res && res.data) || {};

    if (elAccs) elAccs.innerText = stats.accountsCount ?? 0;
    if (elSubs) elSubs.innerText = stats.activeSubscribers ?? 0;
    if (elEvts) elEvts.innerText = stats.eventsTotal ?? 0;
    if (elDels) elDels.innerText = stats.failedDeliveries ?? 0;

    [elAccs, elSubs, elEvts, elDels].forEach(el => {
      if (el) {
        el.style.animation = 'none';
        void el.offsetHeight;
        el.style.animation = 'fadeInRow 0.25s ease-out';
      }
    });

    renderThroughputChart(stats.hourlyThroughput || []);
    renderDistributionChart(stats.deliveryDistribution || {});

    setTimeout(() => {
      resizeCharts();
    }, 50);

  } catch (e) {
    // 接口异常已由 api.js 统一提示
  }
}
