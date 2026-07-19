// Package models — 员工行为事件、AI 结果、Agent 运行记录、日报。
//
// 这些是"行为数据资产化"的核心（架构文档 §1.2），也是老板驾驶舱的数据源。
package models

import "time"

// BehaviorEvent 员工行为事件。员工浏览/搜索/收藏/采集的记录沉淀为企业资产。
//
// 索引设计（规范 V1.0 §1.5）:
//   - idx_behavior_user_time: (user_id, occurred_at DESC) — 查某员工最近行为
//   - idx_behavior_type_time: (event_type, occurred_at DESC) — 聚合某类行为趋势
type BehaviorEvent struct {
	BaseModel

	UserID     uint      `gorm:"not null;index:idx_behavior_user_time,priority:1" json:"user_id"`
	EventType  string    `gorm:"not null;type:text;index:idx_behavior_type_time,priority:1" json:"event_type"` // browse|search|collect|favorite|export|compare
	Source     DataSource `gorm:"not null;type:text" json:"source"`                                            // 1688|tiktok|temu|amazon
	TargetID   string    `gorm:"size:100" json:"target_id"`                                                  // 商品ID/供应商ID/关键词
	TargetMeta string    `gorm:"type:text" json:"target_meta"`                                               // JSON: 商品快照
	DurationSec *int     `json:"duration_sec,omitempty"`                                                     // 浏览时长（秒）

	OccurredAt time.Time `gorm:"not null;index:idx_behavior_user_time,priority:2,sort:desc;index:idx_behavior_type_time,priority:2,sort:desc" json:"occurred_at"`
}

// AIResult AI 分析结果。统一存储所有 AI 输出（商品分析/供应商评估/客户评分/日报）。
type AIResult struct {
	BaseModel

	Type       string  `gorm:"not null;type:text;index" json:"type"`      // product_analysis|supplier_assessment|customer_score|daily_report
	TargetID   *uint   `gorm:"index" json:"target_id,omitempty"`
	Score      *float64 `json:"score,omitempty"`
	Content    string  `gorm:"type:text" json:"content"`                  // JSON: 结构化分析结果
	ModelUsed  string  `gorm:"size:50" json:"model_used"`
	TokensUsed *int    `json:"tokens_used,omitempty"`
}

// AgentRun Agent 运行记录。所有 Agent 执行都记一行，便于追踪和成本核算。
type AgentRun struct {
	BaseModel

	AgentType   AgentType     `gorm:"not null;type:text;index" json:"agent_type"`
	TriggeredBy TriggerType   `gorm:"not null;type:text" json:"triggered_by"`
	Input       string        `gorm:"type:text" json:"input"`                       // JSON
	Output      string        `gorm:"type:text" json:"output"`                      // JSON
	ToolsCalled string        `gorm:"type:text" json:"tools_called"`                // JSON: ["query_products", "calc_profit"]
	TokensUsed  *int          `json:"tokens_used,omitempty"`
	Status      AgentRunStatus `gorm:"type:text;default:'running';index" json:"status"`
	StartedAt   time.Time     `gorm:"index" json:"started_at"`
	FinishedAt  *time.Time    `json:"finished_at,omitempty"`
}

// DailyReport 老板日报。AI 生成，可推送飞书。
type DailyReport struct {
	BaseModel

	ReportDate     time.Time `gorm:"not null;uniqueIndex:uk_daily_reports_date" json:"report_date"`
	Summary        string    `gorm:"type:text" json:"summary"`                  // JSON: 结构化数据（商品数/供应商数/机会数）
	AINarrative    string    `gorm:"type:text" json:"ai_narrative"`             // AI 生成的口语化叙述
	Opportunities  string    `gorm:"type:text" json:"opportunities"`            // JSON: 发现的机会
	KillScale      string    `gorm:"type:text" json:"kill_scale"`               // JSON: Kill/Scale 建议
	DeliveredToFeishu *bool  `gorm:"default:false" json:"delivered_to_feishu"`
}
