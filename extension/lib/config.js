// lib/config.js — 配置管理（chrome.storage）
//
// 核心特性：上传目标完全可配置。
// 用户在设置页填写：
//   - 服务端地址（默认 http://localhost:7789）
//   - 用户名 + 密码（用于登录拿 JWT）
//   - 行为采集开关
//   - 采集后是否自动上报

const STORAGE_KEY = 'trademind_config';
const TOKEN_KEY = 'trademind_token';

// 默认配置
export const DEFAULT_CONFIG = {
  serverUrl: 'http://localhost:7789',
  username: '',
  password: '',
  autoCollect: true,        // 进入商品页自动采集
  autoReportBehavior: true, // 行为数据自动上报
  reportIntervalMin: 5,     // 行为批量上报间隔（分钟）
  collectPlatforms: ['1688', 'alibaba', 'amazon'],
};

/**
 * 读取配置（合并默认值）。
 * @returns {Promise<Object>}
 */
export async function getConfig() {
  const result = await chrome.storage.local.get(STORAGE_KEY);
  return { ...DEFAULT_CONFIG, ...(result[STORAGE_KEY] || {}) };
}

/**
 * 保存配置。
 * @param {Object} config
 */
export async function saveConfig(config) {
  const merged = { ...DEFAULT_CONFIG, ...config };
  await chrome.storage.local.set({ [STORAGE_KEY]: merged });
  return merged;
}

/**
 * 读取已缓存的 JWT token（登录后存）。
 */
export async function getToken() {
  const result = await chrome.storage.local.get(TOKEN_KEY);
  return result[TOKEN_KEY] || null;
}

/**
 * 缓存 JWT token + 过期时间。
 */
export async function saveToken(token, refreshToken, expiresAt) {
  await chrome.storage.local.set({
    [TOKEN_KEY]: { token, refreshToken, expiresAt }
  });
}

/**
 * 清除 token（退出登录）。
 */
export async function clearToken() {
  await chrome.storage.local.remove(TOKEN_KEY);
}

/**
 * 规范化服务端地址：去尾部斜杠。
 */
export function normalizeUrl(url) {
  if (!url) return DEFAULT_CONFIG.serverUrl;
  return url.replace(/\/+$/, '');
}
