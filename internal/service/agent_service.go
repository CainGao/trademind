// Package service — Agent 服务：在 AI 网关之上封装业务 Agent。
//
// Agent 与原始 Chat 的区别：
//   - Chat 是通用对话接口
//   - Agent 是「带业务上下文 + 固定目标」的 AI 调用，例如：
//     * AnalyzeProduct：把商品数据喂给 AI → 生成选品建议/风险提示/定价参考
//     * AnalyzeInquiry：解析询盘文本 → 提取国家/需求/阶段/成交概率
//     * GenerateReply：邮件自动回复建议
//
// 架构文档 §1.2: Agent 体系是「行为数据资产化」的产出端。
package service

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
	"github.com/shopspring/decimal"
)

// AgentService 业务 Agent。
type AgentService struct {
	ai           *AIService
	productRepo  *repository.ProductRepo
	supplierRepo *repository.SupplierRepo
	behaviorRepo *repository.BehaviorRepo
	agentRepo    *repository.AgentRepo
}

func NewAgentService(ai *AIService, pr *repository.ProductRepo, sr *repository.SupplierRepo, br *repository.BehaviorRepo, ar *repository.AgentRepo) *AgentService {
	return &AgentService{ai: ai, productRepo: pr, supplierRepo: sr, behaviorRepo: br, agentRepo: ar}
}

// ListRuns Agent 运行历史（Handler 调用）。
func (s *AgentService) ListRuns(page, pageSize int, agentType string) ([]models.AgentRun, int64, error) {
	return s.agentRepo.List(page, pageSize, agentType)
}

// GetRun 单次运行详情。
func (s *AgentService) GetRun(id uint) (*models.AgentRun, error) {
	var run models.AgentRun
	err := s.agentRepo.DB.First(&run, id).Error
	return &run, err
}

// ProductAnalysis 商品 AI 分析结果（结构化）。
type ProductAnalysis struct {
	Score              decimal.Decimal       `json:"score"`                // 综合评分 0-10
	SourcingAdvice     string                `json:"sourcing_advice"`      // 选品建议
	RiskWarnings       []string              `json:"risk_warnings"`        // 风险提示
	PriceAnalysis      string                `json:"price_analysis"`       // 定价参考
	MarketOutlook      string                `json:"market_outlook"`       // 市场前景
	RecommendedActions []string              `json:"recommended_actions"`  // 建议动作
	TargetMarkets      []string              `json:"target_markets"`       // 推荐目标市场
	B2BSuitability     string                `json:"b2b_suitability"`      // B2B 适配度
	B2CSuitability     string                `json:"b2c_suitability"`      // B2C 适配度
	Provider           AIProvider            `json:"provider"`
}

// AnalyzeProduct 分析商品（自动取商品数据 + 喂给 AI + 回写 ai_score/ai_analysis）。
//
// POST /api/agent/analyze-product?product_id=1
func (s *AgentService) AnalyzeProduct(productID uint, provider AIProvider) (*ProductAnalysis, error) {
	// 1. 取商品
	p, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, fmt.Errorf("商品不存在: %w", err)
	}

	// 2. 构造 prompt
	system := `你是资深外贸选品顾问和供应链专家，服务中国外贸/跨境电商公司。
分析给定商品数据，输出严格 JSON 格式（不要 markdown 代码块），字段：
{
  "score": 0-10 的数字（综合选品价值）,
  "sourcing_advice": "选品建议（100字内）",
  "risk_warnings": ["风险1", "风险2"],
  "price_analysis": "定价分析（100字内）",
  "market_outlook": "市场前景（100字内）",
  "recommended_actions": ["动作1", "动作2"],
  "target_markets": ["国家/地区1", "国家/地区2"],
  "b2b_suitability": "B2B 适配度评价（80字内）",
  "b2c_suitability": "B2C 适配度评价（80字内）"
}
只输出 JSON，不要任何额外文字。`

	user := buildProductPrompt(p)

	// 3. 调 AI
	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.3, // 分析类任务低温保证稳定
	})
	if err != nil {
		return nil, err
	}

	// 4. 解析 JSON（容错：去掉可能的 markdown 代码块）
	content := cleanJSON(resp.Content)
	var analysis ProductAnalysis
	if err := json.Unmarshal([]byte(content), &analysis); err != nil {
		// 解析失败：把原始内容塞进 SourcingAdvice，不让用户看到 500
		return &ProductAnalysis{
			SourcingAdvice: resp.Content,
			Provider:       resp.Provider,
		}, nil
	}
	analysis.Provider = resp.Provider

	// 5. 回写到商品表（ai_score + ai_analysis）
	scoreStr := "0"
	if analysis.Score.GreaterThan(decimal.Zero) {
		scoreStr = analysis.Score.String()
	}
	_ = s.productRepo.DB.Model(p).Updates(map[string]interface{}{
		"ai_score":    scoreStr,
		"ai_analysis": content,
	})

	return &analysis, nil
}

