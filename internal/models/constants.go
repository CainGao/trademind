// Package models — 枚举常量定义。
//
// 规范 V1.0 §1.4: 枚举值在 Go 中定义为字符串常量，禁止用数字。
package models

// ===== 用户相关 =====

// UserRole 用户角色。共享底座，B2B/B2C 场景下角色职责不同（见架构文档 §1.2）。
type UserRole string

const (
	RoleAdmin    UserRole = "admin"    // 全部权限 + 系统设置
	RoleBoss     UserRole = "boss"     // 驾驶舱 + Kill/Scale 决策
	RoleSourcing UserRole = "sourcing" // 供应链：商品 + 供应商
	RoleSales    UserRole = "sales"    // B2B: 客户 + 邮件 + 报价
	RoleOperator UserRole = "operator" // B2C: 店铺 + 上架 + 订单
	RoleStaff    UserRole = "staff"    // 个人数据 + 采集
)

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

// ===== 数据来源 =====

// DataSource 采集数据来源平台
type DataSource string

const (
	Source1688    DataSource = "1688"
	SourceAlibaba DataSource = "alibaba"
	SourceFactory DataSource = "factory"
	SourceManual  DataSource = "manual"
	SourceTikTok  DataSource = "tiktok"
	SourceTemu    DataSource = "temu"
	SourceAmazon  DataSource = "amazon"
)

// ===== 商品场景 =====

// ProductScenario 商品业务场景标记
type ProductScenario string

const (
	ScenarioB2B ProductScenario = "b2b"
	ScenarioB2C ProductScenario = "b2c"
)

// ===== AI 相关 =====

// AgentType Agent 类型（共享 + 专用）
type AgentType string

const (
	// 通用 Agent
	AgentSelection AgentType = "selection" // 选品
	AgentSourcing  AgentType = "sourcing"  // 采购
	AgentAnalysis  AgentType = "analysis"  // 分析
	AgentReport    AgentType = "report"    // 日报
	// B2B 专用
	AgentEmail     AgentType = "email"     // 邮件分析
	AgentInquiry   AgentType = "inquiry"   // 询盘回复
	AgentQuotation AgentType = "quotation" // 报价建议
	// B2C 专用
	AgentListing AgentType = "listing" // 上架优化
	AgentAd      AgentType = "ad"      // 广告投放
	AgentReview  AgentType = "review"  // 评论分析
)

// AgentRunStatus Agent 运行状态
type AgentRunStatus string

const (
	AgentRunRunning AgentRunStatus = "running"
	AgentRunDone    AgentRunStatus = "done"
	AgentRunFailed  AgentRunStatus = "failed"
)

// TriggerType Agent 触发方式
type TriggerType string

const (
	TriggerCron  TriggerType = "cron"  // 定时
	TriggerUser  TriggerType = "user"  // 手动
	TriggerEvent TriggerType = "event" // 事件链式
)

// ===== 供应商 =====

// RiskLevel 供应商风险等级
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// ===== 客户（B2B）=====

// CustomerStage 客户阶段
type CustomerStage string

const (
	CustomerStageLead        CustomerStage = "lead"        // 线索
	CustomerStageQuoting     CustomerStage = "quoting"     // 报价中
	CustomerStageNegotiating CustomerStage = "negotiating" // 谈判中
	CustomerStageWon         CustomerStage = "won"         // 成交
	CustomerStageLost        CustomerStage = "lost"        // 流失
)

// ===== 订单（B2C）=====

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderPending   OrderStatus = "pending"
	OrderPaid      OrderStatus = "paid"
	OrderShipped   OrderStatus = "shipped"
	OrderDelivered OrderStatus = "delivered"
	OrderCancelled OrderStatus = "cancelled"
	OrderRefunded  OrderStatus = "refunded"
)

// ===== 知识库（RAG）=====

// FileStatus 知识库文件解析状态
type FileStatus string

const (
	FileStatusProcessing FileStatus = "processing" // 解析+向量化中
	FileStatusReady      FileStatus = "ready"      // 可检索
	FileStatusFailed     FileStatus = "failed"     // 解析失败
)

// ===== 通用 =====

// ModuleName 模块包名
type ModuleName string

const (
	ModuleCommon ModuleName = "common" // 共享底座（必启用）
	ModuleB2B    ModuleName = "b2b"    // 外贸 Pack
	ModuleB2C    ModuleName = "b2c"    // 跨境 Pack
)
