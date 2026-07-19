// popup/popup.js — 弹窗逻辑。
//
// 打开时：检测连接状态 + 检测当前页是否可采集。
// 按钮点击：触发采集。

import { getConfig } from '../lib/config.js';

const $ = (id) => document.getElementById(id);

(async function init() {
  // 1. 连接状态
  chrome.runtime.sendMessage({ type: 'STATUS_CHECK' }, (resp) => {
    const badge = $('status-badge');
    if (chrome.runtime.lastError || !resp?.ok) {
      badge.className = 'badge offline';
      badge.textContent = '未连接';
      $('user-name').textContent = '—';
      return;
    }
    badge.className = 'badge online';
    badge.textContent = '已连接';
    $('user-name').textContent = `${resp.user}（${resp.role}）`;
  });

  // 2. 当前页信息
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (tab?.id && /^https?:\/\//.test(tab.url || '')) {
    chrome.tabs.sendMessage(tab.id, { type: 'GET_PAGE_INFO' }, (resp) => {
      if (chrome.runtime.lastError || !resp?.ok) {
        $('page-platform').textContent = '当前页不支持采集';
        return;
      }
      if (resp.canCollect) {
        $('page-platform').textContent = `✅ ${resp.platform} 商品页`;
        $('page-detail').classList.remove('hidden');
        $('page-name').textContent = resp.preview.name;
        $('page-price').textContent = resp.preview.price ? `¥${resp.preview.price}` : '';
        $('page-supplier').textContent = resp.preview.supplier || '';
        const btn = $('collect-btn');
        btn.disabled = false;
        btn.dataset.tabId = tab.id;
      } else {
        $('page-platform').textContent = `⚠️ ${resp.platform} 页面，未识别到商品`;
      }
    });
  } else {
    $('page-platform').textContent = '非网页';
  }

  // 3. 底部链接
  $('open-options').addEventListener('click', (e) => {
    e.preventDefault();
    chrome.runtime.openOptionsPage();
  });
  $('open-server').addEventListener('click', async (e) => {
    e.preventDefault();
    const cfg = await getConfig();
    chrome.tabs.create({ url: cfg.serverUrl });
  });
})();

// 采集按钮
$('collect-btn').addEventListener('click', async () => {
  const tabId = $('collect-btn').dataset.tabId;
  if (!tabId) return;
  const btn = $('collect-btn');
  btn.disabled = true;
  btn.textContent = '采集中…';

  chrome.tabs.sendMessage(parseInt(tabId), { type: 'COLLECT_PAGE' }, (resp) => {
    if (chrome.runtime.lastError || !resp?.ok) {
      btn.textContent = resp?.message || '❌ 采集失败';
      btn.className = 'btn-collect error';
      resetBtn(btn);
      return;
    }
    // 转发到 background 上传
    chrome.runtime.sendMessage({ type: 'COLLECT_NOW', data: resp.data }, (uploadResp) => {
      if (uploadResp?.ok) {
        const isNew = uploadResp.result?.is_new_product;
        btn.textContent = isNew ? '✅ 采集成功' : '🔄 已更新';
        btn.className = 'btn-collect success';
      } else {
        btn.textContent = uploadResp?.message || '❌ 上传失败（请检查设置）';
        btn.className = 'btn-collect error';
      }
      resetBtn(btn);
    });
  });
});

function resetBtn(btn) {
  setTimeout(() => {
    btn.textContent = '采集此商品';
    btn.disabled = false;
    btn.className = 'btn-collect';
  }, 2500);
}
