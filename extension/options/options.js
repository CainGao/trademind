// options/options.js — 设置页逻辑。
//
// 核心：服务端地址、登录凭据、采集开关的读写 + 连接测试。

import { getConfig, saveConfig } from '../lib/config.js';
import { healthCheck, login } from '../lib/api.js';

const $ = (id) => document.getElementById(id);

(async function init() {
  const cfg = await getConfig();
  $('serverUrl').value = cfg.serverUrl;
  $('username').value = cfg.username;
  $('password').value = cfg.password;
  $('autoCollect').checked = cfg.autoCollect !== false;
  $('autoReportBehavior').checked = cfg.autoReportBehavior !== false;
  $('reportIntervalMin').value = cfg.reportIntervalMin || 5;

  // 进入即检查连接
  doHealthCheck();
})();

// 测试连接
$('testConn').addEventListener('click', async () => {
  doHealthCheck();
});

async function doHealthCheck() {
  const url = $('serverUrl').value.trim();
  $('conn-result').textContent = '检测中…';
  const r = await healthCheck(url);
  const bar = $('connection-status');
  const text = bar.querySelector('.text');
  if (r.ok) {
    bar.className = 'status-bar ok';
    text.textContent = `✅ 已连接 ${r.server} v${r.version}`;
    $('conn-result').textContent = `服务端在线：${r.server} v${r.version}`;
    $('conn-result').className = 'result-line success';
  } else {
    bar.className = 'status-bar fail';
    text.textContent = `❌ 无法连接 ${url}`;
    $('conn-result').textContent = `连接失败：${r.message}`;
    $('conn-result').className = 'result-line error';
  }
}

// 登录测试
$('testLogin').addEventListener('click', async () => {
  const username = $('username').value.trim();
  const password = $('password').value;
  if (!username || !password) {
    $('login-result').textContent = '请填写用户名和密码';
    $('login-result').className = 'result-line error';
    return;
  }
  $('login-result').textContent = '登录中…';
  try {
    // 临时把 serverUrl 存一下，让 api.js 能读到
    await saveConfig({
      serverUrl: $('serverUrl').value.trim(),
      username, password,
      autoCollect: $('autoCollect').checked,
      autoReportBehavior: $('autoReportBehavior').checked,
      reportIntervalMin: parseInt($('reportIntervalMin').value) || 5,
    });
    const data = await login(username, password);
    $('login-result').textContent = `✅ 登录成功：${data.user?.username || username}（${data.user?.role || ''}）`;
    $('login-result').className = 'result-line success';
  } catch (e) {
    $('login-result').textContent = `❌ ${e.message}`;
    $('login-result').className = 'result-line error';
  }
});

// 保存
$('save').addEventListener('click', async () => {
  const cfg = {
    serverUrl: $('serverUrl').value.trim() || 'http://localhost:7789',
    username: $('username').value.trim(),
    password: $('password').value,
    autoCollect: $('autoCollect').checked,
    autoReportBehavior: $('autoReportBehavior').checked,
    reportIntervalMin: parseInt($('reportIntervalMin').value) || 5,
  };
  await saveConfig(cfg);
  $('save-result').textContent = '✅ 已保存';
  $('save-result').className = 'result-line success';
  setTimeout(() => { $('save-result').textContent = ''; }, 2000);
});

// 输入时清状态
['serverUrl', 'username', 'password'].forEach(id => {
  $(id).addEventListener('input', () => {
    $('connection-status').className = 'status-bar unknown';
    $('connection-status').querySelector('.text').textContent = '设置已修改，保存后生效';
  });
});
