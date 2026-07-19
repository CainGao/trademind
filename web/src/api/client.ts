// Axios 客户端：统一拦截 Token、401 跳登录、错误冒泡。
//
// 规范 §3.2: 所有 API 返回 {code,message,data}
// 规范 §6.3: Authorization: Bearer <token>

import axios, { type AxiosError, type AxiosResponse } from "axios";
import { useAuthStore } from "../store/auth";
import type { ApiResponse } from "../types";

const client = axios.create({
  baseURL: "/api",
  timeout: 30000,
});

// 请求拦截：自动加 Bearer Token
client.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 响应拦截：拆包 ApiResponse + 401 处理
client.interceptors.response.use(
  (resp: AxiosResponse<ApiResponse>) => {
    const body = resp.data;
    // 业务错误（code !== 0）：抛 Error，UI 层处理
    if (body.code !== 0) {
      return Promise.reject(new ApiError(body.code, body.message));
    }
    // 成功：直接返回 data 字段
    return body.data as never;
  },
  (err: AxiosError<ApiResponse>) => {
    // HTTP 401: token 失效，跳登录
    if (err.response?.status === 401) {
      useAuthStore.getState().logout();
      window.location.hash = "#/login";
    }
    // 后端返回的业务错误
    const body = err.response?.data;
    if (body) {
      return Promise.reject(new ApiError(body.code, body.message));
    }
    // 网络错误
    return Promise.reject(new ApiError(-1, err.message || "网络错误"));
  },
);

// ApiError 业务错误，带错误码（规范 §3.4）
export class ApiError extends Error {
  code: number;
  constructor(code: number, message: string) {
    super(message);
    this.code = code;
    this.name = "ApiError";
  }
}

export default client;
