// 老板日报 API：生成 + 列表 + 飞书推送。
import client from "./client";

export interface DailyReport {
  id: number;
  report_date: string;
  summary: string;       // JSON: 结构化数据
  ai_narrative: string;  // AI 口语化叙述
  opportunities: string; // JSON: ["机会1", "机会2"]
  kill_scale: string;    // JSON: [{action, target, reason}]
  delivered_to_feishu: boolean;
  created_at: string;
}

export interface FeishuConfig {
  webhook_url: string;
  secret?: string;
}

export const dailyReportApi = {
  /** 手动生成今日日报 */
  generate(autoDeliver = false) {
    return client.post<unknown, DailyReport>(
      "/daily-reports/generate",
      null,
      { params: { auto_deliver: autoDeliver } }
    );
  },

  /** 日报列表 */
  list(params: { page?: number; page_size?: number }) {
    return client.get<unknown, { list: DailyReport[]; total: number }>(
      "/daily-reports",
      { params }
    );
  },

  /** 今日日报 */
  today() {
    return client.get<unknown, DailyReport | null>("/daily-reports/today");
  },

  /** 单条详情 */
  getByID(id: number) {
    return client.get<unknown, DailyReport>(`/daily-reports/${id}`);
  },

  /** 推送到飞书 */
  deliverToFeishu(id: number) {
    return client.post<unknown, { delivered: boolean }>(
      `/daily-reports/${id}/deliver-feishu`
    );
  },

  /** 飞书 webhook 配置 */
  getFeishuConfig() {
    return client.get<unknown, FeishuConfig>("/daily-reports/feishu-config");
  },

  /** 更新飞书 webhook 配置 */
  updateFeishuConfig(data: FeishuConfig) {
    return client.put<unknown, { updated: boolean }>(
      "/daily-reports/feishu-config",
      data
    );
  },
};
