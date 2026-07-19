// lib/api.js — TradeMind 服务端 API 客户端。
//
// 所有请求走用户配置的服务端地址（设置页可改）。
// 鉴权：JWT Bearer token，通过用户名密码登录获取。

import { getConfig, getToken, saveToken, clearToken, normalizeUrl } from './config.js';

/**
 * 内部：统一请求封装。
 */
async function request(path, { method = 'GET', body, auth = true } = {}) {
  const config = await getConfig();
  const base = normalizeUrl(config.serverUrl);
  const headers = { 'Content-Type': 'application/json' };

  if (auth) {
    const tokenInfo = await getToken();
    if (tokenInfo?.token) {
      headers['Authorization'] = `Bearer ${tokenInfo.token}`;
    } else {
      throw new ApiError('未登录', 'NOT_AUTHED');
    }
  }

  let resp;
  try {
    resp = await fetch(`${base}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch (e) {
    throw new ApiError(`无法连接服务端：${config.serverUrl}（${e.message}）`, 'CONN_REFUSED');
  }

  const data = await resp.json().catch(() => ({}));

  // 401：token 过期，尝试清理
  if (resp.status === 401) {
    await clearToken();
    throw new ApiError(data.message || '登录已过期，请重新登录', 'AUTH_EXPIRED');
  }

  if (!resp.ok) {
    throw new ApiError(data.message || `请求失败 (${resp.status})`, 'HTTP_ERROR');
  }

  return data.data;
}

export class ApiError extends Error {
  constructor(message, code) {
    super(message);
    this.code = code;
  }
}

/**
 * 登录（用配置中的用户名密码换 token）。
 */
export async function login(username, password) {
  const config = await getConfig();
  const base = normalizeUrl(config.serverUrl);
  let resp;
  try {
    resp = await fetch(`${base}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    });
  } catch (e) {
    throw new ApiError(`无法连接服务端：${config.serverUrl}`, 'CONN_REFUSED');
  }
  const data = await resp.json().catch(() => ({}));
  if (!resp.ok || data.code !== 0) {
    throw new ApiError(data.message || '登录失败', 'LOGIN_FAILED');
  }
  const { access_token, refresh_token } = data.data;
  // access token 2h 过期
  const expiresAt = Date.now() + 2 * 60 * 60 * 1000 - 5 * 60 * 1000;
  await saveToken(access_token, refresh_token, expiresAt);
  return data.data;
}

/**
 * 检查连接状态。
 */
export async function status() {
  return request('/api/extension/status');
}

/**
 * 采集商品入库。
 */
export async function collectProduct(product) {
  return request('/api/extension/collect', { method: 'POST', body: product });
}

/**
 * 上报单条行为。
 */
export async function reportBehavior(event) {
  return request('/api/extension/behavior', { method: 'POST', body: event });
}

/**
 * 批量上报行为。
 */
export async function reportBehaviorBatch(events) {
  return request('/api/extension/behavior/batch', { method: 'POST', body: { events } });
}

/**
 * 健康检查（不需要登录）。
 */
export async function healthCheck(serverUrl) {
  const base = normalizeUrl(serverUrl);
  try {
    const resp = await fetch(`${base}/health`, { method: 'GET' });
    if (!resp.ok) return { ok: false, message: `HTTP ${resp.status}` };
    const data = await resp.json();
    return { ok: true, server: data.name, version: data.version };
  } catch (e) {
    return { ok: false, message: e.message };
  }
}
