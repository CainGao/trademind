// Agent 任务管理 API：选品/采购 Agent + 运行历史 + 定时配置。
import client from "./client";

// ===== 类型 =====
export interface AgentRun {
  id: number;
  agent_type: string;
  triggered_by: string;
  input?: string;
  output?: string;
  tokens_used?: number;
  status: string;
  started_at: string;
  finished_at?: string;
}

export interface SelectionReport {
  hot_categories?: string[];
  avoid_categories?: string[];
  high_demand_products?: string[];
  market_trends?: string[];
  next_actions?: string[];
  summary?: string;
}

export interface SourcingReport {
  urgent_purchase?: string[];
  supplier_risks?: string[];
  cost_optimization?: string[];
  negotiation_tips?: string[];
  alternatives?: string[];
  summary?: string;
}

export interface AgentScheduleItem {
  agent_type: string;
  cron: string;
  enabled: boolean;
}

// ===== API =====
export const agentApi = {
  /** 手动触发选品 Agent */
  runSelection(days = 14, provider = "") {
    return client.post<unknown, { report: SelectionReport; run: AgentRun }>(
      "/agents/run",
      null,
      { params: { type: "selection", days, provider } }
    );
  },

  /** 手动触发采购 Agent */
  runSourcing(provider = "") {
    return client.post<unknown, { report: SourcingReport; run: AgentRun }>(
      "/agents/run",
      null,
      { params: { type: "sourcing", provider } }
    );
  },

  /** Agent 运行历史 */
  listRuns(params: { page?: number; page_size?: number; type?: string }) {
    return client.get<unknown, { list: AgentRun[]; total: number }>(
      "/agents/runs",
      { params }
    );
  },

  /** 单次运行详情 */
  getRun(id: number) {
    return client.get<unknown, AgentRun>(`/agents/runs/${id}`);
  },

  /** 当前定时配置 */
  getSchedule() {
    return client.get<unknown, AgentScheduleItem[]>("/agents/schedule");
  },

  /** 更新定时 */
  updateSchedule(agent_type: string, cron: string) {
    return client.put<unknown, AgentScheduleItem[]>("/agents/schedule", {
      agent_type,
      cron,
    });
  },

  // ===== Week 6 专用 Agent =====

  /** 邮件分析 Agent */
  analyzeEmail(subject: string, content: string) {
    return client.post<unknown, { analysis: any; run: AgentRun }>(
      "/agents/analyze-email",
      { subject, content }
    );
  },

  /** 询盘分析 Agent */
  analyzeInquiry(inquiry_id: number) {
    return client.post<unknown, { analysis: any; run: AgentRun }>(
      "/agents/analyze-inquiry",
      null,
      { params: { inquiry_id } }
    );
  },

  /** 报价建议 Agent */
  adviseQuotation(inquiry_id: number, product_id: number) {
    return client.post<unknown, { advice: any; run: AgentRun }>(
      "/agents/advise-quotation",
      null,
      { params: { inquiry_id, product_id } }
    );
  },

  /** 上架优化 Agent */
  optimizeListing(product_id: number, platform = "amazon") {
    return client.post<unknown, { optimization: any; run: AgentRun }>(
      "/agents/optimize-listing",
      null,
      { params: { product_id, platform } }
    );
  },

  /** 评论分析 Agent */
  analyzeReviews(reviews: string) {
    return client.post<unknown, { analysis: any; run: AgentRun }>(
      "/agents/analyze-reviews",
      { reviews }
    );
  },
};