// buildProductPrompt 把商品数据格式化成 prompt。
func buildProductPrompt(p *models.Product) string {
	var b strings.Builder
	b.WriteString("请分析以下商品：\n\n")
	b.WriteString(fmt.Sprintf("商品名称: %s\n", p.Name))
	if p.Category != "" {
		b.WriteString(fmt.Sprintf("分类: %s\n", p.Category))
	}
	if p.Description != "" {
		b.WriteString(fmt.Sprintf("描述: %s\n", p.Description))
	}
	if p.PurchasePrice.GreaterThan(decimal.Zero) {
		b.WriteString(fmt.Sprintf("采购价: %s %s\n", p.PurchasePrice, p.PurchaseCurrency))
	}
	if p.Source != "" {
		b.WriteString(fmt.Sprintf("来源: %s (ID: %s)\n", p.Source, p.SourceID))
	}
	if p.B2BMOQ != nil {
		b.WriteString(fmt.Sprintf("起订量: %d\n", *p.B2BMOQ))
	}
	if p.WeightKG.GreaterThan(decimal.Zero) {
		b.WriteString(fmt.Sprintf("重量: %s kg\n", p.WeightKG))
	}
	if p.PackageSpec != "" {
		b.WriteString(fmt.Sprintf("包装: %s\n", p.PackageSpec))
	}
	if p.Scenarios != "" {
		b.WriteString(fmt.Sprintf("目标场景: %s\n", p.Scenarios))
	}
	return b.String()
}

// cleanJSON 去掉 AI 返回的 markdown 代码块包裹。
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// ============================================================================
// 选品 Agent + 采购 Agent（Week 5 新增）
// ============================================================================

// SelectionReport 选品 Agent 报告（结构化）。
type SelectionReport struct {
	HotCategories    []string `json:"hot_categories"`     // 热门类目推荐
	AvoidCategories  []string `json:"avoid_categories"`   // 避坑类目
	HighDemandProducts []string `json:"high_demand_products"` // 高需求商品方向
	MarketTrends     []string `json:"market_trends"`      // 市场趋势
	NextActions      []string `json:"next_actions"`       // 下一步行动建议
	Summary          string   `json:"summary"`            // 总结论（100字内）
}

