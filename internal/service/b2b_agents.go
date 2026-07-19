// Package service — B2B 专用 Agent（邮件分析 / 询盘回复 / 报价建议）。
//
// 这三个 Agent 是外贸 B2B 业务最核心的 AI 增值点：
//   - 邮件 Agent：客户邮件 → AI 提取意图/产品/紧急度/情感 → 生成回复建议
//   - 询盘 Agent：询盘 → AI 分析需求/预算/阶段 → 生成跟进策略
//   - 报价 Agent：商品 + 询盘 → AI 生成报价区间 + 谈判底线
//
// 数据来源：用户手动粘贴邮件文本 / 从 inquiries 表读取
// 输出：结构化 JSON + 可直接复制的回复文本
package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CainGao/trademind/internal/models"
	"github.com/CainGao/trademind/internal/repository"
)

// ============================================================================
// 邮件分析 Agent
// ============================================================================

// EmailAnalysis 邮件分析结果。
type EmailAnalysis struct {
	Intent          string   `json:"intent"`            // 意图：inquiry|negotiation|complaint|order|followup|spam
	ProductsMentioned []string `json:"products_mentioned"` // 提到的产品
	Quantity        *int     `json:"quantity,omitempty"` // 数量
	TargetPrice     *float64 `json:"target_price,omitempty"` // 目标价
	Urgency         string   `json:"urgency"`            // high|medium|low
	Sentiment       string   `json:"sentiment"`          // positive|neutral|negative
	Country         string   `json:"country"`            // 推测国家
	Language        string   `json:"language"`           // 邮件语言
	SuggestedReply  string   `json:"suggested_reply"`    // 建议回复（可直接复制）
	KeyPoints       []string `json:"key_points"`         // 关键信息点
	Summary         string   `json:"summary"`            // 一句话总结
}

// AnalyzeEmail 邮件分析 Agent。
// 输入：邮件主题 + 正文。输出：结构化分析 + 建议回复。
//
// POST /api/agents/analyze-email
func (s *AgentService) AnalyzeEmail(subject, content string, provider AIProvider, triggeredBy models.TriggerType) (*EmailAnalysis, *models.AgentRun, error) {
	system := `你是资深外贸业务助理，服务中国外贸公司。
分析客户邮件，输出严格 JSON（不要 markdown）：
{
  "intent": "inquiry|negotiation|complaint|order|followup|spam",
  "products_mentioned": ["产品1", "产品2"],
  "quantity": 1000,
  "target_price": 5.5,
  "urgency": "high|medium|low",
  "sentiment": "positive|neutral|negative",
  "country": "推测国家",
  "language": "en|zh|es|ar|ru|fr|de|ja",
  "suggested_reply": "建议回复（150字内的英文邮件，可直接复制）",
  "key_points": ["关键点1", "关键点2"],
  "summary": "一句话总结（50字内）"
}
规则：
1. suggested_reply 必须是专业英文商务邮件
2. 如果是 spam（推销/钓鱼），suggested_reply 为空字符串
3. 只输出 JSON，不要任何额外文字。`

	user := fmt.Sprintf("邮件主题: %s\n\n邮件正文:\n%s", subject, content)

	run := &models.AgentRun{
		AgentType:   models.AgentEmail,
		TriggeredBy: triggeredBy,
		Input:       user,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now(),
	}
	_ = s.agentRepo.Create(run)

	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.3,
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

	content2 := cleanJSON(resp.Content)
	var analysis EmailAnalysis
	if err := json.Unmarshal([]byte(content2), &analysis); err != nil {
		run.Status = models.AgentRunDone
		_ = s.agentRepo.Update(run)
		return &EmailAnalysis{Summary: resp.Content, SuggestedReply: resp.Content}, run, nil
	}
	run.Status = models.AgentRunDone
	_ = s.agentRepo.Update(run)
	return &analysis, run, nil
}

// ============================================================================
// 询盘回复 Agent
// ============================================================================

// InquiryAnalysis 询盘分析结果。
type InquiryAnalysis struct {
	CustomerNeed      string   `json:"customer_need"`      // 客户真实需求
	BudgetLevel       string   `json:"budget_level"`       // high|medium|low|unknown
	DecisionStage     string   `json:"decision_stage"`     // early|comparing|ready_to_buy
	EstimatedQuantity *int     `json:"estimated_quantity"` // 预测量
	Competitors       []string `json:"competitors"`        // 可能的竞品
	WinProbability    *int     `json:"win_probability"`    // 成交概率 0-100
	SuggestedStrategy string   `json:"suggested_strategy"` // 跟进策略
	ReplyTemplate     string   `json:"reply_template"`     // 回复模板
	Risks             []string `json:"risks"`              // 风险点
	Summary           string   `json:"summary"`
}

