// 审计日志 API：管理员查看敏感操作记录（登录/失败等）。
import client from "./client";

export interface AuditLogItem {
  id: number;
  user_id: number;
  username: string;
  action: string;
  resource: string;
  resource_id?: number;
  detail: string;
  ip: string;
  created_at: string;
}

export interface AuditLogPage {
  list: AuditLogItem[];
  total: number;
  page: number;
  size: number;
}

export interface AuditLogQuery {
  page?: number;
  page_size?: number;
  user_id?: number;
  action?: string;
  start_date?: string; // YYYY-MM-DD
  end_date?: string; // YYYY-MM-DD
}

export const auditApi = {
  /** 审计日志列表（分页 + 筛选，管理员） */
  list(params: AuditLogQuery) {
    return client.get<unknown, AuditLogPage>("/audit/logs", { params });
  },
};