// RunSelection 选品 Agent — 基于近 N 天行为数据 + 商品库，AI 输出选品建议。
//
// POST /api/agents/run?type=selection
// 定时任务调用：TriggerCron
func (s *AgentService) RunSelection(days int, provider AIProvider, triggeredBy models.TriggerType) (*SelectionReport, *models.AgentRun, error) {
	if days < 1 || days > 90 {
		days = 14
	}

	// 1. 取数据：Top 搜索词 + Top 浏览商品 + 商品库概览
	topKeywords, _ := s.behaviorRepo.TopKeywords(days, 15)
	byType, _ := s.behaviorRepo.StatsByType(days)
	prodList, _ := s.productRepo.List(repository.ListParams{Page: 1, PageSize: 30})

	// 2. 构造 prompt
	system := `你是资深外贸选品顾问，服务中国外贸/跨境电商公司。
基于员工最近一段时间的浏览/搜索行为数据 + 现有商品库，输出严格 JSON（不要 markdown）：
{
  "hot_categories": ["热门类目1", "热门类目2"],
  "avoid_categories": ["饱和/红海类目1"],
  "high_demand_products": ["高需求商品方向1", "方向2"],
  "market_trends": ["趋势1", "趋势2"],
  "next_actions": ["具体下一步1", "下一步2"],
  "summary": "总结论（100字内）"
}
只输出 JSON。`

	user := buildSelectionPrompt(days, topKeywords, byType, prodList.Items)

	// 3. 记录 Agent 运行开始
	run := &models.AgentRun{
		AgentType:   models.AgentSelection,
		TriggeredBy: triggeredBy,
		Input:       user,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now(),
	}
	_ = s.agentRepo.Create(run)

	// 4. 调 AI
	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.4,
	})
	now := time.Now()
	run.FinishedAt = &now
	if err != nil {
		run.Output = err.Error()
		run.Status = models.AgentRunFailed
		_ = s.agentRepo.Update(run)
		return nil, run, err
	}
	// 成功才访问 resp
	run.Output = resp.Content
	if resp.Usage != nil {
		tokens := resp.Usage.TotalTokens
		run.TokensUsed = &tokens
	}

	// 5. 解析 JSON
	content := cleanJSON(resp.Content)
	var report SelectionReport
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		// 解析失败：返回原始文本
		run.Status = models.AgentRunDone
		_ = s.agentRepo.Update(run)
		return &SelectionReport{Summary: resp.Content}, run, nil
	}
	run.Status = models.AgentRunDone
	_ = s.agentRepo.Update(run)
	return &report, run, nil
}

// buildSelectionPrompt 构造选品 Agent 的用户输入。
func buildSelectionPrompt(days int, topKeywords, byType []map[string]interface{}, products []models.Product) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# 最近 %d 天数据\n\n", days))

	b.WriteString("## Top 搜索关键词\n")
	for i, kw := range topKeywords {
		// 聚合查询别名为 cnt（behavior_repo.TopKeywords），缺失时兜底 0，绝不打印 <nil>
		b.WriteString(fmt.Sprintf("%d. %v（%s 次）\n", i+1, kw["keyword"], fmtCount(kw["cnt"])))
	}

	b.WriteString("\n## 行为分布\n")
	for _, t := range byType {
		// 聚合查询别名为 cnt（behavior_repo.StatsByType），缺失时兜底 0
		b.WriteString(fmt.Sprintf("- %v: %s 次\n", t["event_type"], fmtCount(t["cnt"])))
	}

	b.WriteString(fmt.Sprintf("\n## 现有商品库（共 %d 个样本）\n", len(products)))
	for i, p := range products {
		if i >= 10 {
			b.WriteString("...（更多省略）\n")
			break
		}
		b.WriteString(fmt.Sprintf("- %s | 类目: %s | 采购价: %s | AI 评分: %s\n",
			p.Name, p.Category, p.PurchasePrice, p.AIScore))
	}

	b.WriteString("\n请基于以上数据，给出选品建议。")
	return b.String()
}

// SourcingReport 采购 Agent 报告（结构化）。
type SourcingReport struct {
	UrgentPurchase   []string `json:"urgent_purchase"`    // 紧急采购（低库存高需求）
	SupplierRisks    []string `json:"supplier_risks"`     // 供应商风险提示
	CostOptimization []string `json:"cost_optimization"`  // 成本优化建议
	NegotiationTips  []string `json:"negotiation_tips"`   // 谈判建议
	Alternatives     []string `json:"alternatives"`       // 替代供应商/商品建议
	Summary          string   `json:"summary"`
}

