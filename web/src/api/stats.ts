// 统计 API（驾驶舱数据源）。
import client from "./client";
import type { SupplierOverview } from "./supplier";

export interface BehaviorOverview {
  total: number;
  last_7_days: number;
  last_30_days: number;
  browse_cnt: number;
  search_cnt: number;
  collect_cnt: number;
}

export interface TrendRow {
  date: string;
  total: number;
  browse: number;
  search: number;
  collect: number;
  favorite: number;
  compare: number;
  export: number;
}

export interface KeywordRow {
  keyword: string;
  cnt: number;
}

export interface Dashboard {
  behavior_overview: BehaviorOverview;
  supplier_overview: SupplierOverview;
  product_total: number;
  daily_trend: TrendRow[];
  top_keywords: KeywordRow[];
  stats_by_type: { event_type: string; cnt: number }[];
}

export const statsApi = {
  dashboard: (days = 14) =>
    client.get<unknown, Dashboard>("/stats/dashboard", { params: { days } }),

  trend: (days = 14) =>
    client.get<unknown, TrendRow[]>("/stats/behavior/trend", { params: { days } }),

  keywords: (days = 14, limit = 10) =>
    client.get<unknown, KeywordRow[]>("/stats/behavior/keywords", {
      params: { days, limit },
    }),
};
