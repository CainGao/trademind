// 认证相关 API。
import client from "./client";
import type { LoginInput, RegisterInput, TokenPair, UserInfo } from "../types";

export const authApi = {
  login: (input: LoginInput) =>
    client.post<unknown, TokenPair>("/auth/login", input),

  register: (input: RegisterInput) =>
    client.post<unknown, UserInfo>("/auth/register", input),

  refresh: (refreshToken: string) =>
    client.post<unknown, TokenPair>("/auth/refresh", { refresh_token: refreshToken }),

  me: () => client.get<unknown, UserInfo>("/me"),
};