// AnalyzeInquiry 询盘回复 Agent。
// 输入：inquiry_id（从数据库取）或直接传 productDesc + quantity。
//
// POST /api/agents/analyze-inquiry?inquiry_id=1
func (s *AgentService) AnalyzeInquiry(inquiryID uint, provider AIProvider, triggeredBy models.TriggerType) (*InquiryAnalysis, *models.AgentRun, error) {
	// 1. 取询盘
	var inquiry models.Inquiry
	if err := s.productRepo.DB.First(&inquiry, inquiryID).Error; err != nil {
		return nil, nil, fmt.Errorf("询盘不存在: %w", err)
	}

	system := `你是资深外贸业务专家，服务中国外贸公司。
分析询盘并输出跟进策略，严格 JSON：
{
  "customer_need": "客户真实需求（80字内）",
  "budget_level": "high|medium|low|unknown",
  "decision_stage": "early|comparing|ready_to_buy",
  "estimated_quantity": 5000,
  "competitors": ["可能竞品1"],
  "win_probability": 65,
  "suggested_strategy": "跟进策略（150字内）",
  "reply_template": "英文回复模板（200字内，可直接复制）",
  "risks": ["风险1"],
  "summary": "一句话总结"
}
只输出 JSON。`

	user := fmt.Sprintf(`询盘信息：
来源: %s
产品描述: %s
数量: %d
目标价: %s
目的港: %s
当前状态: %s`,
		inquiry.Source, inquiry.ProductDesc,
		derefInt(inquiry.Quantity),
		inquiry.TargetPrice.String(),
		inquiry.Destination, inquiry.Status)

	run := &models.AgentRun{
		AgentType:   models.AgentInquiry,
		TriggeredBy: triggeredBy,
		Input:       user,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now(),
	}
	_ = s.agentRepo.Create(run)

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

	content2 := cleanJSON(resp.Content)
	var analysis InquiryAnalysis
	if err := json.Unmarshal([]byte(content2), &analysis); err != nil {
		run.Status = models.AgentRunDone
		_ = s.agentRepo.Update(run)
		return &InquiryAnalysis{Summary: resp.Content}, run, nil
	}

	// 回写到 inquiry.AIAnalysis
	aiJSON, _ := json.Marshal(analysis)
	s.productRepo.DB.Model(&inquiry).Update("ai_analysis", string(aiJSON))

	run.Status = models.AgentRunDone
	_ = s.agentRepo.Update(run)
	return &analysis, run, nil
}

// ============================================================================
// 报价建议 Agent
// ============================================================================

// QuotationAdvice 报价建议。
type QuotationAdvice struct {
	RecommendedPrice  *float64 `json:"recommended_price"`  // 推荐报价
	PriceRangeLow     *float64 `json:"price_range_low"`    // 底价
	PriceRangeHigh    *float64 `json:"price_range_high"`   // 顶价
	MOQAdvice          *int     `json:"moq_advice"`         // 建议 MOQ
	ProfitMargin      *float64 `json:"profit_margin"`      // 预估利润率 %
	CompetitiveAnalysis string  `json:"competitive_analysis"` // 竞争分析
	NegotiationBottomLine string  `json:"negotiation_bottom_line"` // 谈判底线
	Tactics            []string `json:"tactics"`            // 报价战术
	Risks              []string `json:"risks"`
	Summary            string   `json:"summary"`
}

// AdviseQuotation 报价建议 Agent。
// 输入：inquiry_id + product_id（取商品成本）→ 输出报价策略。
//
// POST /api/agents/advise-quotation?inquiry_id=1&product_id=1
func (s *AgentService) AdviseQuotation(inquiryID, productID uint, provider AIProvider, triggeredBy models.TriggerType) (*QuotationAdvice, *models.AgentRun, error) {
	var inquiry models.Inquiry
	if err := s.productRepo.DB.First(&inquiry, inquiryID).Error; err != nil {
		return nil, nil, fmt.Errorf("询盘不存在: %w", err)
	}
	product, err := s.productRepo.FindByID(productID)
	if err != nil {
		return nil, nil, fmt.Errorf("商品不存在: %w", err)
	}

	system := `你是外贸报价专家。基于商品成本 + 询盘信息，给出报价建议。
严格 JSON：
{
  "recommended_price": 6.5,
  "price_range_low": 5.8,
  "price_range_high": 7.2,
  "moq_advice": 500,
  "profit_margin": 22.5,
  "competitive_analysis": "竞争分析（100字内）",
  "negotiation_bottom_line": "谈判底线（80字内）",
  "tactics": ["战术1", "战术2"],
  "risks": ["风险1"],
  "summary": "一句话总结"
}
只输出 JSON。`

	user := fmt.Sprintf(`商品信息：
名称: %s
类目: %s
采购价: %s %s
重量: %s kg
FOB 价: %s
MOQ: %d

询盘信息：
产品描述: %s
数量: %d
目标价: %s
目的港: %s`,
		product.Name, product.Category,
		product.PurchasePrice, product.PurchaseCurrency,
		product.WeightKG, product.B2BFOBPrice,
		derefInt(product.B2BMOQ),
		inquiry.ProductDesc,
		derefInt(inquiry.Quantity),
		inquiry.TargetPrice.String(),
		inquiry.Destination)

	run := &models.AgentRun{
		AgentType:   models.AgentQuotation,
		TriggeredBy: triggeredBy,
		Input:       user,
		Status:      models.AgentRunRunning,
		StartedAt:   time.Now(),
	}
	_ = s.agentRepo.Create(run)

	resp, err := s.ai.Chat(ChatRequest{
		Provider: provider,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.3,
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

	content2 := cleanJSON(resp.Content)
	var advice QuotationAdvice
	if err := json.Unmarshal([]byte(content2), &advice); err != nil {
		run.Status = models.AgentRunDone
		_ = s.agentRepo.Update(run)
		return &QuotationAdvice{Summary: resp.Content}, run, nil
	}
	run.Status = models.AgentRunDone
	_ = s.agentRepo.Update(run)
	return &advice, run, nil
}

// ============================================================================
// helpers
// ============================================================================

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// 防止 strings 未使用警告（未来可能用于扩展）
var _ = strings.TrimSpace
var _ = repository.ListParams{}
