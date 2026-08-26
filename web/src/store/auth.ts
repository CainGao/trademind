// 认证状态管理（Zustand）。
// 规范 §7.2: 统一 Token 管理 + 401 自动登出。

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { UserInfo } from "../types";

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: UserInfo | null;
  expiresAt: string | null;
  /** admin 仍在使用默认密码（登录时后端检测，gotcha #88） */
  mustChangePassword: boolean;

  setTokens: (data: {
    access_token: string;
    refresh_token: string;
    expires_at: string;
    user: UserInfo;
  }) => void;
  /** 改密成功后清除提醒标志 */
  clearMustChangePassword: () => void;
  logout: () => void;
  isAuthenticated: () => boolean;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,
      user: null,
      expiresAt: null,
      mustChangePassword: false,

      setTokens: (data) =>
        set({
          accessToken: data.access_token,
          refreshToken: data.refresh_token,
          expiresAt: data.expires_at,
          user: data.user,
          mustChangePassword: !!data.user.must_change_password,
        }),

      clearMustChangePassword: () => set({ mustChangePassword: false }),

      logout: () =>
        set({
          accessToken: null,
          refreshToken: null,
          expiresAt: null,
          user: null,
          mustChangePassword: false,
        }),

      isAuthenticated: () => {
        const { accessToken, expiresAt } = get();
        if (!accessToken || !expiresAt) return false;
        return new Date(expiresAt) > new Date();
      },
    }),
    { name: "trademind-auth" }, // localStorage 持久化
  ),
);