// RunSourcing 采购 Agent — 基于商品 + 供应商，AI 输出采购建议。
//
// POST /api/agents/run?type=sourcing
func (s *AgentService) RunSourcing(provider AIProvider, triggeredBy models.TriggerType) (*SourcingReport, *models.AgentRun, error) {
	// 1. 取数据：商品 + 供应商
	prodList, _ := s.productRepo.List(repository.ListParams{Page: 1, PageSize: 20})
	supOv, _ := s.supplierRepo.Overview()

	// 2. 构造 prompt
	system := `你是资深供应链专家，服务中国外贸公司。
基于现有商品库 + 供应商数据，输出严格 JSON（不要 markdown）：
{
  "urgent_purchase": ["紧急采购商品1", "商品2"],
  "supplier_risks": ["供应商风险1", "风险2"],
  "cost_optimization": ["成本优化建议1"],
  "negotiation_tips": ["谈判建议1"],
  "alternatives": ["替代方案1"],
  "summary": "总结论（100字内）"
}
只输出 JSON。`

	user := buildSourcingPrompt(prodList.Items, supOv)

	// 3. 记录运行
	run := &models.AgentRun{
		AgentType:   models.AgentSourcing,
		TriggeredBy: triggeredBy,
		Input:       user,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now(),
	}
	_ = s.agentRepo.Create(run)

	// 4. 调 AI
	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.4,
	})
	now := time.Now()
	run.FinishedAt = &now
	if err != nil {
		run.Output = err.Error()
		run.Status = models.AgentRunFailed
		_ = s.agentRepo.Update(run)
		return nil, run, err
	}
	run.Output = resp.Content
	if resp.Usage != nil {
		tokens := resp.Usage.TotalTokens
		run.TokensUsed = &tokens
	}

	content := cleanJSON(resp.Content)
	var report SourcingReport
	if err := json.Unmarshal([]byte(content), &report); err != nil {
		run.Status = models.AgentRunDone
		_ = s.agentRepo.Update(run)
		return &SourcingReport{Summary: resp.Content}, run, nil
	}
	run.Status = models.AgentRunDone
	_ = s.agentRepo.Update(run)
	return &report, run, nil
}

// buildSourcingPrompt 构造采购 Agent 的用户输入。
func buildSourcingPrompt(products []models.Product, supOv *repository.SupplierOverview) string {
	var b strings.Builder
	b.WriteString("# 商品库（前 15 个）\n")
	for i, p := range products {
		if i >= 15 {
			break
		}
		// SupplierID 是 *uint，直接 %v 会打印内存地址（0x1400...），必须解引用
		b.WriteString(fmt.Sprintf("- %s | 采购价: %s %s | 供应商ID: %s | AI评分: %s\n",
			p.Name, p.PurchasePrice, p.PurchaseCurrency, fmtSupplierID(p.SupplierID), p.AIScore))
	}
	if supOv != nil {
		b.WriteString(fmt.Sprintf("\n# 供应商概览\n- 总数: %d\n- 高风险: %d\n- 中风险: %d\n- 低风险: %d\n",
			supOv.Total, supOv.RiskHigh, supOv.RiskMedium, supOv.RiskLow))
	}
	b.WriteString("\n请基于以上数据，给出采购建议。")
	return b.String()
}

// fmtCount 聚合查询计数值格式化：nil 值（key 不存在或 SQL NULL）兜底为 "0"，
// 绝不把 <nil> 打进 AI prompt（AI 会把它当异常数据）。
func fmtCount(v interface{}) string {
	if v == nil {
		return "0"
	}
	return fmt.Sprintf("%v", v)
}

// fmtSupplierID 主供应商 ID 格式化：*uint 直接 %v 会打印内存地址（0x140005e43b0），
// 既泄露内存布局又对 AI 毫无意义。nil → "无"，非 nil → 数字。
func fmtSupplierID(p *uint) string {
	if p == nil {
		return "无"
	}
	return fmt.Sprintf("%d", *p)
}

// CleanupZombieRuns 清理僵尸运行记录（进程启动时调用一次）。
// 进程重启/崩溃时正在执行的 Agent 运行会永久卡在 running 状态（goroutine 已消失），
// 启动时统一标记为 failed，避免 agent_runs 出现永不结束的僵尸记录（gotcha #30 的遗留面）。
func (s *AgentService) CleanupZombieRuns() {
	n, err := s.agentRepo.CleanupZombieRuns()
	if err != nil {
		log.Printf("[agent] 僵尸运行记录清理失败: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[agent] 已清理 %d 条僵尸运行记录（上次进程退出时中断的任务）", n)
	}
}
