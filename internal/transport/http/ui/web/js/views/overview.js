// 视图模块：系统概览 (Overview)
import { fetchAPI } from '../api.js';

export async function loadOverview() {
  try {
    const [accs, subs, evts, dels] = await Promise.all([
      fetchAPI('/admin/accounts'),
      fetchAPI('/admin/subscribers'),
      fetchAPI('/admin/events?limit=1'),
      fetchAPI('/admin/deliveries?status=failed&limit=200')
    ]);

    const elAccs = document.getElementById('kpiAccounts');
    const elSubs = document.getElementById('kpiSubscribers');
    const elEvts = document.getElementById('kpiEvents');
    const elDels = document.getElementById('kpiFailedDeliveries');

    if (accs && accs.data && elAccs) elAccs.innerText = (accs.data.items || []).length;
    if (subs && subs.data && elSubs) {
      const activeCount = (subs.data.items || []).filter(s => s.enabled).length;
      elSubs.innerText = activeCount;
    }
    if (evts && evts.data && elEvts) {
      elEvts.innerText = (evts.data.items || []).length > 0 ? '活跃' : '0';
    }
    if (dels && dels.data && elDels) {
      elDels.innerText = (dels.data.items || []).length;
    }
  } catch (e) {
    // 错误已被 api.js 捕获并推至 toast
  }
}
