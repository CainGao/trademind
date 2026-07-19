// background/service-worker.js — 后台服务工作者。
//
// 职责：
//   1. 接收 content script 的采集/行为消息，转发到服务端
//   2. 定时批量上报缓冲的行为数据
//   3. 安装时初始化右键菜单
//   4. 自动登录（用配置的用户名密码）

import { getConfig, getToken, clearToken } from '../lib/config.js';
import { login, collectProduct, reportBehaviorBatch, status } from '../lib/api.js';

// 行为缓冲队列（内存，service worker 重启会丢，但会持久化到 storage 兜底）
let behaviorQueue = [];

// ===== 启动时恢复队列 =====
chrome.runtime.onStartup.addListener(restoreQueue);
chrome.runtime.onInstalled.addListener(async (details) => {
  if (details.reason === 'install') {
    console.log('[TradeMind] 插件首次安装，打开设置页');
    chrome.runtime.openOptionsPage();
  }
  restoreQueue();
  setupContextMenus();
  startBehaviorFlushTimer();
});

function setupContextMenus() {
  chrome.contextMenus?.create({
    id: 'trademind-collect',
    title: '采集此商品到 TradeMind',
    contexts: ['page'],
  }, () => {}); // 忽略重复创建错误
}

chrome.contextMenus?.onClicked.addListener((info, tab) => {
  if (info.menuItemId === 'trademind-collect') {
    chrome.tabs.sendMessage(tab.id, { type: 'COLLECT_PAGE' }, (resp) => {
      if (chrome.runtime.lastError) {
        notify('采集失败', '当前页面不支持采集');
        return;
      }
      if (resp?.ok) {
        uploadCollected(resp.data);
      } else {
        notify('采集失败', resp?.message || '未识别到商品');
      }
    });
  }
});

// ===== 消息中枢 =====
chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  switch (msg.type) {
    case 'AUTO_COLLECT':
      handleAutoCollect(msg.data, sender.tab?.url);
      sendResponse({ ok: true });
      break;
    case 'COLLECT_NOW':
      // popup 触发的立即采集
      uploadCollected(msg.data)
        .then(r => sendResponse({ ok: true, result: r }))
        .catch(e => sendResponse({ ok: false, message: e.message }));
      break;
    case 'BEHAVIOR':
      behaviorQueue.push(msg.event);
      persistQueue();
      sendResponse({ ok: true });
      break;
    case 'STATUS_CHECK':
      checkStatus().then(sendResponse).catch(e => sendResponse({ ok: false, message: e.message }));
      break;
    case 'FLUSH_NOW':
      flushBehaviors().then(n => sendResponse({ ok: true, flushed: n })).catch(() => sendResponse({ ok: false }));
      break;
    case 'LOGOUT':
      clearToken().then(() => sendResponse({ ok: true }));
      break;
  }
  return true; // 保持消息通道异步
});

// ===== 自动采集处理（去重：同 URL 5 分钟内不重复）=====
const recentCollects = new Map(); // url → timestamp
async function handleAutoCollect(data, pageUrl) {
  if (!pageUrl) return;
  const now = Date.now();
  const last = recentCollects.get(pageUrl) || 0;
  if (now - last < 5 * 60 * 1000) return; // 5 分钟内已采过
  recentCollects.set(pageUrl, now);
  try {
    await ensureAuthed();
    await collectProduct(data);
    notify('✅ 自动采集成功', data.name);
  } catch (e) {
    if (e.code !== 'NOT_AUTHED') {
      console.warn('[TradeMind] 自动采集失败:', e.message);
    }
  }
}

// ===== 手动采集上传 =====
async function uploadCollected(data) {
  await ensureAuthed();
  const result = await collectProduct(data);
  notify(result.is_new_product ? '✅ 采集成功' : '🔄 已更新', data.name);
  return result;
}

// ===== 确保已登录（token 过期则自动重登）=====
async function ensureAuthed() {
  const tokenInfo = await getToken();
  if (tokenInfo?.token && tokenInfo.expiresAt > Date.now()) {
    return;
  }
  // token 不存在或过期，尝试用配置的用户名密码登录
  const cfg = await getConfig();
  if (!cfg.username || !cfg.password) {
    throw Object.assign(new Error('未配置登录凭据'), { code: 'NOT_AUTHED' });
  }
  await login(cfg.username, cfg.password);
}

// ===== 行为批量上报定时器 =====
let flushTimer = null;
function startBehaviorFlushTimer() {
  if (flushTimer) clearInterval(flushTimer);
  getConfig().then(cfg => {
    const interval = (cfg.reportIntervalMin || 5) * 60 * 1000;
    flushTimer = setInterval(() => {
      flushBehaviors().catch(() => {});
    }, interval);
  });
}

async function flushBehaviors() {
  if (behaviorQueue.length === 0) return 0;
  const events = behaviorQueue.splice(0, behaviorQueue.length);
  persistQueue();
  try {
    await ensureAuthed();
    await reportBehaviorBatch(events);
    return events.length;
  } catch (e) {
    // 上报失败：放回队列，下次重试
    behaviorQueue.unshift(...events);
    persistQueue();
    return 0;
  }
}

async function persistQueue() {
  await chrome.storage.local.set({ trademind_behavior_queue: behaviorQueue });
}

async function restoreQueue() {
  const r = await chrome.storage.local.get('trademind_behavior_queue');
  behaviorQueue = r.trademind_behavior_queue || [];
}

// ===== 连接状态检查 =====
async function checkStatus() {
  await ensureAuthed();
  const st = await status();
  return { ok: true, ...st };
}

// ===== 通知 =====
function notify(title, message) {
  chrome.notifications?.create({
    type: 'basic',
    iconUrl: chrome.runtime.getURL('icons/icon48.png'),
    title,
    message,
    priority: 0,
  });
}
